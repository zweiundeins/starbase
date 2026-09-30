# Removed Rocket elements are never collected

**Datastar v1.0.4 + Rocket beta.2** (`bundles/datastar-rocket.js`), Chromium 151. Draft issue: [`docs/upstream/10-removed-elements-stay-observed.md`](../../upstream/10-removed-elements-stay-observed.md).

## Reproduce

```sh
cd docs/repro/rocket-removed-elements
python3 -m http.server 8804
# open http://localhost:8804, collect garbage in DevTools (Memory panel), press check
```

Measured through the DevTools protocol (`HeapProfiler.collectGarbage` three times, `Memory.getDOMCounters`) with 200 elements like the page's, each holding a `<div>` of 40 `<span>`s as light children, in shadow and in light mode:

| Bundle | Shadow elements alive | DOM nodes kept per shadow element | Light elements alive | DOM nodes kept per light element | Plain custom elements alive |
|---|---|---|---|---|---|
| v1.0.4 | 200 of 200 | 46 | 200 of 200 | 44 | 0 of 200 |
| Starbase's build with `patches/rocket/0010` | 0 of 200 | 0 | 0 of 200 | 0 | 0 of 200 |

With the patch, an element removed and put back later still works: its `$$` counter counts, and a node appended into its shadow root after the reconnect binds.

## Cause

`connectedCallback` ends with `apply(this.#mountRoot, true)` (runtime.ts:1531), which adds the mount root (the shadow root, or the host in light mode) to the engine's module-level `observedRoots` set (engine.ts:47, :255). Nothing ever deletes from it, so every Rocket element that was ever connected stays reachable, with its whole subtree.

## Workaround in nfsen-ng

After a removal (a microtask later, so a move is not mistaken for one), empty the element, its shadow root and its attributes, and detach it from its old parent: what stays is the element itself, 1.7 DOM nodes per removed element on nfsen-ng's pages instead of the whole subtree.
