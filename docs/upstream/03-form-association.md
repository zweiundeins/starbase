**Title:** Rocket: form-associated components (formAssociated, form callbacks, delegatesFocus)

**Filed:** https://github.com/starfederation/datastar/issues/1220

### Feature Request

Rocket components that are form controls (a text field, a select, a toggle, a slider, a radio group) can't take part in a `<form>`. They are missing from `FormData`, and with it from native submits and from Datastar's own `contentType: 'form'` requests. They can't be `required` or invalid (Datastar calls `checkValidity()` before a form request), a form reset doesn't reset them, a disabled `<fieldset>` doesn't disable them, and `<label for>` and `autofocus` don't reach them.

The platform has form-associated custom elements for this, but part of it can only be declared by the element class at `customElements.define()` time, and `rocket()` creates and defines that class itself (`RocketElement`, runtime.ts:690 and :1622), with no option to opt in.

### Reproduce

**CodePen:** https://codepen.io/mbolli/pen/zxZdGwB (open the browser devtools console, not CodePen's).

```html
<!doctype html>
<script type="module">
  import { rocket } from 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  rocket('x-field', {
    props: ({ string }) => ({ name: string, value: string }),
    render: ({ html, props: { value } }) => html`<input value="${value}">`,
  })
</script>
<form id="f">
  <x-field name="planet" value="mars"></x-field>
  <input name="native" value="n">
</form>
<script type="module">
  await customElements.whenDefined('x-field')
  console.log(JSON.stringify([...new FormData(f)]))          // [["native","n"]]: x-field is missing
  console.log(customElements.get('x-field').formAssociated)  // undefined
</script>
```

### What it takes

Checked in Chrome with plain custom elements:

1. **`static formAssociated = true` on the class.** With it, `attachInternals()` does most of the job: `internals.setFormValue()` puts the value in `FormData` under the element's `name` attribute, `setValidity()` makes `form.checkValidity()` fail, and `internals.labels`, `form.elements`, the `form="id"` attribute and exclusion inside a disabled `<fieldset>` all work.
2. **`delegatesFocus`.** The shadow root is created with only `mode` (runtime.ts:1232), so `host.focus()`, `autofocus` and a click on a `<label for>` don't reach the inner control.
3. **The form callbacks.** `ElementInternals` has no events: the browser tells a form-associated element about a reset, a disabled fieldset, a new form owner or a restored value only through `formResetCallback`, `formDisabledCallback`, `formAssociatedCallback` and `formStateRestoreCallback`, and it reads them from the class at `define()` time (a `formResetCallback` added to the prototype afterwards is never called).
4. **`internals` in the setup context.** `attachInternals()` works once per element, and `setup` runs again on every connect, so components keep their internals in a WeakMap today.

With the patch below, a component looks like this:

```js
rocket('x-field', {
  formAssociated: true,
  delegatesFocus: true,
  props: ({ string }) => ({ name: string, value: string }),
  setup: ({ $$, effect, internals, onFormReset, props }) => {
    $$.value = props.value
    effect(() => internals.setFormValue($$.value))
    onFormReset(() => ($$.value = props.value))
  },
  render: ({ html }) => html`<input data-bind:value>`,
})
```

### Suggested fix

`internals` is attached lazily, on first use, so components that never ask for it (or call `attachInternals()` themselves) are unaffected. The form handlers are dropped on disconnect, like other setup state.

```diff
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..4d4834b 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -138,6 +138,15 @@ type SetupContext<Props extends Record<string, any>> = {
     setter?: PropOverrideSetter<Props, Name>,
   ): void
   defineHostProp(name: string, descriptor: HostPropDescriptor): void
+  // The element's ElementInternals, attached once per element (setup reruns on every connect).
+  internals: ElementInternals
+  // Form callbacks of a `formAssociated` element, dropped on disconnect like other setup state.
+  onFormReset(fn: () => void): void
+  onFormDisabled(fn: (disabled: boolean) => void): void
+  onFormAssociated(fn: (form: HTMLFormElement | null) => void): void
+  onFormStateRestore(
+    fn: (state: unknown, mode: 'restore' | 'autocomplete') => void,
+  ): void
   render: SetupRender<Props>
   host: RocketHostWithProps<Props>
 }
@@ -211,6 +220,10 @@ type RocketDefinition<
   onFirstRender?: (context: FirstUpdateContext<InferProps<Defs>, Refs>) => void
   render?: RocketRender<InferProps<Defs>>
   mode?: 'open' | 'closed' | 'light'
+  // A form-associated element: it takes part in its <form> through `internals`.
+  formAssociated?: boolean
+  // Focus on the host (focus(), autofocus, <label for>) goes to the shadow root's first focusable element.
+  delegatesFocus?: boolean
   renderOnPropChange?:
     | boolean
     | ((context: {
@@ -575,6 +588,8 @@ export function rocket<
   const {
     manifest,
     mode,
+    formAssociated,
+    delegatesFocus,
     props,
     render,
     renderOnPropChange,
@@ -688,6 +703,16 @@ export function rocket<
   ]
 
   class RocketElement extends HTMLElement {
+    static formAssociated = formAssociated === true
+    #internals?: ElementInternals
+    #form = {
+      reset: [] as Array<() => void>,
+      disabled: [] as Array<(disabled: boolean) => void>,
+      associated: [] as Array<(form: HTMLFormElement | null) => void>,
+      restore: [] as Array<
+        (state: unknown, mode: 'restore' | 'autocomplete') => void
+      >,
+    }
     #instanceId = ''
     #signalPathBase = ''
     #props = Object.fromEntries(
@@ -1231,6 +1256,7 @@ export function rocket<
             : (this.shadowRoot ??
               this.attachShadow({
                 mode: mode === 'closed' ? 'closed' : 'open',
+                delegatesFocus: delegatesFocus === true,
               }))
       }
       if (!this.#hostTreeScoped) {
@@ -1496,6 +1522,11 @@ export function rocket<
         defineHostProp: (name, descriptor) => {
           this.#defineHostProp(name, descriptor)
         },
+        internals: undefined as unknown as ElementInternals, // defined lazily below
+        onFormReset: (fn) => void this.#form.reset.push(fn),
+        onFormDisabled: (fn) => void this.#form.disabled.push(fn),
+        onFormAssociated: (fn) => void this.#form.associated.push(fn),
+        onFormStateRestore: (fn) => void this.#form.restore.push(fn),
         render: ((
           overrides: RenderContextOverrides<InferProps<Defs>>,
           ...args: any[]
@@ -1508,6 +1539,13 @@ export function rocket<
         ...context,
         refs: refs as unknown as InstancesOf<Refs>,
       }
+      // Lazy, so a component that never asks (or attaches its own) isn't affected.
+      const internals = {
+        get: () => (this.#internals ??= this.attachInternals()),
+        enumerable: true,
+      }
+      Object.defineProperty(context, 'internals', internals)
+      Object.defineProperty(firstUpdateContext, 'internals', internals)
 
       setup?.(context)
       this.#render({})
@@ -1576,6 +1614,21 @@ export function rocket<
       )
     }
 
+    // Read by the browser at define() time, so they exist on every Rocket class;
+    // only a `formAssociated` element ever receives them.
+    formResetCallback() {
+      for (const fn of this.#form.reset) fn()
+    }
+    formDisabledCallback(disabled: boolean) {
+      for (const fn of this.#form.disabled) fn(disabled)
+    }
+    formAssociatedCallback(form: HTMLFormElement | null) {
+      for (const fn of this.#form.associated) fn(form)
+    }
+    formStateRestoreCallback(state: unknown, mode: 'restore' | 'autocomplete') {
+      for (const fn of this.#form.restore) fn(state, mode)
+    }
+
     disconnectedCallback() {
       mergePaths([[this.#signalPathBase, null]])
       this.removeEventListener(
@@ -1595,6 +1648,7 @@ export function rocket<
       for (const name of Object.keys(this.#refs)) delete this.#refs[name]
       this.#propObservers = []
       this.#actions.clear()
+      for (const handlers of Object.values(this.#form)) handlers.length = 0
       this.#mounted = false
     }
     static {
```

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released. With it, the component above submits its value, `form.reset()` resets it and `el.focus()` focuses the inner input.

Related: #1219. Once an element is form-associated, a reflected `disabled="false"` disables it.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
