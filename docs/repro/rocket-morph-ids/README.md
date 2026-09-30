# Morph re-entrancy, the quiet case: moved ids come back new

Upstream: a follow-up to [starfederation/datastar#1209](https://github.com/starfederation/datastar/issues/1209). The loud case (stack overflow, `HierarchyRequestError`) is in [`../rocket-morph-reentrancy/`](../rocket-morph-reentrancy/).

**Datastar v1.0.4 + Rocket beta.2** (`bundles/datastar-rocket.js`), Chromium 151. Found in nfsen-ng, where a chart's canvas container came back empty after a server morph.

## Reproduce

```sh
cd docs/repro/rocket-morph-ids
python3 -m http.server 8767    # @get needs http://, not file://
# open http://localhost:8767, open the devtools console, click one button, reload before the other
```

The patch (`move.html`) inserts one element first, then moves `#a` and the `data-ignore-morph` box `#g` into the second section and `#b` into the first; `#c` stays where it is. The page tags every keyed element before the patch and reports which ones are still the same node afterwards.

| Bundle | Inserted first | `#a` `#b` moved | `#g` moved, its canvas | `#c` in place | Exception |
|---|---|---|---|---|---|
| v1.0.4 | Rocket element with a `render` | new nodes | new node, empty | same node | none |
| v1.0.4 | a tag nothing defines (`move-plain.html`) | same nodes | same node, canvas kept | same node | none |
| v1.0.4 + the depth counter proposed in `../rocket-morph-reentrancy/` | Rocket element with a `render` | new nodes | new node, empty | same node | none |
| v1.0.4 + `patches/rocket/0008` (alone, or in Starbase's build) | Rocket element with a `render` | same nodes | same node, canvas kept | same node | none |

Every run was checked in headless Chromium through the DevTools protocol, with the page's import pointed at a local copy of each bundle.

## Cause

The connected Rocket element renders synchronously in `connectedCallback`, and its render is a nested `morph()` (runtime.ts:1186). The nested call clears the module-level id maps the outer call still needs (`ctxPersistentIds.clear()` and `ctxIdMap.clear()`, patchElements.ts:327 and :343). Back in the outer morph, no remaining element counts as persistent, so every keyed element it reaches after the insertion is created from the patch instead of moved: state, focus, listeners and expandos are gone, and a moved `data-ignore-morph` element is recreated empty, since the patch holds no content for it. Nothing throws, so the console stays clean.

A counter that keeps the pantry attached (the fix proposed for the loud case) does not help here: the id maps are still cleared. Restoring the outer call's `ctxIdMap` and `ctxPersistentIds` when a nested call ends, as `patches/rocket/0008` does, fixes both cases.
