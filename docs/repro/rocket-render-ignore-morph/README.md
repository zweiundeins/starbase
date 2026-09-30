# A light-DOM component draws nothing inside data-ignore-morph

**Datastar v1.0.4 + Rocket beta.2** (`bundles/datastar-rocket.js`), Chromium 151. Draft issue: [`docs/upstream/09-render-inside-ignore-morph.md`](../../upstream/09-render-inside-ignore-morph.md).

## Reproduce

```sh
cd docs/repro/rocket-render-ignore-morph
python3 -m http.server 8803    # module import needs http://, not file://
# open http://localhost:8803; the result prints on the page
```

| Bundle | Inside `data-ignore-morph` | Outside | Inside, after a prop change |
|---|---|---|---|
| v1.0.4 | nothing | `one` | nothing |
| Starbase's build with `patches/rocket/0009` | `one` | `one` | `two` |

With the patch, a server patch that targets the ignored element or anything in it still changes nothing (checked with an outer morph of its parent and a selector patch of a child).

## Cause

A light component renders with `morph(this.#mountRoot, fragment, 'inner')` (runtime.ts:1186), and its mount root is the host. `morph()` returns at once when an ancestor of its target has `data-ignore-morph` (`oldElt.parentElement?.closest(...)`, patchElements.ts:301). A shadow root has no `parentElement`, so shadow components are not affected.

## Workaround

Build the markup in `setup` with the DOM API instead of `render` (nfsen-ng's toasts do), or use a shadow root.
