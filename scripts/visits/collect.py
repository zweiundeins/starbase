#!/usr/bin/env python3
"""Aggregate Caddy access logs for starbase.zweiundeins.gmbh into a JSON summary.

Runs on the web host (it needs to read /var/log/caddy), streams one JSON object
to stdout and never emits a raw IP: visitors are counted through a salted hash
that is thrown away with the process.
"""

import gzip
import hashlib
import json
import os
import re
import sys
from collections import Counter, defaultdict
from datetime import datetime, timezone
from glob import glob
from urllib.parse import urlparse

SITE = os.environ.get("VISITS_SITE", "starbase.zweiundeins.gmbh")
LEGACY = os.environ.get("VISITS_LEGACY_SITE", "rocket.libretto.ch")
LOG_GLOB = os.environ.get("VISITS_LOG_GLOB", "/var/log/caddy/starbase-access*.log*")
SESSION_GAP = 30 * 60  # seconds of inactivity that end a visit
# Clients that send no Sec-Fetch-Dest often send Accept: text/html for assets
# too, so the extension has the last word on what counts as a page.
ASSET_RE = re.compile(
    r"\.(css|m?js|json|map|svg|png|jpe?g|gif|webp|avif|ico|woff2?|ttf|txt|xml|wasm)$",
    re.I,
)

BOT_RE = re.compile(
    r"bot|crawl|spider|slurp|facebookexternalhit|headless|curl|wget|"
    r"python-requests|go-http-client|libwww|monitor|uptime|scrap|probe|"
    r"preview|validator|lighthouse|pingdom|semrush|ahrefs|dataprovider",
    re.I,
)
# Name the crawler for the report: the first group that matches wins.
BOT_NAME_RE = re.compile(
    r"(ClaudeBot|Claude-\w+|GPTBot|OAI-SearchBot|ChatGPT-User|PerplexityBot|"
    r"Googlebot|Google-Extended|bingbot|DuckDuckBot|Applebot|YandexBot|Baiduspider|"
    r"Twitterbot|Discordbot|Slackbot|TelegramBot|WhatsApp|LinkedInBot|facebookexternalhit|"
    r"SemrushBot|AhrefsBot|Bytespider|Amazonbot|curl|wget)",
    re.I,
)
# Shorteners and app referrers that hide the real source.
SOURCE_ALIASES = {
    "t.co": "x.com",
    "com.twitter.android": "x.com",
    "twitter.com": "x.com",
    "lnkd.in": "linkedin.com",
    "href.li": "unknown (href.li)",
    "out.reddit.com": "reddit.com",
}


def parse_ua(ua):
    """Coarse browser/platform buckets - enough to see who shows up, no more."""
    browser = "Other"
    for needle, name in (
        ("Edg/", "Edge"),
        ("OPR/", "Opera"),
        ("Firefox/", "Firefox"),
        ("Chrome/", "Chrome"),
        ("Safari/", "Safari"),
    ):
        if needle in ua:
            browser = name
            break
    if "Android" in ua:
        platform = "Android"
    elif "iPhone" in ua or "iPad" in ua:
        platform = "iOS"
    elif "Mac OS X" in ua or "Macintosh" in ua:
        platform = "macOS"
    elif "Windows" in ua:
        platform = "Windows"
    elif "Linux" in ua:
        platform = "Linux"
    else:
        platform = "Other"
    mobile = platform in ("Android", "iOS") or "Mobile" in ua
    return browser, platform, ("Mobile" if mobile else "Desktop")


def source_of(referer):
    """External referrer -> a source label, or None when it is internal/absent."""
    if not referer:
        return None
    host = urlparse(referer).netloc.split(":")[0].lower()
    if not host or host in (SITE, LEGACY):
        return None
    host = host[4:] if host.startswith("www.") else host
    return SOURCE_ALIASES.get(host, host)


def main():
    salt = os.urandom(16)
    files = sorted(glob(LOG_GLOB))
    if not files:
        sys.exit(f"no log files match {LOG_GLOB}")

    totals = Counter()
    hits = defaultdict(Counter)  # timeline buckets, keyed by hour
    pages = Counter()
    entry_pages = Counter()
    ref_sources = Counter()  # every hit carrying that external referrer
    ref_urls = Counter()
    bots = Counter()
    browsers, platforms, form_factors = Counter(), Counter(), Counter()
    status = Counter()
    visitors = set()
    sessions = {}  # visitor hash -> {"last": ts, "source": str|None}
    session_sources = Counter()  # visits attributed to their first external referrer
    first_ts = last_ts = None
    lines = 0

    for path in files:
        opener = gzip.open if path.endswith(".gz") else open
        with opener(path, "rt", errors="replace") as fh:
            for line in fh:
                lines += 1
                try:
                    rec = json.loads(line)
                except ValueError:
                    continue
                req = rec.get("request") or {}
                host = (req.get("host") or "").split(":")[0]
                ts = rec.get("ts")
                if not isinstance(ts, (int, float)):
                    continue
                if host == LEGACY:
                    totals["legacy_redirects"] += 1
                    continue
                if host != SITE:
                    continue

                totals["requests"] += 1
                first_ts = ts if first_ts is None else min(first_ts, ts)
                last_ts = ts if last_ts is None else max(last_ts, ts)

                headers = req.get("headers") or {}
                ua = (headers.get("User-Agent") or [""])[0]
                referer = (headers.get("Referer") or [""])[0]
                src = source_of(referer)
                code = rec.get("status") or 0
                status[str(code)] += 1

                if BOT_RE.search(ua):
                    totals["bot_requests"] += 1
                    name = BOT_NAME_RE.search(ua)
                    bots[name.group(1) if name else "unidentified automation"] += 1
                    continue

                if src:
                    totals["external_referred_hits"] += 1
                    ref_sources[src] += 1
                    ref_urls[referer[:200]] += 1

                # A page view is a top-level HTML document, not an asset or a
                # Datastar render stream. Older clients omit Sec-Fetch-Dest.
                dest = (headers.get("Sec-Fetch-Dest") or [""])[0]
                accept = (headers.get("Accept") or [""])[0]
                path_only = (req.get("uri") or "/").split("?")[0]
                is_doc = dest == "document" or (
                    not dest and "text/html" in accept and not ASSET_RE.search(path_only)
                )
                if not (is_doc and req.get("method") in ("GET", "") and code == 200):
                    continue

                totals["page_views"] += 1
                pages[path_only] += 1
                bucket = datetime.fromtimestamp(ts, timezone.utc).strftime("%Y-%m-%dT%H")
                hits[bucket]["page_views"] += 1

                who = hashlib.blake2b(
                    salt + (req.get("client_ip") or "") .encode() + ua.encode(),
                    digest_size=16,
                ).hexdigest()
                visitors.add(who)
                prev = sessions.get(who)
                if prev is None or ts - prev["last"] > SESSION_GAP:
                    totals["visits"] += 1
                    hits[bucket]["visits"] += 1
                    entry_pages[path_only] += 1
                    session_sources[src or "direct / none"] += 1
                    browser, platform, factor = parse_ua(ua)
                    browsers[browser] += 1
                    platforms[platform] += 1
                    form_factors[factor] += 1
                    sessions[who] = {"last": ts}
                else:
                    prev["last"] = ts

    def iso(ts):
        return datetime.fromtimestamp(ts, timezone.utc).isoformat(timespec="seconds")

    top = lambda counter, n, keys: [  # noqa: E731 - a local shape helper
        dict(zip(keys, (k, v))) for k, v in counter.most_common(n)
    ]

    json.dump(
        {
            "generated_at": datetime.now(timezone.utc).isoformat(timespec="seconds"),
            "site": SITE,
            "source": {
                "files": [os.path.basename(p) for p in files],
                "lines_read": lines,
                "session_gap_minutes": SESSION_GAP // 60,
            },
            "window": {
                "start": iso(first_ts) if first_ts else None,
                "end": iso(last_ts) if last_ts else None,
            },
            "totals": dict(totals),
            "unique_visitors": len(visitors),
            "timeline": [
                {"hour": h, "visits": c["visits"], "page_views": c["page_views"]}
                for h, c in sorted(hits.items())
            ],
            "referrer_sources": top(ref_sources, 25, ("source", "hits")),
            "referrer_urls": top(ref_urls, 25, ("url", "hits")),
            "visits_by_source": top(session_sources, 25, ("source", "visits")),
            "pages": top(pages, 25, ("path", "views")),
            "entry_pages": top(entry_pages, 15, ("path", "visits")),
            "bots": top(bots, 20, ("name", "requests")),
            "browsers": top(browsers, 10, ("name", "visits")),
            "platforms": top(platforms, 10, ("name", "visits")),
            "form_factors": top(form_factors, 5, ("name", "visits")),
            "status_codes": top(status, 12, ("code", "requests")),
        },
        sys.stdout,
        indent=1,
    )
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
