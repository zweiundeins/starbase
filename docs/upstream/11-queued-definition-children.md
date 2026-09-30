**Title:** Rocket: light children of a shadow component never bind when `rocket()` runs before Datastar's first pass

**Filed:** not yet

### Bug Report

When `rocket()` runs before Datastar's first pass, the definition is queued until `datastar-ready` (runtime.ts:563 to :572), and `markPendingRocketHosts` marks each child of a matching shadow host with `data-ignore` (runtime.ts:393 to :394), so the first pass cannot evaluate `$$` expressions in the page's scope. When the definition runs, `connectedCallback` removes the markers (runtime.ts:1208 to :1214), but nothing applies those children: the upgrade applies the host (`applyElement(this)`, runtime.ts:1524) and the shadow root (`apply(this.#mountRoot, true)`, runtime.ts:1531), and the engine's observer does nothing when `data-ignore` goes away. So the children and everything in them stay inert: no `data-text`, no `data-on`, no `data-init`. Attributes that Rocket rewrites for `$$` do bind, because the rewrite is an attribute change the observer applies, which hides the problem when every expression uses `$$`.

It happens whenever the component's module runs before Datastar's first pass: a module that imports the bundle and defines at once (the CodePen below), or a component script ahead of the bundle.

### Reproduce

**CodePen:** (to create with `codepen.html`, #11; open the browser devtools console, not CodePen's).

```html
<!doctype html>
<x-card>
  <button data-on:click="console.log('child clicked')">child</button>
  <span data-text="'bound'">not bound</span>
</x-card>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-card', { render: ({ html }) => html`<slot></slot>` }) // before Datastar's first pass: queued
  setTimeout(() => {
    console.log(document.querySelector('x-card span').textContent) // "not bound"
    document.querySelector('x-card button').click()                 // logs nothing
  }, 500)
</script>
```

### Suggested fix

Remember the children that were hidden before the first pass, and apply them once the upgrade has scoped them:

```diff
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..7d38c43 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -392,10 +392,14 @@ const markPendingRocketHosts = (root: ParentNode) => {
       }
       target.setAttribute(rocketIgnoreAttr, '')
       target.setAttribute(rocketDeferredIgnoreAttr, tagName)
+      if (!isDocumentObserverActive()) hiddenFromFirstApply.add(target)
     }
   }
 }
 
+// Marked before Datastar's first pass, which therefore skipped them: the upgrade applies them.
+const hiddenFromFirstApply = new WeakSet<Element>()
+
 // Keep the pre-upgrade guard in sync with live DOM insertion.
 //
 // Watch live DOM insertions and scoped attribute additions so newly unresolved Rocket content is deferred before Datastar can evaluate it.
@@ -1205,12 +1209,15 @@ export function rocket<
       )
 
       // Remove this host's child markers before scoping them while leaving independently pending nested Rocket hosts protected.
+      const neverApplied: HTMLOrSVG[] = []
       for (const node of this.querySelectorAll(
         `[${rocketDeferredIgnoreAttr}]`,
       )) {
         if (node.getAttribute(rocketDeferredIgnoreAttr) !== tag) continue
         node.removeAttribute(rocketDeferredIgnoreAttr)
         node.removeAttribute(rocketIgnoreAttr)
+        if (hiddenFromFirstApply.delete(node))
+          neverApplied.push(node as HTMLOrSVG)
       }
 
       // Apply host markers here because custom-element constructors cannot mutate attributes on freshly created elements.
@@ -1529,6 +1536,12 @@ export function rocket<
       // In `light` mode that surface is the host itself.
       // In shadow modes, re-applying the host would incorrectly rewrite outer-page bindings into Rocket's private signal path.
       apply(this.#mountRoot!, true)
+      // Light children of a shadow host: the mount root's apply does not reach them, and removing
+      // their marker applies nothing.
+      if (this.#mountRoot !== this) {
+        for (const node of neverApplied)
+          if (node.parentNode === this) apply(node, false)
+      }
       this.#datastarApplied = true
       this.#conditionalCleanups.push(
         ...initRocketStructures(this.#mountRoot!, {
```

With it, the children bind, nested ones included, a child's `data-init` runs once, and a child whose expression uses `$$` binds in the component's scope. A component defined after the first pass (a lazily imported module) applies its children only once, as before: their `data-init` still runs once (checked in Chrome). A child whose `data-init` uses `$$` runs it twice, with or without the fix.

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
