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
at Qt                         ← morph
at #w                         ← Rocket render (morphs into its shadow root)
at HTMLElement.connectedCallback
at Qt
at #w
at HTMLElement.connectedCallback
…
```

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

The morph (`Qt`, the `outer`/`inner` patch path) keeps **module-global state**: the
pantry element `yt`, where id-matched nodes are parked while siblings are
reordered, and the id maps `z`, `Le`, `$e`, `ze`. Each call does:

```js
R.body.insertAdjacentElement("afterend", yt)   // start: (re)insert the pantry
…                                               // park/move id-matched nodes via yt
yt.remove()                                     // end
```

Rocket renders **synchronously in `connectedCallback`**, and that render is itself a
morph into the shadow root (`#w → Qt(this.#t, g, "inner")`). So:

1. The outer morph parks `<p id="a">…<x-item>` in `yt`.
2. Moving it back into the list connects `<x-item>`, and its `connectedCallback`
   starts a **nested** `Qt`.
3. The nested `Qt` re-inserts `yt` with `insertAdjacentElement`. That is a move of
   the pantry *with the parked `<x-item>` still inside*, which disconnects and
   reconnects it, so `connectedCallback` runs again and goes back to step 2.
   The recursion never ends.
4. When a nested call does finish, its `yt.remove()` detaches the pantry that the
   outer call is still using. The outer call's next `moveBefore` out of the
   detached pantry throws `HierarchyRequestError`. The nested call also clears the
   outer call's id maps.

## Fix that works

Make the pantry lifecycle re-entrant: only the outermost morph inserts and removes
it. In the minified v1.0.4 bundle, that is three edits:

```diff
-var Ze=B("ignore-morph")
+var Qd=0,Ze=B("ignore-morph")
-o.append(t),R.body.insertAdjacentElement("afterend",yt);
+o.append(t),Qd++||R.body.insertAdjacentElement("afterend",yt);
-ir(s,o,r==="outer"?e:null,e.nextSibling),yt.remove()},
+ir(s,o,r==="outer"?e:null,e.nextSibling),--Qd||yt.remove()},
```

A fuzz test ran 30 random patches, each reordering, adding and dropping 2–9 keyed
items that contain Rocket elements:

| Bundle | exceptions | frames with wrong result |
|---|---|---|
| v1.0.4 as shipped | 2444 | 1/30 |
| guard only the insert (`yt.isConnected \|\| …`) | 3 (`HierarchyRequestError`) | 3/30 |
| depth counter (diff above) | 0 | 0/30 |

A cleaner upstream fix would also scope the id maps per call, or defer Rocket's
first render out of `connectedCallback`, for example to a microtask. Nested
morphs would then never run inside an outer one.

## Workaround used in Starbase

Don't put `id`s on repeated or reordered elements that contain Rocket components
(the gallery cards). Without ids nothing is parked, and the morph goes positional.
