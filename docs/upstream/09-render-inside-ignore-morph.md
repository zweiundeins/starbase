**Title:** Rocket: a light-DOM component draws nothing inside a `data-ignore-morph` element

**Filed:** https://github.com/starfederation/datastar/issues/1224

### Bug Report

A component with `mode: 'light'` renders by morphing its own host: `morph(this.#mountRoot, fragment, 'inner')` (runtime.ts:1186), where the mount root is the host. `morph()` returns at once when an ancestor of its target carries `data-ignore-morph` (`oldElt.parentElement?.closest(...)`, patchElements.ts:301). So inside such an element a light component never draws, neither when it connects nor on a prop change. The same component draws outside it, and a shadow-DOM component draws in both places, since a shadow root has no `parentElement`.

`data-ignore-morph` is where pages keep DOM the client owns: a chart's canvas, a toast stack, a result the server sends once. It keeps server patches out of that subtree; a component rendering its own content is not a patch. We found it in nfsen-ng (a NetFlow viewer on Datastar), whose toasts live in a morph-ignored stack and now build their markup in `setup` instead of `render`.

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/bNqrOOL (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<div id="ignored" data-ignore-morph></div>
<div id="plain"></div>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-badge', { mode: 'light', render: ({ html }) => html`<b>drawn</b>` })
  await customElements.whenDefined('x-badge')
  ignored.append(document.createElement('x-badge'))
  plain.append(document.createElement('x-badge'))
  await new Promise(requestAnimationFrame)
  console.log('ignored:', ignored.innerHTML) // <x-badge ...></x-badge>: empty
  console.log('plain:', plain.innerHTML)     // <x-badge ...><b>drawn</b></x-badge>
</script>
```

A page with a prop change: `docs/repro/rocket-render-ignore-morph/` in https://github.com/zweiundeins/starbase.

### Suggested fix

Rocket's own render skips the ancestor check; a server patch that targets the ignored element or anything in it still does nothing:

```diff
diff --git a/library/src/plugins/watchers/patchElements.ts b/library/src/plugins/watchers/patchElements.ts
index 0208f43..ac7b692 100644
--- a/library/src/plugins/watchers/patchElements.ts
+++ b/library/src/plugins/watchers/patchElements.ts
@@ -292,13 +292,16 @@ export const morph = (
   oldElt: Element | ShadowRoot,
   newContent: DocumentFragment | Element,
   mode: 'outer' | 'inner' = 'outer',
+  // A component morphing its own content: an ignore-morph ancestor keeps patches
+  // out of the subtree, not the component out of itself.
+  ownContent = false,
 ): void => {
   if (
     (isHTMLOrSVG(oldElt) &&
       isHTMLOrSVG(newContent) &&
       oldElt.hasAttribute(aliasedIgnoreMorph) &&
       newContent.hasAttribute(aliasedIgnoreMorph)) ||
-    oldElt.parentElement?.closest(aliasedIgnoreMorphAttr)
+    (!ownContent && oldElt.parentElement?.closest(aliasedIgnoreMorphAttr))
   ) {
     return
   }
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..51ec0c5 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -1183,7 +1183,7 @@ export function rocket<
           }
         }
       }
-      morph(this.#mountRoot!, fragment, 'inner')
+      morph(this.#mountRoot!, fragment, 'inner', true)
       this.#syncAdoptedStyles()
       if (this.#datastarApplied) {
         this.#conditionalCleanups.push(
```

With it, the component draws inside the ignored element and re-renders there on a prop change, and both an outer morph of the ignored element's parent and a selector patch of a child in it still change nothing (checked in Chrome).

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
