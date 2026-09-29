**Title:** Rocket: `data-on:*__window` inside a shadow root outlives the element

**Filed:** https://github.com/starfederation/datastar/issues/1221

### Bug Report

A `data-on:x__window="@act()"` on an element inside a Rocket component's shadow root adds a listener to `window` that is never removed when the component is removed. Datastar's document observer runs attribute cleanups for removed elements and their descendants, but it can't see into a shadow root. Rocket clears the component's actions on disconnect, so every later event on `window` throws `UndefinedAction` (and an uncaught `ExecuteExpression`), once per removed instance, and the listener keeps the detached element alive.

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/ogZeXWe (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-win', {
    setup: ({ action }) => action('seen', () => console.log('seen')),
    render: ({ html }) => html`<div data-on:x-ping__window="@seen()"></div>`,
  })
  await customElements.whenDefined('x-win')
  const el = document.createElement('x-win')
  document.body.append(el)
  await new Promise(requestAnimationFrame)
  el.remove()
  dispatchEvent(new Event('x-ping')) // Uncaught Error: ExecuteExpression (UndefinedAction "seen")
</script>
```

### Suggested fix

Rocket knows its shadow root, so it runs the Datastar cleanups for that tree on disconnect. The engine already has `cleanupEls`; this exports a tree version of it. On the next connect Rocket applies the attributes again, so a removed and re-inserted element hears `window` events again. An atomic move (`moveBefore()` with `connectedMoveCallback`, #1218) doesn't disconnect, so a moved element keeps its listeners.

```diff
diff --git a/library/src/engine/engine.ts b/library/src/engine/engine.ts
index 5db457b..bc723e6 100644
--- a/library/src/engine/engine.ts
+++ b/library/src/engine/engine.ts
@@ -111,6 +111,11 @@ const cleanupEls = (els: Iterable<HTMLOrSVG>): void => {
   }
 }
 
+// Runs the attribute cleanups of every element under root: for a shadow root,
+// which the document observer can't see into when its host is removed.
+export const cleanupTree = (root: ParentNode): void =>
+  cleanupEls(root.querySelectorAll<HTMLOrSVG>('*'))
+
 const aliasedIgnore = aliasify('ignore')
 const aliasedIgnoreAttr = `[${aliasedIgnore}]`
 const shouldIgnore = (el: HTMLOrSVG) =>
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..d88a06f 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -2,6 +2,7 @@ import {
   actions,
   apply,
   applyElement,
+  cleanupTree,
   isDocumentObserverActive,
   action as registerAction,
 } from '@engine'
@@ -1595,6 +1596,8 @@ export function rocket<
       for (const name of Object.keys(this.#refs)) delete this.#refs[name]
       this.#propObservers = []
       this.#actions.clear()
+      // Listeners on window or document (data-on:*__window) would outlive us.
+      if (this.#mountRoot instanceof ShadowRoot) cleanupTree(this.#mountRoot)
       this.#mounted = false
     }
     static {
```

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

Reported earlier as the second point of @troygilman's comment on #1209, for hosts recreated by a reorder; it happens on any removal. If the fix for #1209 already covers this, please close it. Related: #1211 (callbacks queued by timing modifiers survive cleanup).

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
