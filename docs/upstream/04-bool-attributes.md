**Title:** Rocket: `bool` props don't follow HTML boolean attributes (`checked="checked"` reads as false, false reflects as `"false"`)

**Filed:** https://github.com/starfederation/datastar/issues/1219

### Bug Report

Rocket's `bool` codec differs from HTML boolean attributes in both directions:

1. **Reading.** It accepts only `''`, `'true'` and `'1'` as true (codecs.ts:420-425). The XHTML form that many server frameworks and templating libraries emit, `checked="checked"` or `disabled="disabled"`, reads as false. A native element treats any present value as true, and CSS like `[disabled]` matches it too, so the component's look and its behaviour disagree.
2. **Reflecting.** A false value is written back as the string `"false"` (codecs.ts:426), so `el.disabled = false` leaves `disabled="false"` on the element. `[disabled]` selectors still match, and the element looks disabled while it isn't. A form-associated custom element (#1220) goes further: the browser treats any `disabled` attribute as disabled, so it matches `:disabled` and is left out of `FormData`.

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/EaWvjmg (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-switch', {
    props: ({ bool }) => ({ checked: bool, disabled: bool }),
    render: ({ html, props: { checked } }) => html`<span>${checked ? "on" : "off"}</span>`,
  })
</script>
<style>x-switch[disabled] { opacity: 0.5 }</style>
<x-switch id="a" checked="checked" disabled="disabled"></x-switch>
<x-switch id="b" disabled></x-switch>
<script type="module">
  await customElements.whenDefined('x-switch')
  console.log('a.checked', a.checked, '| a.disabled', a.disabled)  // false | false
  b.disabled = false
  console.log('b attribute', JSON.stringify(b.getAttribute('disabled')), '| opacity', getComputedStyle(b).opacity)  // "false" | 0.5
</script>
```

### Suggested fix

- Read any present value as true, except `"false"` and `"0"`. Keeping those two false preserves a useful convention: a server re-rendering the page can switch a prop off explicitly (`checked="false"`), which it can't do by leaving the attribute out when the element never had one.
- Reflect a false boolean prop by removing its attribute. The removal happens where the prop is reflected, not in the codec: array, tuple and object codecs encode nested booleans through the same codec and still need `"false"` there.

```diff
diff --git a/library/src/rocket/codecs.ts b/library/src/rocket/codecs.ts
index bbc14ba..b5f17b7 100644
--- a/library/src/rocket/codecs.ts
+++ b/library/src/rocket/codecs.ts
@@ -417,12 +417,12 @@ const createNumberCodec = (
 
 // Decode HTML-style truthy values because attribute presence and string forms both represent boolean props.
 const createBoolCodec = (
+  // Like a native boolean attribute, any present value is true (checked="checked"),
+  // except "false" and "0", so a server can still say false in markup.
   decode: (value: unknown) => boolean = (value) =>
     value === true ||
-    value === '' ||
-    value === 'true' ||
     value === 1 ||
-    value === '1',
+    (typeof value === 'string' && value !== 'false' && value !== '0'),
   encode: (value: boolean) => string = (value) => (value ? 'true' : 'false'),
   defaultFactory?: () => boolean,
   manifestMeta: CodecManifest = { type: 'boolean' },
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..5214377 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -1023,7 +1023,12 @@ export function rocket<
       const attrName = kebab(name)
       this.#reflecting = true
       try {
-        const encoded = propDefs[name].encode(value)
+        // A false boolean removes its attribute, as native ones do: [disabled]
+        // selectors and form-associated elements treat disabled="false" as disabled.
+        const encoded =
+          propDefs[name].manifestMeta?.type === 'boolean' && !value
+            ? null
+            : propDefs[name].encode(value)
         if (encoded == null) {
           this.removeAttribute(attrName)
         } else {
```

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
