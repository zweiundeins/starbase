// Generated from tree.ts by `go tool task ts`: edit the TypeScript, not this file.
import { rocket, startPeeking, stopPeeking } from 'datastar';
// Host getters must not subscribe callers (e.g. data-bind's sync effect).
const peek = (fn) => {
    startPeeking();
    try {
        return fn();
    }
    finally {
        stopPeeking();
    }
};
// One ElementInternals per element: attachInternals() works once, and setup
// runs again when the element is re-attached. Its custom states
// (:state(pending)) are styleable from the page and morph-proof.
const internals = new WeakMap();
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host);
const fromItems = (items, loaded) => (Array.isArray(items) ? items : []).map((it) => {
    const id = String(it.id);
    const kids = it.children ?? loaded?.[id];
    return { id, label: String(it.label ?? id), icon: it.icon ?? '', art: null, meta: '', href: '', target: '', rel: '', title: '', current: '', lazy: !!it.lazy, open: null, kids: kids && fromItems(kids, loaded) };
});
// The tree in the element's own markup: nested lists, an item per <li>.
// null without a list, and then the JSON form applies.
const fromMarkup = (host) => {
    const top = host.querySelector(':scope > ul, :scope > ol');
    if (!top)
        return null;
    const seen = new Set();
    const skip = '[data-icon], [data-meta], ul, ol, script, style, template';
    const text = (nodes) => nodes.map((n) => (n.nodeType === Node.TEXT_NODE ? n.textContent : n instanceof Element && !n.matches(skip) ? text([...n.childNodes]) : '')).join('');
    const squash = (s) => (s ?? '').replace(/\s+/g, ' ').trim();
    const read = (list, path) => [...list.children]
        .filter((li) => li.localName === 'li')
        .map((li) => {
        const kids = li.querySelector(':scope > ul, :scope > ol');
        const head = [...li.childNodes].filter((n) => !(n instanceof Element && n.matches('ul, ol')));
        const find = (sel) => {
            for (const n of head) {
                const m = n instanceof Element && (n.matches(sel) ? n : n.querySelector(sel));
                if (m)
                    return m;
            }
            return null;
        };
        const a = find('a[href]');
        const label = squash(text(head));
        const base = li.dataset.id || a?.getAttribute('href') || `${path}/${label}`;
        let id = base;
        for (let n = 2; seen.has(id); n++)
            id = `${base}~${n}`;
        seen.add(id);
        const current = find('[aria-current]')?.getAttribute('aria-current') ?? li.getAttribute('aria-current');
        return {
            id,
            label,
            icon: '',
            art: find('[data-icon]'),
            meta: squash(find('[data-meta]')?.textContent),
            href: a?.getAttribute('href') ?? '',
            target: a?.getAttribute('target') ?? '',
            rel: a?.getAttribute('rel') ?? '',
            title: a?.getAttribute('title') ?? '',
            current: current && current !== 'false' ? current : '',
            lazy: li.hasAttribute('data-lazy'),
            open: li.hasAttribute('data-open'),
            kids: kids ? read(kids, `${path}/${label}`) : undefined,
        };
    });
    return read(top, '');
};
const styles = /* css */ `
:host {
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-muted, #7785A8);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_sel: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_brand: var(--sb-brand, #8C6BFF);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_indent: 1.25rem;
	--_notch: var(--sb-notch, 1);
	display: block;
	color: var(--_text);
	font-size: 0.875rem;
}
:host([hidden]) { display: none; }
[role="tree"] { display: grid; gap: 1px; outline: none; }
.row {
	display: flex;
	align-items: center;
	gap: 0.4rem;
	min-block-size: 2rem;
	padding-inline: calc(var(--depth) * var(--_indent) + 0.25rem) 0.5rem;
	border-radius: var(--sb-radius-sm, 6px);
	color: inherit;
	text-decoration: none;
	cursor: pointer;
	user-select: none;
}
.row:hover { background: var(--_hover); }
.row:is([aria-selected="true"], [aria-current]) { background: var(--_sel); box-shadow: inset 2px 0 0 var(--_brand); }
.row[aria-current] { font-weight: 600; }
.row:focus-visible { outline: 2px solid var(--_focus); outline-offset: -2px; }
.caret { display: grid; place-items: center; inline-size: 1rem; block-size: 1rem; flex: none; color: var(--_muted); }
/* A triangle that turns when open: stepped and turning in two steps at notch 1,
   plain and turning smoothly at 0. */
.caret::before {
	content: "";
	inline-size: 6px;
	block-size: 8px;
	background: currentColor;
	clip-path: polygon(0px 0px, calc(0px + 2px * var(--_notch)) 0px, calc(1.5px + 0.5px * var(--_notch)) 1px, calc(1.5px + 2.5px * var(--_notch)) 1px, calc(4.5px + -0.5px * var(--_notch)) 3px, 6px calc(4px + -1px * var(--_notch)), 6px calc(4px + 1px * var(--_notch)), calc(4.5px + -0.5px * var(--_notch)) 5px, calc(1.5px + 2.5px * var(--_notch)) 7px, calc(1.5px + 0.5px * var(--_notch)) 7px, calc(0px + 2px * var(--_notch)) 8px, 0px 8px);
	transition: rotate 120ms steps(calc(2 + 998 * (1 - var(--_notch))));
}
[aria-expanded="true"] > .caret::before { rotate: 90deg; }
:not([aria-expanded]) > .caret::before { display: none; }
.icon { flex: none; inline-size: 1.1rem; text-align: center; }
/* A copy of the markup's icon, at the size of the text icons. */
.art { display: grid; place-items: center; block-size: 1.1rem; }
.art > *, .art svg, .art img { display: block; inline-size: 100%; block-size: 100%; object-fit: contain; }
.label { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.meta {
	flex: none;
	margin-inline-start: auto;
	padding-inline-start: 0.5rem;
	color: var(--_muted);
	font-size: 0.75rem;
	font-weight: 400;
	font-variant-numeric: tabular-nums;
}
.busy { color: var(--_muted); font-size: 0.75rem; }
.busy::after { content: "…"; animation: dots 1s steps(3) infinite; display: inline-block; inline-size: 1.2em; overflow: hidden; vertical-align: bottom; }
@keyframes dots { from { inline-size: 0 } }
@media (prefers-reduced-motion: reduce) { .caret::before, .busy::after { transition: none; animation: none; } }
@media (forced-colors: active) {
	.caret::before { forced-color-adjust: none; background: CanvasText; }
	.row:is([aria-selected="true"], [aria-current]) { forced-color-adjust: none; background: Highlight; color: HighlightText; }
	.row:is([aria-selected="true"], [aria-current]) .caret::before { background: HighlightText; }
	.row:is([aria-selected="true"], [aria-current]) .meta { color: HighlightText; }
	.row:not([aria-selected="true"], [aria-current], :focus-visible):hover { outline: 1px solid Highlight; outline-offset: -1px; }
	.row:focus-visible { outline-color: Highlight; }
}
`;
rocket('sb-tree', {
    props: ({ bool, json, oneOf, string }) => ({
        items: json.default(() => []).docs({ description: 'The tree as JSON, for an element without a list inside: [{id, label, icon?, children?: [...], lazy?: true}]. A lazy item without children asks for them with sb-load when opened.' }),
        loaded: json.default(() => ({})).docs({ description: 'With items: loaded children by parent id, {"<id>": [items]}. Bind it to a signal the server patches (see the docs).' }),
        selection: oneOf('single', 'multiple', 'none').default('single').docs({ description: 'How many items can be selected. Links are followed, not selected.' }),
        value: string.docs({ description: 'Selection: an id, or ids separated by spaces (multiple). A new value from the server replaces it; the live value is the value property.' }),
        expanded: string.docs({ description: 'Open items: ids separated by spaces. A new list from the server replaces it; lazy items in it load their children.' }),
        label: string.trim.default('Tree').docs({ description: 'Accessible name.' }),
        confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
        name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
    }),
    manifest: {
        events: [
            { name: 'sb-load', kind: 'custom-event', bubbles: true, composed: true, description: 'A lazy item was opened and has no children yet. detail: { id }. Answer with a list in its <li>, or by adding its children to the loaded prop.' },
            { name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The selection changed.' },
            { name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'The selection changed. detail: { name, value } (an id, or an array of ids for multiple): ready for a command.' },
            { name: 'sb-toggle', kind: 'custom-event', bubbles: true, composed: true, description: 'An item opened or closed. detail: { id, open }.' },
        ],
    },
    // Rendered once: everything that changes goes through signals, so a new
    // `loaded` doesn't rebuild the DOM (and take the keyboard focus with it).
    renderOnPropChange: false,
    setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
        adoptStyles(host, styles);
        $$.label = props.label;
        $$.mode = props.selection;
        const ids = (v) => (Array.isArray(v) ? v.map(String) : String(v ?? '').split(/\s+/).filter(Boolean));
        $$.selected = ids(props.value);
        // Plain sets, not signals: assigning an object to a signal merges into
        // it (keys never go away), and only $$.rows is rendered anyway.
        const open = new Set(ids(props.expanded));
        const loading = new Set(); // ids whose children are on their way
        $$.focus = ''; // the row in the tab order (roving tabindex)
        $$.hasFocus = false; // whether keyboard focus is inside the tree
        $$.rows = [];
        // Move DOM focus to the focus row, also after the rows re-render (they
        // are new elements then, e.g. when a branch opens).
        const refocus = () => requestAnimationFrame(() => {
            if (!$$.hasFocus)
                return;
            // Only pick the focus back up when it fell on the floor (the row was
            // replaced or dropped): if the user moved on to something else, leave
            // it alone.
            const active = document.activeElement;
            if (active && active !== document.body && active !== host)
                return;
            const el = host.shadowRoot.querySelector(`[data-id="${CSS.escape($$.focus)}"]`);
            if (el && host.shadowRoot.activeElement !== el)
                el.focus();
        });
        // The tree: the element's own list wins over items. read() takes it in
        // and applies the markup's data-open and aria-current; rebuild() walks it.
        let tree = [];
        const art = new Map(); // the markup's icons by id
        const marked = new Map(); // data-open as last read, by id
        let current = '';
        const read = () => {
            tree = fromMarkup(host) ?? fromItems(props.items, props.loaded);
            art.clear();
            let path = [];
            const visit = (nodes, up) => nodes.forEach((n) => {
                if (n.art)
                    art.set(n.id, n.art);
                // A changed data-open wins, like a changed attribute; the same one
                // again keeps the user's choice.
                if (n.open !== null) {
                    const was = marked.get(n.id);
                    marked.set(n.id, n.open);
                    if (was === undefined ? n.open : was !== n.open)
                        n.open ? open.add(n.id) : open.delete(n.id);
                }
                if (n.current)
                    path = [...up, n.id];
                if (n.kids)
                    visit(n.kids, [...up, n.id]);
            });
            visit(tree, []);
            // A new current item opens the branches above it.
            const now = path.at(-1) ?? '';
            if (now !== current)
                path.slice(0, -1).forEach((id) => open.add(id));
            current = now;
        };
        // Rows are reused by position, so each one's copy of the markup's icon
        // is checked against the item it shows now.
        const painted = new WeakMap();
        const paint = () => host.shadowRoot?.querySelectorAll('.row').forEach((el) => {
            const box = el.querySelector('.art');
            const src = art.get(el.dataset.id ?? '');
            const key = src?.outerHTML ?? '';
            if (painted.get(box) === key)
                return;
            painted.set(box, key);
            box.replaceChildren(...(src ? [src.cloneNode(true)] : []));
        });
        // The visible rows: a depth-first walk through the open items.
        const rebuild = () => {
            const rows = [];
            const walk = (nodes, depth, parent) => {
                nodes.forEach((n, i) => {
                    const branch = !!(n.kids?.length || (n.lazy && !n.kids));
                    const isOpen = open.has(n.id) && branch;
                    // Children arrived: done loading. An open lazy item without
                    // children asks for them (also when the server opened it).
                    if (n.kids)
                        loading.delete(n.id);
                    else if (isOpen && n.lazy && !loading.has(n.id)) {
                        loading.add(n.id);
                        // The request goes out in a later task: during setup the page's
                        // data-on:sb-load isn't attached yet, and inside another effect
                        // (a morph) the @get it starts would be tracked by that effect.
                        setTimeout(() => emit('sb-load', { id: n.id }));
                    }
                    const { id, label, icon, meta, href, target, rel, title, current } = n;
                    rows.push({ id, label, icon, art: !!n.art, meta, href, target, rel, title, current, depth, parent, branch, open: isOpen, loading: loading.has(id), pos: i + 1, size: nodes.length });
                    if (isOpen && n.kids?.length)
                        walk(n.kids, depth + 1, id);
                });
            };
            walk(tree, 0, '');
            // A row the server dropped hands the focus to its neighbour, so the
            // keyboard stays in the tree instead of falling out to the document.
            const was = $$.rows.findIndex((r) => r.id === $$.focus);
            $$.rows = rows;
            if (!rows.some((r) => r.id === $$.focus))
                $$.focus = rows[Math.min(Math.max(was, 0), rows.length - 1)]?.id ?? '';
            // Inside another effect the rows render later: paint again then.
            paint();
            requestAnimationFrame(paint);
            refocus();
        };
        read();
        rebuild();
        // peek: attribute changes arrive inside the effect of whoever set them
        // (e.g. data-attr:loaded); reading signals here must not subscribe it.
        observeProps(() => peek(() => {
            $$.label = props.label;
            $$.mode = props.selection;
            read();
            rebuild();
        }));
        // Any change to the markup inside (a morph, a list of children patched
        // into an item) reads it again.
        const source = new MutationObserver((records) => {
            if (records.some((r) => r.target !== host || r.type !== 'attributes'))
                peek(() => (read(), rebuild()));
        });
        source.observe(host, { childList: true, subtree: true, characterData: true, attributes: true });
        cleanup(() => source.disconnect());
        // The server's value and expanded win whenever it sends new ones, also
        // ones the props already decode to (value="" on an element that never
        // had one: observeProps stays silent). The same attribute again keeps
        // the user's edits, and a removed one is ignored (morphs also strip
        // reflected attributes).
        const served = { value: host.getAttribute('value'), expanded: host.getAttribute('expanded') };
        const watch = new MutationObserver(() => peek(() => {
            for (const a in served) {
                const v = host.getAttribute(a);
                if (v === served[a])
                    continue;
                served[a] = v;
                if (v === null)
                    continue;
                if (a === 'value')
                    $$.selected = ids(v);
                else
                    (open.clear(), ids(v).forEach((id) => open.add(id)));
            }
            rebuild();
        }));
        watch.observe(host, { attributeFilter: ['value', 'expanded'] });
        cleanup(() => watch.disconnect());
        const value = () => (props.selection === 'multiple' ? [...$$.selected] : $$.selected[0] ?? '');
        overrideProp('value', () => peek(value), (v) => peek(() => ($$.selected = ids(v))));
        // Commands: the attribute is the server's value, $$.selected the local one
        // (compared as sets: in multiple mode the click order doesn't matter).
        // With confirm, :state(pending) marks an edit the server hasn't confirmed
        // yet; revert() returns to the server's value (e.g. a rejected command).
        const states = internalsOf(host).states;
        const key = (a) => a?.toSorted().join(' ');
        const sync = () => peek(() => (props.confirm && key($$.selected) !== key(ids(props.value)) ? states.add('pending') : states.delete('pending')));
        effect(() => (JSON.stringify($$.selected), sync()));
        observeProps(sync);
        defineHostProp('revert', { value: () => peek(() => (($$.selected = ids(props.value)), sync())) });
        const row = (id) => $$.rows.find((r) => r.id === id);
        // Tabbing in lands on the selected row, else the current page (the first
        // visible one). (rows?.: effects run once more with the signals gone
        // when the element is removed.)
        effect(() => {
            if ($$.hasFocus)
                return;
            const r = $$.rows?.find((r) => !r.href && $$.selected.includes(r.id)) ?? $$.rows?.find((r) => r.current);
            if (r)
                $$.focus = r.id;
        });
        const setOpen = (id, wantOpen) => {
            const r = row(id);
            if (!r?.branch || r.open === wantOpen)
                return;
            if (wantOpen)
                open.add(id);
            else
                open.delete(id);
            emit('sb-toggle', { id, open: wantOpen });
            rebuild();
        };
        const select = (id) => {
            if (props.selection === 'none')
                return;
            // Choosing the selected item again is no change.
            if (props.selection === 'single' && key($$.selected) === id)
                return;
            const has = $$.selected.includes(id);
            $$.selected = props.selection === 'multiple' ? (has ? $$.selected.filter((x) => x !== id) : [...$$.selected, id]) : [id];
            emit('change');
            emit('sb-change', { name: props.name, value: value() });
        };
        const focus = (id) => id && (($$.focus = id), refocus());
        // Focus can also arrive by Tab or a click: keep the roving focus in step.
        // The target is a row (their spans don't take focus), or the container
        // when a page makes ::part(tree) scroll (no data-id: nothing changes).
        action('focusin', ({ evt }) => {
            $$.hasFocus = true;
            const id = evt.target.dataset.id;
            if (id)
                $$.focus = id;
        });
        action('focusout', ({ evt }) => {
            // A row the server dropped also "loses" focus, but that's not leaving:
            // its focusout comes while it is still connected and without a
            // relatedTarget, just like a click on the page. So decide a frame
            // later: a removed row is gone by then (refocus() hands the focus to
            // its neighbour); otherwise the focus left, unless it is on another
            // row. (Keep evt.target: it is cleared after dispatch.)
            const t = evt.target;
            requestAnimationFrame(() => t.isConnected && !host.shadowRoot.activeElement && ($$.hasFocus = false));
        });
        // A link row is the browser's: a click or Enter follows it (with a
        // modifier, in a new tab). Only its caret opens and closes it.
        action('click', ({ evt }, id) => {
            $$.focus = id;
            const r = row(id);
            if (r?.branch && evt.target.closest('.caret')) {
                evt.preventDefault();
                return setOpen(id, !r.open);
            }
            if (r?.href)
                return;
            select(id);
            // A click also opens or closes a branch, except in multiple mode (setOpen skips leaves).
            if (props.selection !== 'multiple')
                setOpen(id, !r?.open);
        });
        action('key', ({ evt }) => {
            const e = evt;
            // Alt+Left is the browser's Back, Ctrl+Enter a link in a new tab.
            if (e.altKey || e.ctrlKey || e.metaKey)
                return;
            const rows = $$.rows;
            const i = rows.findIndex((r) => r.id === $$.focus);
            const r = rows[i];
            if (!r)
                return;
            switch (e.key) {
                case 'ArrowDown':
                    focus(rows[i + 1]?.id);
                    break;
                case 'ArrowUp':
                    focus(rows[i - 1]?.id);
                    break;
                case 'Home':
                    focus(rows[0].id);
                    break;
                case 'End':
                    focus(rows.at(-1).id);
                    break;
                case 'ArrowRight':
                    if (r.branch && !r.open)
                        setOpen(r.id, true);
                    else if (rows[i + 1]?.parent === r.id)
                        focus(rows[i + 1].id);
                    break;
                case 'ArrowLeft':
                    if (r.open)
                        setOpen(r.id, false);
                    else
                        focus(r.parent);
                    break;
                case 'Enter':
                    if (r.href)
                        return;
                    select(r.id);
                    break;
                case ' ':
                    if (r.href)
                        e.target.click();
                    else
                        select(r.id);
                    break;
                default:
                    return;
            }
            e.preventDefault();
        });
    },
    render: ({ html }) => html `
		<div role="tree" part="tree"
			data-attr:aria-label="$$label"
			data-attr:aria-multiselectable="$$mode === 'multiple' && 'true'"
			data-on:keydown="@key()" data-on:focusin="@focusin()" data-on:focusout="@focusout()">
			<!-- r?.: when the list shrinks, data-for can re-evaluate a removed row once with r undefined.
			     Every row is an <a>: a link with an href, a plain item without. -->
			<template data-for="r in $$rows">
				<a role="treeitem" class="row"
					data-attr:part="['item', $$mode !== 'none' && !r?.href && $$selected.includes(r?.id) && 'selected', r?.current && 'current'].filter(Boolean).join(' ')"
					data-attr:href="r?.href || null"
					data-attr:target="(r?.href && r?.target) || null"
					data-attr:rel="(r?.href && r?.rel) || null"
					data-attr:title="r?.title || null"
					data-attr:aria-current="r?.current || null"
					data-attr:data-id="r?.id"
					data-attr:aria-level="r?.depth + 1"
					data-attr:aria-posinset="r?.pos"
					data-attr:aria-setsize="r?.size"
					data-attr:aria-expanded="r?.branch && String(r?.open)"
					data-attr:aria-selected="$$mode !== 'none' && !r?.href && String($$selected.includes(r?.id))"
					data-attr:aria-busy="r?.loading && 'true'"
					data-attr:tabindex="r?.id === $$focus ? 0 : -1"
					data-style:--depth="r?.depth"
					data-on:click="@click(r?.id)">
					<span class="caret" part="caret" aria-hidden="true"></span>
					<span class="icon" part="icon" aria-hidden="true" data-show="r?.icon" data-text="r?.icon"></span>
					<span class="icon art" part="icon" aria-hidden="true" data-show="r?.art"></span>
					<span class="label" part="label" data-text="r?.label"></span>
					<span class="busy" data-show="r?.loading">Loading</span>
					<span class="meta" part="meta" data-show="r?.meta" data-text="r?.meta"></span>
				</a>
			</template>
		</div>
	`,
});
