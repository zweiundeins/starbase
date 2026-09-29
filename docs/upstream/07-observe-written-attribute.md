**Title:** Rocket: `observeProps` misses an attribute write that decodes to the current value

**Filed:** https://github.com/starfederation/datastar/issues/1223

### Bug Report

`#setProp` returns early when the decoded value is unchanged (`Object.is`, runtime.ts:991), so `observeProps` never hears about it. That's right for identical markup re-sent by a morph, which doesn't write the attribute at all. It's wrong when the attribute really changes but its decoded value happens to equal the prop's current value, typically its default.

A server-driven toggle shows the problem. It keeps the user's choice locally and lets the server overrule it with a new `checked`. The page renders `<x-toggle>` without `checked`, the user switches it on, and the server answers with `checked="false"`. The attribute goes from absent to `"false"`, which decodes to `false`, the prop's current value, so no observer runs and the toggle stays on, against the server. Every value component that must let the server clear it runs into this and ends up watching its own attribute with a `MutationObserver`.

This is different from #1035 and #1139, where identical markup was re-sent and nothing was written: here the attribute is written, from absent to present.

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/gbmxpWj (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-obs', {
    props: ({ bool }) => ({ checked: bool }),
    setup: ({ observeProps }) => observeProps(() => console.log('observed')),
    render: ({ html }) => html`<span>x</span>`,
  })
  await customElements.whenDefined('x-obs')
  const el = document.createElement('x-obs')
  document.body.append(el)
  await new Promise(requestAnimationFrame)
  el.setAttribute('checked', 'false') // nothing logged
  el.setAttribute('checked', '')      // "observed"
</script>
```

### Suggested fix

`attributeChangedCallback` knows whether the attribute text changed. When it did, observers hear about it even if the decoded value is the same. A re-render still happens only when the value changes.

```diff
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..bef9afb 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -987,8 +987,12 @@ export function rocket<
     #setProp(
       name: keyof InferProps<Defs> & string,
       nextValue: InferProps<Defs>[typeof name],
+      // The attribute's text changed: observers hear it even if the value is the
+      // same (checked="false" written onto a toggle that had no checked attribute).
+      written = false,
     ) {
-      if (Object.is(this.#props[name], nextValue)) return
+      const same = Object.is(this.#props[name], nextValue)
+      if (same && !written) return
       this.#props[name] = nextValue
       const changes = { [name]: nextValue } as Partial<InferProps<Defs>>
       for (const observer of this.#propObservers) {
@@ -998,6 +1002,7 @@ export function rocket<
       }
 
       // Notify prop observers synchronously, then optionally queue a render so a burst of writes still collapses into one render.
+      if (same) return
       if (
         typeof renderOnPropChange === 'function'
           ? renderOnPropChange({
@@ -1573,6 +1578,7 @@ export function rocket<
         newValue === null
           ? getCodecDefault(propDefs[propName])
           : decodeCodec(propDefs[propName], newValue),
+        oldValue !== newValue,
       )
     }
 
```

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
