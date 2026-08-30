#!/usr/bin/env python3
"""Recover incident windows from the Telegram channel's own archive.

`checks` is dropped after ~35 days, and the durable `incidents` table only
started being written in August 2026, so outages between the two are gone from
the database. The channel (https://t.me/bgmmonitor) kept posting through that
window, and its public web view is scrapable, which makes it the only surviving
record of that period.

What comes out is coarser than the probe-derived walk `SyncIncidents` mirrors,
and deliberately so:

  * Only what the notifier announces exists here — authenticated checks on the
    main sites. Guest checks and the API subdomains never posted, so their
    windows in this period stay lost.
  * "Bangumi 活了！" carries both endpoints and is the primary source. Its
    times are group aggregates: the earliest start and latest recovery among
    the components a single message merged, so a component that joined late
    inherits the group's earlier start.
  * Alert debounce (two consecutive bad observations) puts every start one to
    two probe intervals after the failure the walk would have seen.
  * A degraded episode that recovered after the notifier went silent on
    degraded recoveries has no end in the channel at all. Those become
    --degraded-stub second stubs: both measured degraded windows in this period
    lasted 40s and 80s, so a floor is closer than dropping the event.

Output is an upsert against (domain, kind, start_ts), the same shape
cmd/importincidents emits, so re-applying it is harmless and a window the probe
data can still account for wins on every field that only grows.

  python3 scripts/telegram-incidents.py --since 2026-06-08 --until 2026-07-27 \
      --cache /tmp/tg -o dist/incidents-telegram.sql
"""

import argparse
import datetime as dt
import glob
import html
import json
import os
import re
import sys
import time
import urllib.request

CHANNEL = "bgmmonitor"
CST = dt.timezone(dt.timedelta(hours=8))
UA = ("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) "
      "Chrome/126 Safari/537.36")
KNOWN_DOMAINS = ("bangumi.tv", "bgm.tv", "chii.in", "next.bgm.tv/p1", "api.bgm.tv")
TAGS = ("#炸了", "#降级", "#活了", "#日报", "#公告")


def fetch(url):
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept-Language": "zh-CN,zh"})
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.read().decode("utf-8", "replace")


def scrape(cache_dir, floor_day):
    """Page the public web view backwards until it reaches before floor_day."""
    os.makedirs(cache_dir, exist_ok=True)
    before = None
    while True:
        url = f"https://t.me/s/{CHANNEL}" + (f"?before={before}" if before else "")
        page = fetch(url)
        ids = sorted(int(x) for x in set(re.findall(rf'data-post="{CHANNEL}/(\d+)"', page)))
        stamps = sorted(re.findall(r'<time datetime="([^"]+)"', page))
        if not ids:
            break
        open(os.path.join(cache_dir, f"p{ids[0]:06d}.html"), "w", encoding="utf-8").write(page)
        first = stamps[0][:10] if stamps else "?"
        print(f"  fetched {ids[0]}-{ids[-1]} from {first}", file=sys.stderr)
        if first <= floor_day or ids[0] <= 1:
            break
        before = ids[0]
        time.sleep(1.0)


def message_text(chunk):
    """The message body, without the reply preview that reuses the same class."""
    blocks = list(re.finditer(r'<div class="tgme_widget_message_text[^"]*"[^>]*>', chunk))
    if not blocks:
        return []
    tail = chunk[blocks[-1].end():]
    cut = re.search(r'<div class="tgme_widget_message_(?:footer|meta|reply)', tail)
    if cut:
        tail = tail[:cut.start()]
    tail = re.sub(r"<br\s*/?>", "\n", tail)
    # Inline markup sits inside a line; block tags end one.
    tail = re.sub(r"</?(?:b|i|u|s|em|strong|code|pre|a|span|tg-spoiler|del)\b[^>]*>", "", tail)
    tail = re.sub(r"<[^>]+>", "\n", tail)
    lines = [l.strip() for l in html.unescape(tail).split("\n") if l.strip()]
    # Reactions trail the body; the hashtag is always the last line of our own text.
    for i, l in enumerate(lines):
        if any(tag in l for tag in TAGS):
            return lines[:i + 1]
    return lines


def parse_messages(cache_dir):
    msgs = {}
    for fn in sorted(glob.glob(os.path.join(cache_dir, "p*.html"))):
        page = open(fn, encoding="utf-8").read()
        for chunk in re.split(r'(?=<div class="tgme_widget_message[ "])', page):
            m = re.search(rf'data-post="{CHANNEL}/(\d+)"', chunk)
            stamp = re.search(r'<time datetime="([^"]+)"', chunk)
            if not m or not stamp:
                continue
            posted = dt.datetime.fromisoformat(stamp.group(1)).astimezone(CST)
            msgs[int(m.group(1))] = {"id": int(m.group(1)), "posted": posted,
                                     "lines": message_text(chunk)}
    return [msgs[k] for k in sorted(msgs)]


CLOCK = r"(?:(\d{2})-(\d{2})\s+)?(\+1\s+)?(\d{2}):(\d{2}):(\d{2})"


def parse_duration(text):
    """`1 小时 47 分 20 秒` -> seconds. Returns None when absent."""
    m = re.search(r"(?:(\d+)\s*小时)?\s*(?:(\d+)\s*分)?\s*(?:(\d+)\s*秒)?", text)
    if not m or not any(m.groups()):
        return None
    h, mi, s = (int(g) if g else 0 for g in m.groups())
    return h * 3600 + mi * 60 + s


def resolve(clock, posted, ref=None):
    """Turn one message clock into an absolute CST time.

    The channel prints wall-clock without a year: bare `HH:MM:SS` when both ends
    of a window fall on one day, and either `MM-DD` (current notifier) or a `+1`
    marker (the version running in mid-2026) when they straddle midnight. Only
    the message's own date anchors any of it.

    Without `ref` the clock is the message's own moment — a recovery is posted
    within minutes of happening, so the candidate date at or just before the
    post time is the right one. With `ref` (the window's other end, already
    resolved) the clock is read as the day of `ref`, or the day before when it
    would otherwise sit after it.
    """
    mm, dd, _, hh, mi, ss = clock
    if mm and dd:
        year = (ref or posted).year
        cand = dt.datetime(year, int(mm), int(dd), int(hh), int(mi), int(ss), tzinfo=CST)
        if (cand - posted).days > 300:      # printed in December, posted in January
            cand = cand.replace(year=year - 1)
        return cand
    base = (ref or posted).date()
    cand = dt.datetime(base.year, base.month, base.day, int(hh), int(mi), int(ss), tzinfo=CST)
    if ref is None:
        if cand > posted + dt.timedelta(minutes=2):
            cand -= dt.timedelta(days=1)
    elif cand > ref:
        cand -= dt.timedelta(days=1)
    return cand


def parse_recovery(msg):
    """`Bangumi 活了！` -> (start, end, status, [domain]) or None."""
    body = " ".join(msg["lines"])
    if "#活了" not in body:
        return None
    ms = re.search(r"开始于\s*" + CLOCK, body)
    me = re.search(r"恢复于\s*" + CLOCK, body)
    if not ms or not me:
        return None
    # The recovery is the anchor: it is what the message was posted for. The
    # start then reads back from it, which is also what makes `恢复于+1` fall out
    # for free — a start clock later than the recovery belongs to the day before.
    end = resolve(me.groups(), msg["posted"])
    start = resolve(ms.groups(), msg["posted"], ref=end)
    status = "degraded" if "降级了" in body else "down"
    doms = re.search(r"([^\s]+)\s*恢复正常", body)
    domains = [d for d in re.split(r"、", doms.group(1))] if doms else []
    domains = [d for d in domains if d in KNOWN_DOMAINS]
    stated = parse_duration(body.split("大概")[1]) if "大概" in body else None
    return {"id": msg["id"], "start": start, "end": end, "status": status,
            "domains": domains, "stated": stated, "posted": msg["posted"]}


def parse_outage(msg):
    """`Bangumi 可能Boom了` / `有点不对劲` -> (opened, {domain: status}) or None."""
    body = " ".join(msg["lines"])
    if "#炸了" not in body and "#降级" not in body:
        return None
    at = re.search(r"在\s*" + CLOCK, body)
    if not at:
        return None
    opened = resolve(at.groups(), msg["posted"])
    comps = {}
    for dom, word in re.findall(r"(?:🔴|🟡)\s*(\S+?)\s*(中断|响应异常)", body):
        if dom in KNOWN_DOMAINS:
            comps[dom] = "down" if word == "中断" else "degraded"
    return {"id": msg["id"], "opened": opened, "components": comps, "posted": msg["posted"]}


def build(msgs, since, until, stub_s, warn):
    """Windows per (domain, kind=auth), newest source first."""
    rows = {}   # (domain, start_ts) -> row

    def add(domain, start, end, status, src):
        if not (since <= start < until):
            return False
        end = max(end, start + dt.timedelta(seconds=60))
        key = (domain, int(start.timestamp()))
        prev = rows.get(key)
        if prev is None or (prev["status"] != "down" and status == "down") or \
                int(end.timestamp()) > prev["end_ts"]:
            rows[key] = {"domain": domain, "start_ts": int(start.timestamp()),
                         "end_ts": max(int(end.timestamp()), prev["end_ts"] if prev else 0),
                         "status": "down" if (prev and prev["status"] == "down") or status == "down" else status,
                         "src": src}
        return True

    recoveries = [r for r in (parse_recovery(m) for m in msgs) if r]
    outages = [o for o in (parse_outage(m) for m in msgs) if o]

    for r in recoveries:
        if r["stated"] is not None:
            drift = abs((r["end"] - r["start"]).total_seconds() - r["stated"])
            if drift > 300:
                warn(f"msg {r['id']}: window {r['start']}..{r['end']} disagrees with "
                     f"stated {r['stated']}s by {int(drift)}s")
        if not r["domains"]:
            warn(f"msg {r['id']}: recovery names no known domain")
        for dom in r["domains"]:
            add(dom, r["start"], r["end"], r["status"], f"活了/{r['id']}")

    # An outage announcement whose recovery the channel never printed: the
    # notifier went silent on degraded recoveries partway through this period.
    for o in outages:
        for dom, status in o["components"].items():
            covered = any(dom in r["domains"] and
                          r["start"] - dt.timedelta(minutes=5) <= o["opened"] <= r["end"]
                          for r in recoveries)
            if covered:
                continue
            if add(dom, o["opened"], o["opened"] + dt.timedelta(seconds=stub_s), status,
                   f"{'炸了' if status == 'down' else '降级'}/{o['id']}") and status == "down":
                warn(f"msg {o['id']}: {dom} outage has no recovery message, stubbed at {stub_s}s")

    return [rows[k] for k in sorted(rows, key=lambda k: (-k[1], k[0]))]


def lit(s):
    return "'" + s.replace("'", "''") + "'"


def write_sql(rows, out, since, until):
    body = [
        "-- Incident windows recovered from https://t.me/bgmmonitor.",
        f"-- Window: {since:%Y-%m-%d %H:%M %Z} .. {until:%Y-%m-%d %H:%M %Z}",
        f"-- Rows: {len(rows)}  (generated by scripts/telegram-incidents.py)",
        "--",
        "-- Starts are debounce-delayed and group-aggregated; see the script's",
        "-- docstring. Only authenticated main-site checks are recoverable here.",
        "BEGIN;",
        "INSERT INTO incidents (domain, kind, start_ts, end_ts, status, peak_down, peak_total) VALUES",
    ]
    vals = [f"  ({lit(r['domain'])}, 'auth', {r['start_ts']}, {r['end_ts']}, {lit(r['status'])}, 0, 0)"
            for r in rows]
    body.append(",\n".join(vals))
    body += [
        "ON CONFLICT (domain, kind, start_ts) DO UPDATE SET",
        "  end_ts     = GREATEST(incidents.end_ts, excluded.end_ts),",
        "  status     = CASE WHEN excluded.status = 'down' OR incidents.status = 'down'",
        "                    THEN 'down' ELSE excluded.status END,",
        "  peak_down  = GREATEST(incidents.peak_down, excluded.peak_down),",
        "  peak_total = GREATEST(incidents.peak_total, excluded.peak_total);",
        "COMMIT;",
        "",
    ]
    out.write("\n".join(body))


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--since", required=True, help="first CST day to emit (YYYY-MM-DD)")
    ap.add_argument("--until", required=True, help="last CST day to emit, inclusive")
    ap.add_argument("--cache", default="/tmp/bgmmonitor", help="directory for scraped pages")
    ap.add_argument("--no-fetch", action="store_true", help="parse the cache, do not scrape")
    ap.add_argument("--degraded-stub", type=int, default=60, metavar="S",
                    help="length given to an announcement with no recovery (0 skips it)")
    ap.add_argument("-o", "--out", help="write SQL here instead of stdout")
    args = ap.parse_args()

    since = dt.datetime.fromisoformat(args.since).replace(tzinfo=CST)
    until = dt.datetime.fromisoformat(args.until).replace(tzinfo=CST) + dt.timedelta(days=1)

    if not args.no_fetch:
        # A page holds 20 messages; stop a few days past the window so a window
        # that opened before --since is still seen whole.
        scrape(args.cache, (since - dt.timedelta(days=3)).strftime("%Y-%m-%d"))
    msgs = parse_messages(args.cache)
    if not msgs:
        sys.exit(f"no messages in {args.cache}")
    print(f"{len(msgs)} messages, {msgs[0]['posted']:%Y-%m-%d} .. {msgs[-1]['posted']:%Y-%m-%d}",
          file=sys.stderr)

    warnings = []
    rows = build(msgs, since, until, args.degraded_stub, warnings.append)
    for w in warnings:
        print(f"  ! {w}", file=sys.stderr)

    per_domain, total = {}, 0
    for r in rows:
        d = per_domain.setdefault(r["domain"], [0, 0])
        d[0] += 1
        d[1] += r["end_ts"] - r["start_ts"]
        total += r["end_ts"] - r["start_ts"]
    for dom in sorted(per_domain):
        n, secs = per_domain[dom]
        print(f"  {dom:<16} {n:4d} windows  {secs / 3600:7.2f} h", file=sys.stderr)
    print(f"  {'total':<16} {len(rows):4d} windows  {total / 3600:7.2f} h", file=sys.stderr)

    if not rows:
        sys.exit("nothing to write")
    if args.out:
        with open(args.out, "w", encoding="utf-8") as fh:
            write_sql(rows, fh, since, until)
        print(f"wrote {args.out}", file=sys.stderr)
    else:
        write_sql(rows, sys.stdout, since, until)


if __name__ == "__main__":
    main()
