# Light children of a shadow component never bind when the definition was queued

**Datastar v1.0.4 + Rocket beta.2** (`bundles/datastar-rocket.js`), Chromium 151. Draft issue: [`docs/upstream/11-queued-definition-children.md`](../../upstream/11-queued-definition-children.md).

## Reproduce

```sh
cd docs/repro/rocket-queued-definition-children
python3 -m http.server 8805
# open http://localhost:8805; the result prints on the page
```

| Bundle | `data-text` child | `data-signals` + `data-on` + `data-text` child |
|---|---|---|
| v1.0.4 | `not bound` | `not bound`, the click does nothing |
| Starbase's build with `patches/rocket/0011` | `bound` | `clicked 1` |

Also checked with the patch: a nested child binds, a child's `data-init` runs once, and a child whose expression uses `$$` binds in the component's scope. A component defined after Datastar's first pass (a lazily imported module) applies its children only once, as before: their `data-init` still runs once.

## Cause

`rocket()` before Datastar's first pass queues the definition (runtime.ts:563 to :572) and marks each child of a matching shadow host with `data-ignore` (`markPendingRocketHosts`, runtime.ts:393 to :394), so the first pass cannot evaluate `$$` expressions in the page's scope. When the definition runs, `connectedCallback` removes the markers (runtime.ts:1208 to :1214), but nothing applies the children: the upgrade applies the host (`applyElement(this)`, runtime.ts:1524) and the shadow root (`apply(this.#mountRoot, true)`, runtime.ts:1531), and the engine's observer does nothing when `data-ignore` goes away. Attributes that Rocket rewrites for `$$` still bind, because the rewrite is an attribute change the observer applies; every other attribute stays inert.

## Workaround in nfsen-ng

Nothing in a shadow component's light children may carry a Datastar attribute unless the module is guaranteed to run after Datastar's first pass.
