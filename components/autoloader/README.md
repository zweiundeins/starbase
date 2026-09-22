---
name: Autoloader
tag: sb-autoloader
category: utilities
summary: Loads any web component the first time its tag appears, morphs included.
author: zweiundeins
tags: [loader, lazy, modules, custom-elements, infrastructure]
since: 2026-09-23
preview: |
  <sb-autoloader modules='{"demo-badge": "/c/autoloader/demo-badge.js"}'></sb-autoloader>
  <demo-badge>loaded on demand</demo-badge>
playground:
  attrs: {modules: '{"demo-badge": "/c/autoloader/demo-badge.js"}'}
  content: <demo-badge>loaded on demand</demo-badge>
---

Put it on a page and every custom element loads itself the first time its tag shows up: on the first render, and later too, when a Datastar morph or a script adds markup. Nothing else changes, and pages ship only the components they actually use.

It is the generic version of the autoloader this site serves at [`/c/autoloader.js`](/c/autoloader.js), which carries Starbase's own tag map. This one takes the map (or a URL pattern) from you, and works with any web component, not only Rocket ones.

## Examples

### A map of tags

```html preview
<sb-autoloader modules='{"demo-badge": "/c/autoloader/demo-badge.js"}'></sb-autoloader>
<demo-badge>loaded on demand</demo-badge>
```

`demo-badge` is a plain custom element with no dependencies. Nothing on the page imports it; the autoloader fetches it because the tag is there.

### A pattern for a whole folder

With a naming convention, one line covers every component. `{tag}` is the element's name, and `match` keeps it to your own tags, so unrelated elements (a third-party widget, an icon element) are left alone:

```html
<sb-autoloader pattern="/components/{tag}/{tag}.js" match="^x-"></sb-autoloader>

<x-chart></x-chart>   <!-- loads /components/x-chart/x-chart.js -->
<x-table></x-table>   <!-- loads /components/x-table/x-table.js -->
```

Entries in `modules` win over the pattern, so exceptions stay easy:

```html
<sb-autoloader
  pattern="/components/{tag}/{tag}.js"
  match="^x-"
  modules='{"x-legacy": "https://cdn.example.com/legacy/bundle.js"}'
></sb-autoloader>
```

### No flash of undefined elements

Custom elements are empty until their module arrives. Put a class on `<html>`, hide undefined elements with it, and name it in `cloak`: it is removed once the first round of components is defined, or after `timeout` (3 s by default), so a failed module can never leave the page blank.

```html
<html class="loading">
<style>.loading :not(:defined) { visibility: hidden }</style>

<sb-autoloader pattern="/c/{tag}/{tag}.js" cloak="loading"></sb-autoloader>
```

### Watch what it loads

`sb-load` fires per component. `sb-ready` fires once, when the components that were on the page at startup are defined; later arrivals keep firing `sb-load` only.

```html preview
<div data-signals="{_log: 'waiting…'}">
  <sb-autoloader
    modules='{"demo-card": "/c/autoloader/demo-badge.js?tag=demo-card"}'
    data-on:sb-load="$_log = 'loaded <' + evt.detail.tag + '>'"
    data-on:sb-ready="$_log += ' · ready (' + evt.detail.loaded + ')'"
    data-on:sb-load-error="$_log = 'could not load <' + evt.detail.tag + '>: ' + evt.detail.error"
  ></sb-autoloader>
  <demo-card>a tag of its own</demo-card>
  <p data-text="$_log"></p>
</div>
```

(The demo module takes the tag it defines from `?tag=`, so this example loads something the examples above haven't.) A tag that is already defined is never fetched again, so an autoloader that finds nothing to do goes straight to `sb-ready` with `loaded: 0`.

From JavaScript, `el.ready` is a promise that resolves with the number of components loaded, and `el.load('x-chart')` fetches one by hand.

## Dependencies between components

A component that renders another tag inside its own shadow root would otherwise only be discovered after it rendered. List those tags in `requires` and they load together:

```html
<sb-autoloader
  pattern="/c/{tag}/{tag}.js"
  requires='{"sb-code-playground": ["sb-code-editor"]}'
></sb-autoloader>
```

## Notes

- **Wrapping is optional.** `<sb-autoloader>` works as a single tag anywhere on the page; children render untouched, so you can also wrap the markup that needs the components.
- **Once per tag.** Two autoloaders on a page, or one that survives a morph, never fetch the same module twice. A failed load is retried when the tag appears again.
- **Shadow roots.** It watches the document, so components inside other components' shadow roots are discovered when those render into the page, not before. `requires` covers the rest.
- **Rocket components** import `datastar`, so the page still needs the import map for it. Plain custom elements need nothing.
- **A failed module** reports through `reportError` (so `window.onerror` sees it) and emits `sb-load-error`. A `match` that isn't a valid regular expression is reported the same way, and the pattern is skipped.
- **`demo-badge.js`** in this component's folder belongs to the examples above, not to the loader; that is why it shows up in the size table. It is served at `/c/autoloader/demo-badge.js`, so in a pull-request preview (where the component is not in the catalog yet) the examples report a failed load: the loader works, the demo module is simply not there.

## Accessibility

The element has no box of its own (`display: contents`) and no role; it adds nothing to the accessibility tree and takes no focus. Children, if you wrap any, render exactly where they are written. Use `cloak` so people don't see half-built elements, and remember the page must work if a module never arrives.
