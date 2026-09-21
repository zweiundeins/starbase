---
name: Button
tag: sb-button
category: forms
summary: A versatile button component with Datastar interactions.
author: zweiundeins
tags: [button, action, link, cta]
since: 2026-09-21
preview: |
  <sb-button variant="pixel" size="lg" caret>Blast off</sb-button>
---

Buttons start things. `sb-button` comes in five variants, three sizes, and renders a real `<a>` when given an `href`. Clicks bubble out of the shadow root, so any Datastar `data-on:click` just works.

## Examples

### Variants

```html preview
<sb-button>Primary</sb-button>
<sb-button variant="outline">Outline</sb-button>
<sb-button variant="ghost">Ghost</sb-button>
<sb-button variant="danger">Abort</sb-button>
<sb-button variant="pixel" caret>Blast off</sb-button>
```

### Sizes

```html preview
<sb-button size="sm">Small</sb-button>
<sb-button size="md">Medium</sb-button>
<sb-button size="lg">Large</sb-button>
```

### With Datastar

Local signals start with an underscore, so they stay in the browser and are never sent to the server.

```html preview
<div data-signals:_count="0">
  <sb-button data-on:click="$_count++">Launches: <span data-text="$_count"></span></sb-button>
  <sb-button variant="ghost" data-on:click="$_count = 0" data-attr:disabled="$_count === 0">Reset</sb-button>
</div>
```

Talking to the backend works the same way: `data-on:click="@post('/launch')"`.

### As a link

```html preview
<sb-button href="https://data-star.dev" variant="outline" caret>Read the Datastar docs</sb-button>
```

## Accessibility

A native `<button>` (or `<a>`) inside the shadow root does the work: keyboard focus, Enter and Space, and screen reader semantics. Disabled buttons are removed from the tab order.
