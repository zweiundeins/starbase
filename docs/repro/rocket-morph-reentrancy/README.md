# Morph re-entrancy with Rocket components

**Datastar v1.0.4 + Rocket beta.2** (`bundles/datastar-rocket.js`), Chromium 14x.

## Reproduce

```sh
cd docs/repro/rocket-morph-reentrancy
python3 -m http.server 8766    # @get needs http://, not file://
# open http://localhost:8766 and click "swap (Rocket)"
```

**Expected:** `#a` and `#b` swap places.
**Actual:** they swap, but the console shows `RangeError: Maximum call stack size exceeded` (three times).
It is visible only in devtools: with the stack exhausted, no `error` or `unhandledrejection`
listener can run, so the failure is silent to the page itself:

```
at morph                      ← plugins/watchers/patchElements.ts
at #render                    ← rocket/runtime.ts (morphs into its shadow root)
at HTMLElement.connectedCallback
at morph
at #render
at HTMLElement.connectedCallback
…
```

(Source names; the minified bundle shows `Qt` / `#w`.)

With more elements (reorders plus inserts and removals in one patch) there are also
`HierarchyRequestError: Failed to execute 'moveBefore' on 'Element': State-preserving
atomic move cannot be performed on nodes participating in an invalid hierarchy`, and
the morph aborts halfway, leaving the DOM out of sync with the patch.

**Minimal conditions** (each one is necessary):

| Variant | Result |
|---|---|
| keyed (`id`) siblings reordered, each containing a Rocket element | ❌ stack overflow |
| same, but with a plain custom element (the "plain" button) | ✅ |
| same Rocket content, no `id`s (positional morph) | ✅ |
| `Element.prototype.moveBefore` deleted (the `insertBefore` fallback) | ❌ same failure |

## Cause

Source: `library/src/plugins/watchers/patchElements.ts` and `library/src/rocket/runtime.ts`,
unchanged on `main` as of v1.0.4.

`morph()` keeps **module-level state**: the pantry `ctxPantry`, where id-matched nodes are
parked while siblings are reordered, and the id maps `ctxIdMap`, `ctxPersistentIds`,
`oldIdTagNameMap` and `duplicateIds`. Each call does:

```ts
export const morph = (oldElt, newContent, mode = 'outer') => {
  …
  DOCUMENT.body.insertAdjacentElement('afterend', ctxPantry)  // start: (re)insert the pantry
  …                                                           // fill the id maps, clear them
  morphChildren(…)                                            // park/move id-matched nodes via ctxPantry
  ctxPantry.remove()                                          // end
}
```

Rocket renders **synchronously in `connectedCallback`**: it runs `setup` and then
`this.#render({})`, and `#render` ends in `morph(this.#mountRoot!, fragment, 'inner')`. So:

1. The outer `morph` parks `<p id="a">…<x-item>` in `ctxPantry`.
2. Moving it back into the list connects `<x-item>`, and its `connectedCallback`
   starts a **nested** `morph`.
3. The nested `morph` re-inserts `ctxPantry` with `insertAdjacentElement`. That is a move
   of the pantry *with the parked `<x-item>` still inside*, which disconnects and
   reconnects it, so `connectedCallback` runs again and goes back to step 2.
   The recursion never ends.
4. When a nested call does finish, its `ctxPantry.remove()` detaches the pantry that the
   outer call is still using. The outer call's next `moveBefore` out of the detached
   pantry throws `HierarchyRequestError`. The nested call also clears the outer call's
   id maps (`ctxIdMap.clear()`, `ctxPersistentIds.clear()`).
5. The failure outlives the patch: the pantry ends up detached with a parked Rocket
   element still inside. The next `morph` anywhere on the page (even on plain elements)
   re-inserts it first, reconnects that element, overflows again and never applies its
   own patch. Click "swap (Rocket)", then "swap (plain)": the plain items don't swap.

## Fix that works

Make the pantry lifecycle re-entrant: only the outermost `morph` inserts and removes it.

```diff
 const ctxPantry = DOCUMENT.createElement('div')
 ctxPantry.hidden = true
+let morphDepth = 0

 export const morph = (…): void => {
   …
   const normalizedElt = DOCUMENT.createElement('div')
   normalizedElt.append(newContent)
-  DOCUMENT.body.insertAdjacentElement('afterend', ctxPantry)
+  if (!morphDepth++) DOCUMENT.body.insertAdjacentElement('afterend', ctxPantry)
+  try {
   …
   morphChildren(
     parent,
     normalizedElt,
     mode === 'outer' ? oldElt : null,
     oldElt.nextSibling,
   )
-
-  ctxPantry.remove()
+  } finally {
+    if (!--morphDepth) ctxPantry.remove()
+  }
 }
```

The `try`/`finally` keeps an exception from leaving the counter raised. The fuzz test below ran the
same counter patched into the minified v1.0.4 bundle (without the `try`).

A fuzz test ran 30 random patches, each reordering, adding and dropping 2–9 keyed
items that contain Rocket elements:

| Bundle | exceptions | frames with wrong result |
|---|---|---|
| v1.0.4 as shipped | 2444 | 1/30 |
| guard only the insert (`ctxPantry.isConnected \|\| …`) | 3 (`HierarchyRequestError`) | 3/30 |
| depth counter (diff above) | 0 | 0/30 |

A cleaner upstream fix would also scope the id maps per call, or defer Rocket's
first render out of `connectedCallback`, for example to a microtask. Nested
morphs would then never run inside an outer one.

## Workaround used in Starbase

Don't put `id`s on repeated or reordered elements that contain Rocket components
(the gallery cards). Without ids nothing is parked, and the morph goes positional.
