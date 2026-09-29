// Package jobs runs the aggregator's periodic background work.
package jobs

import (
	"context"
	"time"
)

// Every runs fn immediately and then every d until ctx is done.
func Every(ctx context.Context, d time.Duration, fn func(context.Context)) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		fn(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Daily runs fn at every midnight in loc, passing the start of the day that
// just ended.
func Daily(ctx context.Context, loc *time.Location, fn func(ctx context.Context, day time.Time)) {
	for {
		now := time.Now().In(loc)
		midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(midnight)):
		}
		fn(ctx, midnight.AddDate(0, 0, -1))
	}
}
