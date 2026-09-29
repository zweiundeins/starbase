**Title:** Rocket: an effect that throws during disconnect leaves the element dead

**Filed:** https://github.com/starfederation/datastar/issues/1217

### Bug Report

On disconnect, Rocket deletes the component's local signals first (`mergePaths([[base, null]])`, runtime.ts:1580). That re-runs the effects that read them, and one of them can throw, for example a computed signal that reads a signal that is now gone. The exception leaves `disconnectedCallback` before the cleanups, before `#propObservers` and `#actions` are reset and before `#mounted = false`. On the next connect, `connectedCallback` returns early (`if (this.#mounted) return`, runtime.ts:1200), so `setup` never runs again, and the old closure keeps running against deleted signals: every later action that reads `$$` throws.

Datastar's morph moves keyed elements, and apps move elements (tabs, drag and drop), so a remove and re-insert is routine.

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/dPvzovz (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<div id="box"></div>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  let setups = 0
  rocket('x-dead', {
    setup: ({ $$, action }) => {
      setups++
      $$.items = ['a']
      $$.first = () => $$.items[0].toUpperCase() // a computed signal
      action('ping', () => console.log('ping', $$.items.length))
    },
    render: ({ html }) => html`<b data-text="$$first"></b><button data-on:click="@ping()">ping</button>`,
  })
  await customElements.whenDefined('x-dead')
  const el = document.createElement('x-dead')
  box.append(el)
  await new Promise(requestAnimationFrame)
  el.remove()   // TypeError: Cannot read properties of undefined (reading '0')
  box.append(el)
  await new Promise(requestAnimationFrame)
  console.log('setups', setups) // 1: setup did not run again
  el.shadowRoot.querySelector('button').click() // throws: $$.items is gone
</script>
```

### Suggested fix

Finish the teardown whatever the signal deletion throws. The error is still reported, but the element is torn down and sets up again on the next connect:

```diff
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..5adf914 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -1577,7 +1577,11 @@ export function rocket<
     }
 
     disconnectedCallback() {
+      // Deleting the signals re-runs their effects, which can throw; the teardown
+      // must still finish, or the element stays mounted and setup never reruns.
+      try {
         mergePaths([[this.#signalPathBase, null]])
+      } finally {
         this.removeEventListener(
           DATASTAR_SCOPE_CHILDREN_EVENT,
           this.#scopePatchedChildren,
@@ -1597,6 +1601,7 @@ export function rocket<
         this.#actions.clear()
         this.#mounted = false
       }
+    }
     static {
       for (const name of propNames) {
         Object.defineProperty(RocketElement.prototype, name, {
```

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

Related: datastar-pro#75 (another `#mounted` problem on re-insert, fixed in RC8).

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
