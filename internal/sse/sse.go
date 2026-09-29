// Package sse serves Server-Sent Events: hubs broadcast "something changed"
// pings, and Serve turns any number of hubs into one multiplexed event stream.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// heartbeat keeps idle connections alive through proxies.
const heartbeat = 25 * time.Second

// Hub fans change notifications out to subscribers. Pings carry no payload:
// each subscriber re-reads the current state itself, so every client can get
// a view tailored to it, and a burst of changes coalesces into one ping.
type Hub struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func NewHub() *Hub { return &Hub{subs: map[chan struct{}]struct{}{}} }

// Subscribe registers a subscriber; call the returned func to leave.
func (h *Hub) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

// Notify pings every subscriber without blocking; a subscriber that already
// has a pending ping keeps just that one.
func (h *Hub) Notify() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Stream is one named event type inside a connection: Data is re-read and
// sent once on connect and again after every ping from Hub.
type Stream struct {
	Event string
	Hub   *Hub
	Data  func(context.Context) (any, error)
}

// Serve holds the connection open and writes events until the client leaves
// or a Data call fails (the client's EventSource then reconnects).
func Serve(w http.ResponseWriter, r *http.Request, streams ...Stream) {
	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no") // disable proxy buffering
	// Lift the server's WriteTimeout: this response lives for as long as the tab.
	_ = rc.SetWriteDeadline(time.Time{})

	ctx := r.Context()
	pings := make(chan int, len(streams))
	for i, s := range streams {
		ch, leave := s.Hub.Subscribe()
		defer leave()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ch:
					select {
					case pings <- i:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	send := func(s Stream) bool {
		v, err := s.Data(ctx)
		if err != nil {
			return false
		}
		body, err := json.Marshal(v)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", s.Event, body); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	for _, s := range streams {
		if !send(s) {
			return
		}
	}

	tick := time.NewTicker(heartbeat)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case i := <-pings:
			if !send(streams[i]) {
				return
			}
		case <-tick.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		}
	}
}
