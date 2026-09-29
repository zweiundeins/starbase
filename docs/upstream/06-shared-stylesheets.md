**Title:** Rocket: `adoptStyles` parses one stylesheet per instance

**Filed:** https://github.com/starfederation/datastar/issues/1222

### Feature Request

`adoptStyles(host, css)` builds and parses a new `CSSStyleSheet` for every instance of a component (`#syncAdoptedStyles`, runtime.ts:1045). A page with many instances parses the same CSS again and again: on the docs page for our dropdown component (38 component instances), that is 382 KB of CSS, of which 65 KB is unique. Constructed stylesheets can be shared between shadow roots of the same document, so every instance could adopt one sheet.

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/myWMJmL (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-styled', {
    setup: ({ adoptStyles, host }) => adoptStyles(host, ':host { color: red }'),
    render: ({ html }) => html`<span>x</span>`,
  })
  await customElements.whenDefined('x-styled')
  const a = document.createElement('x-styled'), b = document.createElement('x-styled')
  document.body.append(a, b)
  await new Promise(requestAnimationFrame)
  console.log(a.shadowRoot.adoptedStyleSheets[0] === b.shadowRoot.adoptedStyleSheets[0]) // false
</script>
```

### Suggested fix

Keep constructed sheets in a map keyed by their CSS text. Rocket always constructs them in its own window, so sharing across that document's shadow roots is allowed.

```diff
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..9e79cfb 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -555,6 +555,10 @@ registerAction({
 })
 
 // Register a custom element tag and wire its props, setup, render, and Datastar bridge into the element class.
+// Constructed stylesheets by their text: every instance of a component adopts
+// the same sheet, parsed once, instead of a copy each.
+const sharedSheets = new Map<string, CSSStyleSheet>()
+
 export function rocket<
   Refs extends RefCtors = RefCtors,
   Defs extends PropDefs = PropDefs,
@@ -1050,8 +1054,12 @@ export function rocket<
           'replaceSync' in CSSStyleSheet.prototype
         ) {
           if (!style.sheet) {
-            style.sheet = new CSSStyleSheet()
-            style.sheet.replaceSync(style.text)
+            style.sheet = sharedSheets.get(style.text)
+            if (!style.sheet) {
+              style.sheet = new CSSStyleSheet()
+              style.sheet.replaceSync(style.text)
+              sharedSheets.set(style.text, style.sheet)
+            }
           }
           if (!style.root.adoptedStyleSheets.includes(style.sheet)) {
             style.root.adoptedStyleSheets = [
```

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
