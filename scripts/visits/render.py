#!/usr/bin/env python3
"""Render collect.py's JSON (on stdin) into a self-contained HTML report.

No dependencies and no network: the output is one file you can open anywhere.
Times are shown in VISITS_TZ (default Europe/Zurich); the JSON stays UTC.
"""

import html
import json
import os
import sys
from datetime import datetime, timezone
from zoneinfo import ZoneInfo

TZ = ZoneInfo(os.environ.get("VISITS_TZ", "Europe/Zurich"))
BAR_H, BAR_GAP, MAX_BAR = 22, 12, 24  # mark specs: thin bars, air between them


def esc(s):
    return html.escape(str(s), quote=True)


def fmt(n):
    return f"{n:,}".replace(",", " ")  # thin space, easier on the eye


def local(iso_str, pattern="%a %d %b %Y, %H:%M"):
    if not iso_str:
        return "-"
    return datetime.fromisoformat(iso_str).astimezone(TZ).strftime(pattern)


def nice_top(v):
    """Round the axis maximum up to a 1/2/5 x 10^n step, so ticks read cleanly."""
    if v <= 4:
        return max(v, 1)
    mag = 10 ** (len(str(int(v))) - 1)
    for mult in (1, 2, 2.5, 5, 10):
        if v <= mag * mult:
            return int(mag * mult)
    return int(mag * 10)


def hbar_chart(rows, label_key, value_key, unit, empty="No data yet.", compact=False):
    """Horizontal bars: value at the tip, 4px rounded data-end, no legend.

    compact is for the half-width sections - a narrow viewBox keeps the SVG from
    being scaled down (which would shrink the text with it).
    """
    if not rows:
        return f'<p class="empty">{esc(empty)}</p>'
    top = max(r[value_key] for r in rows) or 1
    if compact:
        label_w, plot_w, value_w, pad, clip = 160, 142, 46, 8, 23
    else:
        label_w, plot_w, value_w, pad, clip = 190, 520, 74, 12, 40
    width = label_w + plot_w + value_w
    height = len(rows) * (BAR_H + BAR_GAP)
    parts = [
        f'<svg class="chart" viewBox="0 0 {width} {height}" role="img" '
        f'width="{width}" height="{height}">'
    ]
    for i, r in enumerate(rows):
        y = i * (BAR_H + BAR_GAP)
        val = r[value_key]
        w = max(2, round(plot_w * val / top))
        share = val / sum(x[value_key] for x in rows) * 100
        label = str(r[label_key])
        tip = f"{esc(label)} - {fmt(val)} {unit} ({share:.0f}%)"
        # A label that won't fit is shortened here, never clipped by the SVG.
        shown = label if len(label) <= clip else "…" + label[-(clip - 1):]
        parts.append(
            f'<g class="bar-row" tabindex="0" data-tip="{tip}">'
            f'<rect x="0" y="{y}" width="{width}" height="{BAR_H}" class="hit"/>'
            f'<text x="{label_w - pad}" y="{y + BAR_H / 2}" class="lbl" '
            f'text-anchor="end" dominant-baseline="central">{esc(shown)}</text>'
            # Square at the baseline, rounded at the data-end.
            f'<path class="mark" d="M{label_w},{y} h{w - 4} a4,4 0 0 1 4,4 '
            f'v{BAR_H - 8} a4,4 0 0 1 -4,4 h-{w - 4} z"/>'
            f'<text x="{label_w + w + pad}" y="{y + BAR_H / 2}" class="val" '
            f'dominant-baseline="central">{fmt(val)}</text>'
            f"</g>"
        )
    parts.append("</svg>")
    return "".join(parts)


def column_chart(buckets):
    """Visits per hour. One series, so no legend - the heading names it."""
    if not buckets:
        return '<p class="empty">No page views in the window.</p>'
    top = nice_top(max(b["visits"] for b in buckets) or 1)
    n = len(buckets)
    plot_h, pad_l, pad_b, pad_t = 180, 44, 30, 16
    slot = max(28, min(72, 720 // n))
    bar_w = min(MAX_BAR, slot - 10)
    width = pad_l + n * slot + 16
    height = pad_t + plot_h + pad_b
    parts = [
        f'<svg class="chart" viewBox="0 0 {width} {height}" role="img" '
        f'width="{width}" height="{height}">'
    ]
    # Gridlines: hairline, solid, recessive - and they carry the values we don't label.
    steps = 4
    for i in range(steps + 1):
        v = round(top * i / steps)
        y = pad_t + plot_h - plot_h * i / steps
        parts.append(f'<line class="grid" x1="{pad_l}" y1="{y}" x2="{width - 8}" y2="{y}"/>')
        parts.append(
            f'<text class="tick" x="{pad_l - 10}" y="{y}" text-anchor="end" '
            f'dominant-baseline="central">{fmt(v)}</text>'
        )
    for i, b in enumerate(buckets):
        h = max(2, round(plot_h * b["visits"] / top))
        x = pad_l + i * slot + (slot - bar_w) / 2
        y = pad_t + plot_h - h
        t = datetime.strptime(b["hour"], "%Y-%m-%dT%H").replace(tzinfo=timezone.utc)
        t = t.astimezone(TZ)
        tip = (
            f'{t.strftime("%a %d %b, %H:%M")} - {fmt(b["visits"])} visits, '
            f'{fmt(b["page_views"])} page views'
        )
        parts.append(
            f'<g class="bar-row" tabindex="0" data-tip="{esc(tip)}">'
            f'<rect x="{x - 4}" y="{pad_t}" width="{bar_w + 8}" height="{plot_h}" class="hit"/>'
            f'<path class="mark" d="M{x},{pad_t + plot_h} v-{h - 4} a4,4 0 0 1 4,-4 '
            f'h{bar_w - 8} a4,4 0 0 1 4,4 v{h - 4} z"/>'
            f'<text class="tick" x="{x + bar_w / 2}" y="{height - pad_b + 16}" '
            f'text-anchor="middle">{t.strftime("%H")}</text>'
            f"</g>"
        )
    parts.append(
        f'<text class="tick axis" x="{pad_l}" y="{height - 4}">'
        f'hour of day ({TZ.key})</text></svg>'
    )
    return "".join(parts)


def table(rows, cols, keys):
    head = "".join(f"<th>{esc(c)}</th>" for c in cols)
    body = "".join(
        "<tr>" + "".join(f"<td>{esc(r[k]) if i == 0 else fmt(r[k])}</td>"
                         for i, k in enumerate(keys)) + "</tr>"
        for r in rows
    )
    return f"<table><thead><tr>{head}</tr></thead><tbody>{body}</tbody></table>"


def details_table(rows, cols, keys, summary="Show the data"):
    if not rows:
        return ""
    return (
        f"<details class='data-table'><summary>{esc(summary)}</summary>"
        f"{table(rows, cols, keys)}</details>"
    )


def tile(label, value, note=""):
    note = f'<div class="tile-note">{esc(note)}</div>' if note else ""
    return (
        f'<div class="tile"><div class="tile-label">{esc(label)}</div>'
        f'<div class="tile-value">{value}</div>{note}</div>'
    )


def main():
    d = json.load(sys.stdin)
    t = d["totals"]
    visits = t.get("visits", 0)
    by_source = d["visits_by_source"]
    referred = [r for r in by_source if r["source"] != "direct / none"]
    referred_visits = sum(r["visits"] for r in referred)
    lead = referred[0] if referred else None
    span_h = 0
    if d["window"]["start"] and d["window"]["end"]:
        start = datetime.fromisoformat(d["window"]["start"])
        end = datetime.fromisoformat(d["window"]["end"])
        span_h = (end - start).total_seconds() / 3600

    tiles = "".join([
        tile("Visits", fmt(visits), f"{d['source']['session_gap_minutes']}-min session gap"),
        tile("Unique visitors", fmt(d["unique_visitors"])),
        tile("Page views", fmt(t.get("page_views", 0)),
             f"{t.get('page_views', 0) / visits:.1f} per visit" if visits else ""),
        tile("Referred visits", fmt(referred_visits),
             f"{referred_visits / visits * 100:.0f}% of all visits" if visits else ""),
        tile("Requests", fmt(t.get("requests", 0)),
             f"{span_h:.1f} h of traffic" if span_h else ""),
        tile("Bot requests", fmt(t.get("bot_requests", 0)),
             "crawlers, excluded below"),
    ])

    # Once the window crosses midnight, "17:13 - 05:21" reads as one evening, so
    # the end gets its date back.
    same_day = local(d["window"]["start"], "%F") == local(d["window"]["end"], "%F")
    window_label = "{} – {}".format(
        local(d["window"]["start"]),
        local(d["window"]["end"], "%H:%M" if same_day else "%a %d %b %Y, %H:%M"),
    )

    # Form factor is a second cut of the same visits, so it rides as a note
    # rather than sharing a bar scale with the platforms.
    form_factor_note = ", ".join(
        f"{fmt(f['visits'])} {f['name'].lower()}" for f in d["form_factors"]
    ) or "No visits."

    headline = (
        f"{lead['source']} sent {fmt(lead['visits'])} of {fmt(visits)} visits "
        f"({lead['visits'] / visits * 100:.0f}%)"
        if lead and visits else "No referred visits in this window"
    )

    doc = f"""<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Visits - {esc(d['site'])}</title>
<style>
  :root {{
    color-scheme: light dark;
    --surface-0: #f6f5f2; --surface-1: #fcfcfb; --border: #e3e1db;
    --text-primary: #0b0b0b; --text-secondary: #52514e; --text-muted: #7a7873;
    --series-1: #2a78d6; --grid: #e8e6e0;
  }}
  /* Dark is its own selected set of steps, not a flip of the light one. */
  @media (prefers-color-scheme: dark) {{
    :root:where(:not([data-theme="light"])) {{
      --surface-0: #121211; --surface-1: #1a1a19; --border: #33332f;
      --text-primary: #ffffff; --text-secondary: #c3c2b7; --text-muted: #94938a;
      --series-1: #3987e5; --grid: #2b2b28;
    }}
  }}
  :root[data-theme="dark"] {{
    --surface-0: #121211; --surface-1: #1a1a19; --border: #33332f;
    --text-primary: #ffffff; --text-secondary: #c3c2b7; --text-muted: #94938a;
    --series-1: #3987e5; --grid: #2b2b28;
  }}
  * {{ box-sizing: border-box; }}
  body {{
    margin: 0; padding: 32px 24px 64px; background: var(--surface-0);
    color: var(--text-primary); font: 15px/1.55 ui-sans-serif, system-ui, -apple-system,
    "Segoe UI", Roboto, sans-serif; -webkit-font-smoothing: antialiased;
  }}
  main {{ max-width: 900px; margin: 0 auto; }}
  h1 {{ font-size: 26px; margin: 0 0 4px; letter-spacing: -0.01em; }}
  h2 {{ font-size: 17px; margin: 0 0 2px; letter-spacing: -0.005em; }}
  .sub {{ color: var(--text-secondary); margin: 0 0 6px; }}
  .meta {{ color: var(--text-muted); font-size: 13px; margin: 0; }}
  .headline {{
    margin: 24px 0 0; padding: 18px 20px; background: var(--surface-1);
    border: 1px solid var(--border); border-radius: 12px; font-size: 19px;
    line-height: 1.35; letter-spacing: -0.01em;
  }}
  .tiles {{
    display: grid; gap: 10px; margin: 20px 0 8px;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  }}
  .tile {{
    background: var(--surface-1); border: 1px solid var(--border);
    border-radius: 12px; padding: 14px 16px;
  }}
  .tile-label {{ font-size: 12px; color: var(--text-secondary);
    text-transform: uppercase; letter-spacing: 0.06em; }}
  .tile-value {{ font-size: 28px; font-weight: 600; letter-spacing: -0.02em;
    margin-top: 4px; font-variant-numeric: tabular-nums; }}
  .tile-note {{ font-size: 12px; color: var(--text-muted); margin-top: 2px; }}
  section {{
    background: var(--surface-1); border: 1px solid var(--border);
    border-radius: 12px; padding: 20px; margin-top: 16px;
  }}
  .chart {{ max-width: 100%; height: auto; display: block; margin-top: 14px; }}
  .mark {{ fill: var(--series-1); }}
  .hit {{ fill: transparent; }}
  .bar-row {{ cursor: default; outline: none; }}
  .bar-row:hover .mark, .bar-row:focus-visible .mark {{ filter: brightness(1.12); }}
  .bar-row:focus-visible .hit {{ fill: color-mix(in srgb, var(--series-1) 10%, transparent); }}
  .lbl {{ fill: var(--text-secondary); font-size: 13px; }}
  .val {{ fill: var(--text-primary); font-size: 13px; font-variant-numeric: tabular-nums; }}
  .tick {{ fill: var(--text-muted); font-size: 11px; font-variant-numeric: tabular-nums; }}
  .axis {{ font-size: 11px; }}
  .grid {{ stroke: var(--grid); stroke-width: 1; }}
  .empty {{ color: var(--text-muted); font-style: italic; }}
  .cols {{ display: grid; gap: 16px; grid-template-columns: 1fr; }}
  @media (min-width: 720px) {{ .cols {{ grid-template-columns: 1fr 1fr; }} }}
  table {{ border-collapse: collapse; width: 100%; font-size: 13px; margin-top: 10px; }}
  th, td {{ text-align: left; padding: 6px 10px 6px 0;
    border-bottom: 1px solid var(--border); }}
  th {{ color: var(--text-secondary); font-weight: 600; }}
  td:not(:first-child), th:not(:first-child) {{ text-align: right;
    font-variant-numeric: tabular-nums; }}
  td:first-child {{ word-break: break-all; }}
  details.data-table {{ margin-top: 14px; }}
  summary {{ cursor: pointer; color: var(--text-secondary); font-size: 13px; }}
  footer {{ color: var(--text-muted); font-size: 12.5px; margin-top: 24px;
    line-height: 1.6; }}
  code {{ font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px;
    background: var(--surface-0); padding: 1px 5px; border-radius: 4px; }}
  #tip {{
    position: fixed; pointer-events: none; opacity: 0; transition: opacity .1s;
    background: var(--text-primary); color: var(--surface-1); font-size: 12.5px;
    padding: 6px 9px; border-radius: 7px; max-width: 320px; z-index: 10;
  }}
</style></head>
<body><main>
  <h1>Visits to {esc(d['site'])}</h1>
  <p class="sub">{esc(window_label)} &middot; {TZ.key}</p>
  <p class="meta">Generated {local(d['generated_at'], '%d %b %Y, %H:%M')} from
     {fmt(d['source']['lines_read'])} Caddy log lines. Bots excluded from visits,
     visitors, page views and referrers.</p>

  <p class="headline">{esc(headline)}</p>

  <div class="tiles">{tiles}</div>

  <section>
    <h2>Where the visits came from</h2>
    <p class="sub">Visits by the external referrer that started them.</p>
    {hbar_chart(by_source, 'source', 'visits', 'visits')}
    {details_table(by_source, ['Source', 'Visits'], ['source', 'visits'])}
  </section>

  <section>
    <h2>Referring URLs</h2>
    <p class="sub">Every hit that carried an off-site Referer header
       ({fmt(t.get('external_referred_hits', 0))} in total).</p>
    {table(d['referrer_urls'], ['URL', 'Hits'], ['url', 'hits'])
     if d['referrer_urls'] else '<p class="empty">No external referrers.</p>'}
  </section>

  <section>
    <h2>Visits per hour</h2>
    {column_chart(d['timeline'])}
    {details_table(d['timeline'], ['Hour (UTC)', 'Visits', 'Page views'],
                   ['hour', 'visits', 'page_views'])}
  </section>

  <div class="cols">
    <section>
      <h2>Most-viewed pages</h2>
      {hbar_chart(d['pages'][:12], 'path', 'views', 'views', compact=True)}
      {details_table(d['pages'], ['Path', 'Views'], ['path', 'views'])}
    </section>
    <section>
      <h2>Landing pages</h2>
      <p class="sub">The first page of each visit.</p>
      {hbar_chart(d['entry_pages'][:12], 'path', 'visits', 'visits', compact=True)}
      {details_table(d['entry_pages'], ['Path', 'Visits'], ['path', 'visits'])}
    </section>
  </div>

  <div class="cols">
    <section>
      <h2>Platforms</h2>
      <p class="sub">{esc(form_factor_note)}</p>
      {hbar_chart(d['platforms'][:6], 'name', 'visits', 'visits', compact=True)}
    </section>
    <section>
      <h2>Browsers</h2>
      {hbar_chart(d['browsers'][:6], 'name', 'visits', 'visits', compact=True)}
    </section>
  </div>

  <section>
    <h2>Automated traffic</h2>
    <p class="sub">{fmt(t.get('bot_requests', 0))} requests, excluded from everything above.</p>
    {table(d['bots'], ['Crawler', 'Requests'], ['name', 'requests'])
     if d['bots'] else '<p class="empty">No bot traffic.</p>'}
  </section>

  <footer>
    <strong>How this is counted.</strong> A <em>page view</em> is a 200 response to a
    top-level HTML document (assets and Datastar render streams are not page views).
    A <em>visit</em> is one visitor's run of page views with no gap longer than
    {d['source']['session_gap_minutes']} minutes. A <em>visitor</em> is a salted hash of
    IP + user agent, discarded when the script exits &mdash; no IP reaches this file.
    Requests to the old <code>rocket.libretto.ch</code> domain
    ({fmt(t.get('legacy_redirects', 0))} redirects) are counted separately and are not
    part of the numbers above.<br>
    <strong>Source.</strong> {esc(', '.join(d['source']['files']))} on the web host.
    Re-run with <code>scripts/visits/report.sh</code>.
  </footer>
</main>
<div id="tip" role="status"></div>
<script>
  const tip = document.getElementById('tip');
  const show = (e) => {{
    const g = e.target.closest('[data-tip]');
    if (!g) return;
    tip.textContent = g.dataset.tip;
    tip.style.opacity = 1;
    const r = g.getBoundingClientRect();
    const x = (e.clientX ?? r.left + r.width / 2), y = (e.clientY ?? r.top);
    tip.style.left = Math.min(x + 14, innerWidth - tip.offsetWidth - 10) + 'px';
    tip.style.top = Math.max(8, y - tip.offsetHeight - 10) + 'px';
  }};
  addEventListener('mousemove', show);
  addEventListener('focusin', show);
  addEventListener('mouseout', (e) => {{
    if (!e.relatedTarget?.closest?.('[data-tip]')) tip.style.opacity = 0;
  }});
  addEventListener('focusout', () => tip.style.opacity = 0);
</script>
</body></html>
"""
    sys.stdout.write(doc)


if __name__ == "__main__":
    main()
