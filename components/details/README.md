---
name: Details
tag: sb-details
category: layout
summary: A disclosure panel that animates open, alone or as an exclusive accordion.
author: zweiundeins
tags: [details, disclosure, accordion, collapse, expand, summary, faq]
since: 2026-09-23
preview: |
  <sb-details open summary="Flight plan" icon="🛰️" style="inline-size: 16rem">Burn 4.2 s at T+38, then coast to the transfer orbit.</sb-details>
usage: |
  <sb-details summary="Flight plan">Burn 4.2 s at T+38, then coast to the transfer orbit.</sb-details>
playground:
  content: Burn 4.2 s at T+38, then coast to the transfer orbit.
  values: {summary: Flight plan, icon: "🛰️"}
  attrs: {open: "true"}
  exclude: [name, group, open]
---

A summary row that opens to reveal its content, with a height animation that always lands on the content's own height. On its own it is a disclosure; give a few of them the same `group` and they become an accordion where only one panel is open.

`open` is **view state**, not a value: the server may own it, it reports changes with its own `sb-toggle` event, and it never takes part in a pending or `revert()` flow.

## Examples

### A single panel

Click the row, or focus it and press Enter or Space.

```html preview
<div style="display: grid; gap: 8px; inline-size: min(100%, 26rem)">
  <sb-details summary="What is a Rocket component?" icon="🚀">
    A custom element declared with <code>rocket()</code>. It renders its own shadow DOM from
    Datastar signals, so the page can wire it with attributes only.
  </sb-details>
  <sb-details open summary="Already open">
    <code>open</code> on the first render opens the panel without an animation.
  </sb-details>
</div>
```

### An accordion

Panels that share a `group` close each other. Every panel that closes emits its own `sb-toggle`, so the page (or the server) can keep the whole group's state. Render at most one of them `open`: as with `<details name>`, a panel that arrives open while another one in its group is open starts closed.

```html preview
<div data-signals="{_last: '—'}" style="display: grid; gap: 8px; inline-size: min(100%, 26rem)">
  <sb-details group="specs" name="engine" summary="Engine" icon="🔥"
    data-on:sb-toggle="$_last = evt.detail.name + ' → ' + evt.detail.open">
    Two Raptor-class thrusters, 1.8 MN combined.
  </sb-details>
  <sb-details group="specs" name="hull" summary="Hull" icon="🛡️"
    data-on:sb-toggle="$_last = evt.detail.name + ' → ' + evt.detail.open">
    Ablative ceramic over a titanium frame.
  </sb-details>
  <sb-details group="specs" name="crew" summary="Crew" icon="👩‍🚀"
    data-on:sb-toggle="$_last = evt.detail.name + ' → ' + evt.detail.open">
    Four, plus provisions for 90 days.
  </sb-details>
  <p>Last <code>sb-toggle</code>: <b data-text="$_last"></b></p>
</div>
```

### When the server owns the open state

The buttons below stand in for the server: they change the `open` **attribute**, exactly like a re-render would. A changed attribute wins over whatever the user did; `open="false"` closes the panel. `data-preserve-attr` keeps a signal-driven attribute through a morph.

```html preview
<div data-signals="{_open: 'false'}" style="display: grid; gap: 10px; inline-size: min(100%, 26rem)">
  <div style="display: flex; gap: 8px; flex-wrap: wrap">
    <sb-button size="sm" data-on:click="$_open = 'true'">Server sends open</sb-button>
    <sb-button size="sm" variant="outline" data-on:click="$_open = 'false'">Server sends open="false"</sb-button>
  </div>
  <sb-details name="brief" summary="Mission brief" data-attr:open="$_open" data-preserve-attr="open">
    Toggle the panel by hand, then press a button: the attribute wins. Press the same button
    twice and nothing happens — re-sent identical markup is not a change.
  </sb-details>
</div>
```

On a Starbase page the same thing is one `sb-toggle` handler and one command:

```html
<sb-details name="brief" summary="Mission brief" open="{{ .Brief }}"
  data-on:sb-toggle="@post('/cmd/panel', {payload: {tabid: $tabid, ...evt.detail}})">
  …
</sb-details>
```

The command writes the panel into `tab_state`, the next render brings the `open` attribute back, and nothing was ever shown before the server produced it.

## View state, not a value

A value is what the user *decides*; view state is what they *look at*. `sb-details` only ever holds the second, and that changes the contract in three ways ([Components with several values](/contribute#components-with-several-values)):

- **Its event is `sb-toggle`, not `sb-change`.** `{ name, open }`, on every open and close, including the panels an accordion closes for you.
- **There is no `confirm` and no `revert()`.** Nothing here is "not confirmed yet": a panel the user opened is open, whatever the server thinks. A failed command simply leaves the next render's `open` attribute as it was, and that attribute wins.
- **The server can still own it.** `open` behaves like every other server-driven attribute:

| The server sends | What happens |
| --- | --- |
| `open` on the first render | Open on the first paint, with no opening animation |
| a **changed** `open` attribute | It wins, whatever the user had done. No `sb-toggle` is emitted: the server already knows |
| the **same** markup again | Nothing. A morph can never re-open a panel the user just closed |
| **no** `open` attribute any more | Nothing. Morphs also strip attributes that were only reflected, so a removal is not an instruction |
| `open="false"` | Closed. This is how the server closes a panel |

Local toggling never writes the attribute back, which is what makes the table above consistent. A panel that app code moves (a sortable list, a portal, `moveBefore`) keeps its open state, like a native `<details>`, and an `open` attribute changed while it was detached still wins when it comes back.

That last row is also why the playground above has no `open` switch: its boolean switches *remove* the attribute for `false`, and removal is exactly the case `sb-details` ignores. Click the summary, or use the example below.

One implementation note, since other components may want to copy this: `sb-details` watches the `open` **attribute** (a `MutationObserver` on the host), not the decoded prop. Rocket's `observeProps` only fires when the decoded value changes, so a server adding `open="false"` to an element that never carried the attribute — a real change of mind — would otherwise go unnoticed.

## A native `<details>` in the shadow root

The panel is a real `<details>` and `<summary>`, inside the component's shadow root. That is the whole design:

- **Native semantics for free.** The disclosure role and expanded state, Enter and Space on the summary, the browser's own find-in-page reaching closed content and opening the panel — none of it is re-implemented, so none of it can drift from what the platform does.
- **The shadow root keeps the server-owned attribute clean.** A `<details>` reflects its state into its own `open` attribute, which is exactly why a `<details>` in a morphed page needs `data-preserve-attr="open"`: the attribute is both the server's instruction and the browser's scratch space. Here the reflected attribute is *inside the shadow root*, where a morph never goes, while the **host's** `open` attribute is only ever read. The two never meet.
- **The animation is the browser's box.** `::details-content` is the box the browser already wraps the revealed content in, so it grows from `block-size: 0` to `auto` (with `interpolate-size: allow-keywords`) and lands on the content's own height — nothing is measured, and `content-visibility` flips discretely at the ends, which is what keeps closed content out of the layout, out of the tab order and still findable.

Where `::details-content` or `interpolate-size` is not supported, the panel simply opens and closes at once; everything else is unchanged.

Only one thing is not native: `group`. `<details name="…">` groups exclusively within a single tree scope, and each host has its own shadow root, so no two panels are ever in the same one. A small module-level registry does it instead, which is also what lets a closing sibling emit its own `sb-toggle`.

## Props

`summary` is the heading (the `summary` slot replaces it), `icon` a decorative glyph before it, `disabled` blocks toggling (the server can still open or close it), `group` makes an accordion, and `name` is what `sb-toggle` reports.

`host.open` reads and writes the live state, and `show()` / `hide()` do the same; all three emit `sb-toggle`, because a page asking for a change is a change.

```html preview
<div style="display: grid; gap: 8px; inline-size: min(100%, 26rem)">
  <sb-details data-ref:_panel>
    <b slot="summary">A slotted summary row 🛠️</b>
    Any markup can go in the <code>summary</code> slot.
  </sb-details>
  <div style="display: flex; gap: 8px">
    <sb-button size="sm" data-on:click="$_panel.show()">show()</sb-button>
    <sb-button size="sm" variant="outline" data-on:click="$_panel.hide()">hide()</sb-button>
  </div>
  <sb-details disabled summary="Disabled" icon="🔒">Locked by the server.</sb-details>
</div>
```

## Styling

Parts: `details` (the `<details>` box), `summary`, `icon`, `label`, `marker` (the pixel triangle, mirrored in right-to-left text) and `content` (the body inside the panel). The animated box itself is the browser's `::details-content`, which no part can name; its speed is `--sb-details-duration`.

The open panel is `sb-details::part(details):open`. Not `sb-details[open]`: the host's `open` attribute is the server's word, not the live state.

It styles against the semantic tokens — `--sb-surface-card`, `--sb-surface-hover`, `--sb-border`, `--sb-text-1`, `--sb-text-2`, `--sb-text-muted`, `--sb-brand-light`, `--sb-radius` — and notches its corners by `--sb-notch` (at `0` the radius takes over). `--sb-details-duration` (220 ms) sets the animation, and `prefers-reduced-motion: reduce` turns it off, opening and closing at once.

```html preview
<div style="display: grid; gap: 8px; inline-size: min(100%, 26rem)">
  <sb-details open summary="Slow and smooth" style="--sb-notch: 0; --sb-details-duration: 600ms">
    No notches, and a long animation.
  </sb-details>
  <sb-details open summary="Tinted panel" style="--sb-surface-card: var(--sb-brand-subtle); --sb-border: var(--sb-brand)">
    Re-map the tokens rather than hard-coding colours, and it still works on every theme.
  </sb-details>
</div>
```

## Accessibility

- **Summary:** a real `<summary>`, so the disclosure role, Enter and Space and every assistive technology's "expand" come from the browser. `aria-expanded` is kept in sync with it for the engines that do not expose the native state.
- **Panel:** `role="region"` named by the summary (`aria-labelledby`), which puts it in the landmark and rotor lists while it is open.
- **Closed content:** the browser's own `content-visibility: hidden` on `::details-content` keeps it out of the accessibility tree and out of the tab order, and find-in-page still reaches it and opens the panel.
- **Motion:** `prefers-reduced-motion: reduce` removes the height transition and the marker's rotation; the panel still opens and closes.
- **Forced colours:** the marker is drawn in `CanvasText`, and the focus ring is an outline, which forced colours keep.
- **Disabled:** `<summary>` has no disabled state, so the component sets `aria-disabled` and `tabindex="-1"` (out of the tab order) and cancels the default action of the click — which is also what Enter and Space trigger.
