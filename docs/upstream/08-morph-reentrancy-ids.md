**Title:** #1209, the quiet case: a morph that inserts a Rocket element first recreates every keyed element it moves after it

**Filed:** not yet; a comment on https://github.com/starfederation/datastar/issues/1209

### Comment

A follow-up to #1209 with a case that throws nothing, so the fix for the next release can be checked against it too.

A patch that *inserts* one Rocket element with a `render` ahead of keyed elements it *moves* loses those elements' identity. The inserted element renders in `connectedCallback`, and its render is a nested `morph()` (runtime.ts:1186) that clears the module-level id maps the outer morph still needs (`ctxPersistentIds.clear()`, `ctxIdMap.clear()`, patchElements.ts:327 and :343). The outer morph then treats every keyed element after the insertion as new: it builds it from the patch instead of moving it, so state, focus, listeners and expandos are gone, and a moved `data-ignore-morph` element comes back empty, because the patch has no content for it. An element that stays in place keeps its identity. No exception, nothing in the console. nfsen-ng (a NetFlow viewer on Datastar) lost a chart's canvas this way whenever a server morph inserted a component ahead of the chart.

### Reproduce

`docs/repro/rocket-morph-ids/` in https://github.com/zweiundeins/starbase (served over http, as `@get` needs):

```html
<!-- before: #a and the box #g in #s1, #b in #s2 -->
<div id="root">
  <section id="s1"><p id="a">A</p><div id="g" data-ignore-morph><canvas></canvas></div></section>
  <section id="s2"><p id="b">B</p></section>
  <p id="c">C</p>
</div>
<!-- the patch: a Rocket element with a render first, then #b and #a (with #g) swap sections -->
<div id="root"><x-note></x-note><section id="s1"><p id="b">B</p></section><section id="s2"><p id="a">A</p><div id="g" data-ignore-morph></div></section><p id="c">C</p></div>
```

| Bundle | Inserted first | `#a`, `#b`, `#g` | canvas in `#g` | `#c` |
|---|---|---|---|---|
| v1.0.4 | `<x-note>` with a `render` | new nodes | gone | same node |
| v1.0.4 | a tag nothing defines | same nodes | kept | same node |
| v1.0.4 + a depth counter that keeps the pantry attached | `<x-note>` with a `render` | new nodes | gone | same node |
| v1.0.4 + the depth counter and the outer morph's id maps restored after a nested one | `<x-note>` with a `render` | same nodes | kept | same node |

A fix that only makes the pantry re-entrant stops the stack overflow but leaves this case broken; restoring the outer call's `ctxIdMap` and `ctxPersistentIds` when a nested call ends fixes both. That is what Starbase runs until your release (`patches/rocket/0008`):

```ts
  const saved = morphDepth++
    ? ([new Map(ctxIdMap), new Set(ctxPersistentIds)] as const)
    : null
  if (!saved) DOCUMENT.body.insertAdjacentElement('afterend', ctxPantry)
  try {
    // ... the morph as before ...
  } finally {
    morphDepth--
    if (!saved) {
      ctxPantry.remove()
    } else {
      const [idMap, persistentIds] = saved
      ctxIdMap.clear()
      for (const [node, ids] of idMap) ctxIdMap.set(node, ids)
      ctxPersistentIds.clear()
      for (const id of persistentIds) ctxPersistentIds.add(id)
    }
  }
```

*Claude was used to draft this comment, find the case and check it in Chrome.*
