// Live state for the status page. One EventSource on /api/events carries the
// status snapshot, the viewer count and reaction counts; a 30s poll is the
// safety net, so a dropped or zombie stream can never freeze the page.

import type { Overall, ReactionCount } from "./api-types";

export const REACTION_IDS = [44, 40, 15, 23, 83, 65, 41, 102, 49, 46, 51, 101];

export const live = $state({
  overall: null as Overall | null,
  error: "",
  viewers: null as number | null,
  viewersAt: 0,
  reactions: REACTION_IDS.map((id) => ({ emoji_id: id, count: 0, mine: false })) as ReactionCount[],
});

type BurstListener = (emojiID: number, n: number) => void;
const burstListeners = new Set<BurstListener>();

/** Calls fn(emoji, n) whenever other visitors add n reactions; returns an unsubscribe. */
export function onReactionBurst(fn: BurstListener): () => void {
  burstListeners.add(fn);
  return () => burstListeners.delete(fn);
}

/** An anonymous id so the server can flag "your" reactions. */
export const userID: string = (() => {
  let id = "";
  try {
    id = localStorage.getItem("rx_uid") ?? "";
  } catch {
    /* private mode */
  }
  if (id.length < 8) {
    id = crypto.randomUUID().replace(/-/g, "");
    try {
      localStorage.setItem("rx_uid", id);
    } catch {
      /* the id just lives for this visit */
    }
  }
  return id;
})();

function setReactions(next: ReactionCount[], animate: boolean) {
  if (animate) {
    const prev = new Map(live.reactions.map((r) => [r.emoji_id, r.count]));
    for (const r of next) {
      const delta = r.count - (prev.get(r.emoji_id) ?? 0);
      if (delta > 0) burstListeners.forEach((fn) => fn(r.emoji_id, delta));
    }
  }
  live.reactions = next;
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, { cache: "no-store", ...init });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function refreshStatus() {
  try {
    live.overall = await fetchJSON<Overall>("/api/status");
    live.error = "";
  } catch (err) {
    live.error = err instanceof Error ? err.message : String(err);
  }
}

let stream: EventSource | null = null;

async function refreshReactions() {
  try {
    setReactions(await fetchJSON<ReactionCount[]>("/api/reactions", { headers: { "X-User-ID": userID } }), false);
  } catch {
    /* keep the last counts */
  }
}

function connect(backoff = 1000) {
  const es = new EventSource("/api/events?uid=" + encodeURIComponent(userID));
  stream = es;
  let baseline = true; // the first reactions push is the baseline, not news
  const on = <T>(event: string, fn: (data: T) => void) =>
    es.addEventListener(event, (e) => {
      try {
        fn(JSON.parse((e as MessageEvent).data));
        backoff = 1000;
      } catch {
        /* ignore a malformed frame */
      }
    });
  on<Overall>("status", (o) => {
    live.overall = o;
    live.error = "";
  });
  on<{ viewers: number; updated_at: number }>("viewers", (v) => {
    live.viewers = v.viewers;
    live.viewersAt = v.updated_at;
  });
  on<ReactionCount[]>("reactions", (r) => {
    setReactions(r, !baseline);
    baseline = false;
  });
  es.onerror = () => {
    es.close();
    stream = null;
    setTimeout(() => connect(Math.min(backoff * 2, 30000)), backoff);
  };
}

/** Starts live updates; returns a function that stops them. */
export function startLive(): () => void {
  refreshStatus();
  refreshReactions();
  connect();
  const poll = setInterval(() => {
    refreshStatus();
    if (!stream) refreshReactions();
  }, 30000);
  const onVisible = () => {
    if (!document.hidden) refreshStatus();
  };
  document.addEventListener("visibilitychange", onVisible);
  return () => {
    clearInterval(poll);
    document.removeEventListener("visibilitychange", onVisible);
    stream?.close();
  };
}

/** Adds one reaction; resolves false when rate-limited. */
export async function react(id: number): Promise<boolean> {
  const cur = live.reactions.find((r) => r.emoji_id === id);
  if (cur) {
    cur.count++;
    cur.mine = true;
  }
  try {
    const res = await fetch("/api/reactions", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-User-ID": userID },
      body: JSON.stringify({ emoji_id: id }),
    });
    if (res.status === 429) {
      if (cur) cur.count = Math.max(0, cur.count - 1);
      return false;
    }
  } catch {
    /* the stream or the poll reconciles */
  }
  return true;
}
