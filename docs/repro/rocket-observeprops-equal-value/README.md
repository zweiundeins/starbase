# observeProps ignores attribute writes with an unchanged decoded value

**Datastar v1.0.4 + Rocket** (`bundles/datastar-rocket.js`), Chromium. `library/src/rocket/runtime.ts` on `main` has the same code. Not reported upstream as of 2026-09-23.

## Reproduce

```sh
cd docs/repro/rocket-observeprops-equal-value
python3 -m http.server 8802    # module import needs http://, not file://
# open http://localhost:8802
```

The page runs a self-test on load and prints the result; no devtools needed. It also works as a single CodePen (HTML panel).

**Expected:** writing an attribute that changes (`null` → `"false"`, `null` → `""`) runs the component's `observeProps` callback.
**Actual:** it is silent whenever the *decoded* value equals the prop's current value.

| Attribute write | observeProps |
|---|---|
| `open="true"` → `open="false"` | ✅ runs |
| no attribute → `open="true"` | ✅ runs |
| no attribute → `open="false"` | ❌ silent — the attribute changed, the decoded `false` did not |
| `open="false"` → `open="false"` again | silent, and correctly so: nothing changed |
| no attribute → `value=""` | ❌ silent — same, for a string |
| `value="abc"` → `value=""` | ✅ runs |

## Why it matters

A component often keeps local state that follows a server-owned attribute: a disclosure's `open`, a switch's `checked`, an input's `value`. The user changes the local state; later the server writes the attribute to reset it. If the element never had the attribute, the server's `open="false"` or `value=""` decodes to the prop's default — the value the prop already holds — and nothing runs, so the local state the user set survives the server's reset.

The page's A/B demo shows it: A (no attribute) is switched on by the user, the server writes `open="false"`, the attribute changes (the log shows the write), and A stays on. B, which started with `open="true"`, turns off.

## Cause

`attributeChangedCallback` decodes the new value and hands it to `#setProp`, which starts with:

```ts
#setProp(name, nextValue) {
  if (Object.is(this.#props[name], nextValue)) return
  …                                   // observers, then the optional render
}
```

That guard is right for property writes and for re-sending an identical attribute, but it also swallows a real attribute change whose decoded value happens to be equal.

## Suggested fix

Let an attribute that actually changed notify the observers, while keeping identical re-writes (and the render) quiet:

```diff
   attributeChangedCallback(name, oldValue, newValue) {
     …
-    this.#setProp(propName, newValue === null ? getCodecDefault(propDefs[propName]) : decodeCodec(propDefs[propName], newValue))
+    this.#setProp(propName, newValue === null ? getCodecDefault(propDefs[propName]) : decodeCodec(propDefs[propName], newValue), oldValue !== newValue)
   }

-  #setProp(name, nextValue) {
-    if (Object.is(this.#props[name], nextValue)) return
+  #setProp(name, nextValue, attributeChanged = false) {
+    const same = Object.is(this.#props[name], nextValue)
+    if (same && !attributeChanged) return
     …notify observers…
+    if (same) return                  // nothing to re-render
     …queue the render…
   }
```

Browsers call `attributeChangedCallback` for every `setAttribute`, identical values included, so `oldValue !== newValue` is what keeps a re-sent identical attribute silent.

## Workaround used in Starbase

Components that let the server own a boolean or clear a value watch the attribute itself with a `MutationObserver` (`attributeFilter: ['open']`) and apply their server-wins logic there: `sb-details`, `sb-dropdown`.
