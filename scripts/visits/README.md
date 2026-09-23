# Visits report

A standalone traffic report for `starbase.zweiundeins.gmbh`, built from Caddy's
JSON access log. There is no analytics script on the site and no tracking cookie;
this reads what the reverse proxy already writes.

```sh
go tool task visits                 # -> tmp/visits-<date>.html
go tool task visits -- --keep-json  # keep the raw aggregate next to it
sh scripts/visits/report.sh --host libretto --out /tmp/report.html
```

- `collect.py` runs **on the web host** (`ssh <host> sudo python3 -`, because the
  logs are `caddy:caddy 0640`) and prints a JSON summary. Only that summary
  crosses the wire, not the log itself, which is tens of megabytes.
- `render.py` turns the JSON into one self-contained HTML file: no network, no
  dependencies, light and dark.

## What the numbers mean

- **Page view** - a 200 response to a top-level HTML document. Assets and
  Datastar render streams (POST) are not page views. A file opened directly in a
  tab (`Sec-Fetch-Dest: document`) is, which is why the odd `.svg` or `.js` shows
  up in the page lists - those are real navigations.
- **Visit** - one visitor's run of page views with no gap longer than 30 minutes.
- **Visitor** - a salted hash of IP + user agent. The salt is random per run and
  dies with the process, so the numbers are not comparable between runs and no IP
  ever reaches the report.
- **Referred visit** - attributed to the external referrer on the visit's first
  page view. `t.co` and the Twitter Android app are folded into `x.com`.
- Bots are matched on the user agent and excluded from every number above; they
  get their own table.

Requests to the old `rocket.libretto.ch` domain are counted separately (it only
redirects) and are never mixed into the totals.

## Coverage

Caddy rolls the log at 100 MB and keeps 5 rolls for 720 h, so the report reaches
back as far as those files go - `starbase-access*.log*`, gzipped rolls included.
It is not an all-time archive: the domain's log starts 2026-09-22, when
`starbase.zweiundeins.gmbh` began serving.
