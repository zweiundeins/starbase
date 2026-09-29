**Title:** Rocket: moving an element runs setup again and loses its state (connectedMoveCallback)

**Filed:** https://github.com/starfederation/datastar/issues/1218

### Bug Report

Every move of a Rocket element is a full disconnect and connect: Rocket deletes the local `$$` signals and runs `setup` again. A counter, text being typed, an unsaved code buffer, a count-up that already counted: all start over. Native elements keep their state when moved.

Datastar's morph moves keyed elements with `moveBefore()`, and for that the platform has an answer: when an element defines `connectedMoveCallback`, the browser calls it instead of `disconnectedCallback` and `connectedCallback` (Chrome 133+).

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/emvENWp (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<div id="box"><i></i></div>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-count', {
    setup: ({ $$, action }) => { $$.n = 0; action('inc', () => $$.n++) },
    render: ({ html }) => html`<button data-on:click="@inc()" data-text="$$n"></button>`,
  })
  await customElements.whenDefined('x-count')
  const el = document.createElement('x-count')
  box.prepend(el)
  await new Promise(requestAnimationFrame)
  el.shadowRoot.querySelector('button').click()
  el.shadowRoot.querySelector('button').click()
  box.moveBefore(el, null) // an atomic move, as the morph does
  await new Promise(requestAnimationFrame)
  console.log(el.shadowRoot.querySelector('button').textContent) // "0", was "2"
</script>
```

### Suggested fix

```diff
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..f846c13 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -1576,6 +1576,10 @@ export function rocket<
       )
     }
 
+    // An atomic move (moveBefore, which Datastar's morph uses) keeps the element
+    // connected: no teardown, no second setup, nothing lost.
+    connectedMoveCallback() {}
+
     disconnectedCallback() {
       mergePaths([[this.#signalPathBase, null]])
       this.removeEventListener(
```

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released. With it, a counter moved with `moveBefore()` keeps its count. A remove followed by an append is still a disconnect and a connect, as the platform defines it.

Possibly related: the first point in @troygilman's comment on #1209 (hosts recreated on a reorder). If the fix for #1209 already covers this, please close it.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
