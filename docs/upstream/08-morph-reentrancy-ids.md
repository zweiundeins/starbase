**Title:** #1209, the quiet case: a morph that inserts a Rocket element first recreates every keyed element it moves after it

**Filed:** not yet; a comment on https://github.com/starfederation/datastar/issues/1209

### Comment

A quiet case to check the #1209 fix against: nothing throws, but keyed elements lose their identity.

When a patch *inserts* a Rocket element with a `render` ahead of keyed elements it *moves*, the inserted element renders through a nested `morph()` (runtime.ts:1186), which clears the id maps the outer morph still needs (`ctxPersistentIds.clear()`, `ctxIdMap.clear()`, patchElements.ts:327 and :343). The outer morph then rebuilds every keyed element after the insertion from the patch instead of moving it: state, focus and listeners are lost, and a moved `data-ignore-morph` element comes back empty. nfsen-ng lost a chart's canvas this way.

Repro: `docs/repro/rocket-morph-ids/` in https://github.com/zweiundeins/starbase. The patch inserts `<x-note>` (a component with a `render`) first, then swaps `#a` (with the `data-ignore-morph` box `#g`) and `#b` between two sections.

| Bundle | `#a`, `#b`, `#g` after the patch |
|---|---|
| v1.0.4 | new nodes, the canvas in `#g` gone |
| v1.0.4, a plain tag inserted instead | same nodes |
| v1.0.4 + a depth counter that keeps the pantry attached | new nodes |
| the same, and the outer call's `ctxIdMap` and `ctxPersistentIds` restored when a nested call ends | same nodes |

So a fix that only makes the pantry re-entrant stops the stack overflow but not this. Starbase runs the last version until your release: [`patches/rocket/0008`](https://github.com/zweiundeins/starbase/blob/main/patches/rocket/0008-Morph-a-morph-that-starts-inside-another-keeps-the-o.patch).

*Claude was used to draft this comment, find the case and check it in Chrome.*
