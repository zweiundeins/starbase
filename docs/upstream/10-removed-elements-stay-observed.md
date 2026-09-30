**Title:** Rocket: removed elements are never garbage-collected (`observedRoots` only grows)

**Filed:** not yet

### Bug Report

`connectedCallback` ends with `apply(this.#mountRoot, true)` (runtime.ts:1531). That adds the mount root (the shadow root, or the host in light mode) to the engine's module-level `observedRoots` set (engine.ts:47, :255), and nothing ever deletes from it. Every Rocket element that was ever connected stays reachable, with its whole subtree: in the repro, after `remove()` and three garbage collections, 200 of 200 removed elements are still alive, and each keeps 46 DOM nodes (a shadow element with 40 light children; 44 in light mode). A plain custom element in the same place is collected.

Pages that replace components keep growing: a dashboard that redraws its charts, toasts, the rows of a table. nfsen-ng's dashboards stay open for days; it now empties every removed element and detaches it from its old parent, which leaves the bare element (1.7 DOM nodes each on its pages) but cannot free it.

The set is read in one place: when an attribute plugin registers late, `attribute()` applies it to every observed root (engine.ts:62). A disconnected root does not need that, and `connectedCallback` observes it again when the element comes back. The `MutationObserver` registration can stay: an observer holds the nodes it observes weakly.

### Reproduce

**CodePen:** (to create with `codepen.html`, #10; open the browser devtools console, not CodePen's).

```html
<!doctype html>
<div id="box"></div>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-tile', { render: ({ html }) => html`<b>tile</b>` })
  await customElements.whenDefined('x-tile')
  const refs = []
  for (let i = 0; i < 200; i++) {
    const el = document.createElement('x-tile')
    box.append(el)
    refs.push(new WeakRef(el))
  }
  await new Promise(requestAnimationFrame)
  box.replaceChildren()
  // Collect garbage (DevTools, Memory panel), then run check()
  window.check = () => console.log(refs.filter((r) => r.deref()).length, 'of 200 removed elements still alive') // 200
</script>
```

The DOM node counts come from `docs/repro/rocket-removed-elements/` in https://github.com/zweiundeins/starbase, measured through the DevTools protocol.

### Suggested fix

```diff
diff --git a/library/src/engine/engine.ts b/library/src/engine/engine.ts
index 5db457b..1c854e9 100644
--- a/library/src/engine/engine.ts
+++ b/library/src/engine/engine.ts
@@ -257,6 +257,11 @@ export const apply = (
   }
 }
 
+// Drops a root that apply() observed, once its owner is gone: the set would hold it for good.
+export const forgetRoot = (root: HTMLOrSVG | ShadowRoot): void => {
+  if (root !== DOCUMENT.documentElement) observedRoots.delete(root)
+}
+
 export const applyElement = (el: HTMLOrSVG, onlyNew = false): void => {
   applyEls([el], onlyNew)
 }
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..5cb4d24 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -2,6 +2,7 @@ import {
   actions,
   apply,
   applyElement,
+  forgetRoot,
   isDocumentObserverActive,
   action as registerAction,
 } from '@engine'
@@ -1595,6 +1596,8 @@ export function rocket<
       for (const name of Object.keys(this.#refs)) delete this.#refs[name]
       this.#propObservers = []
       this.#actions.clear()
+      // connectedCallback observes the mount root again; kept, it would keep this element alive.
+      if (this.#mountRoot) forgetRoot(this.#mountRoot)
       this.#mounted = false
     }
     static {
```

With it, 0 of 200 removed elements are alive and none of their DOM nodes stay, in both modes. An element removed and put back later still works: its `$$` counter counts, and a node appended into its shadow root after the reconnect binds (checked in Chrome). An atomic move (`moveBefore`) never disconnects the element, so it keeps its root.

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
