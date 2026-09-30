# Reordering n children costs O(n²) in Rocket's pending-host observer

**Datastar v1.0.4 + Rocket beta.2** (`bundles/datastar-rocket.js`), Chromium 151. Draft issue: [`docs/upstream/12-pending-host-observer-rescan.md`](../../upstream/12-pending-host-observer-rescan.md).

## Reproduce

```sh
cd docs/repro/rocket-observer-rescan
python3 -m http.server 8806
# open http://localhost:8806; the timings print on the page
```

Rows of 8 cells, reordered with `replaceChildren(...[...rows.rows].reverse())`, headless Chromium 151, the range of three runs:

| Bundle | 250 rows | 500 rows | 1000 rows |
|---|---|---|---|
| v1.0.4 | 205 to 209 ms | 735 to 770 ms | 2862 to 2880 ms |
| Starbase's build with `patches/rocket/0012` | 24 to 27 ms | 43 to 44 ms | 22 to 85 ms |

Filling the body with new rows (one record) takes 6 to 45 ms on both.

## Cause

The bundle starts a document-wide `MutationObserver` for hosts whose definition has not run yet (`observePendingRocketHosts`, runtime.ts:402 to :437). For every `childList` record it calls `markPendingRocketHosts(record.target)` (runtime.ts:419 to :421), which walks the target's whole subtree (`querySelectorAll('*')`). `replaceChildren(...nodes)` over nodes already in the parent queues one removal record per node before the insertion, so the parent's subtree is walked once per row.

## Workaround in nfsen-ng

nfsen-ng's result table empties the body first and then appends the new page, which gives one record per call: from 100 to 250 rows per page took 633 ms of script before that.
