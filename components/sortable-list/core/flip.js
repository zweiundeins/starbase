// Generated from flip.ts by `go tool task ts`: edit the TypeScript, not this file.
/*!
 * From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), vendored by
 * `go run ./cmd/vendorpd` with patches/pd-rockets applied, pd- names renamed to sb-.
 *
 * THE BEER-WARE LICENSE (Revision 42)
 *
 * PD rockets contributors wrote this software. As long as you retain this notice,
 * you can do whatever you want with it. If we meet someday and you think this
 * software is worth it, you can buy us a beer in return.
 */
function capture(options) {
    const result = new Map();
    options.host.querySelectorAll(options.itemSelector).forEach((item) => {
        const id = options.itemId(item);
        if (id) {
            const rect = item.getBoundingClientRect();
            result.set(id, { left: rect.left, top: rect.top });
        }
    });
    return result;
}
// Where each item sits in the layout. Transforms don't count: a projection or a running animation moves nothing.
function places(options) {
    const result = new Map();
    options.host.querySelectorAll(options.itemSelector).forEach((item) => {
        const id = options.itemId(item);
        if (id)
            result.set(id, `${item.offsetLeft} ${item.offsetTop}`);
    });
    return result;
}
function play(options, before) {
    if (matchMedia("(prefers-reduced-motion: reduce)").matches)
        return;
    options.host.querySelectorAll(options.itemSelector).forEach((item) => {
        const id = options.itemId(item);
        const first = id ? before.get(id) : undefined;
        if (!first)
            return;
        const last = item.getBoundingClientRect();
        const dx = first.left - last.left;
        const dy = first.top - last.top;
        if (Math.abs(dx) < 1 && Math.abs(dy) < 1)
            return;
        item.animate([{ transform: `translate(${dx}px, ${dy}px)` }, { transform: "translate(0, 0)" }], {
            duration: 180,
            easing: "cubic-bezier(.2, 0, 0, 1)",
        });
    });
}
/** Watches a host for the DOM change caused by a semantic move, including a later SSE morph. */
export function installFlip(options) {
    let before = null;
    let placed = null;
    let addedOrRemoved = false;
    let timer = null;
    let frame = null;
    const clearPending = () => {
        before = null;
        placed = null;
        addedOrRemoved = false;
        if (timer !== null)
            clearTimeout(timer);
        if (frame !== null)
            cancelAnimationFrame(frame);
        timer = null;
        frame = null;
    };
    // A patch may move items through attributes alone (a grid placement, ids on elements reused in place): without items
    // added or removed, play once one has moved in the layout, so a change that moves none keeps waiting.
    const moved = () => {
        const now = places(options);
        return now.size !== placed?.size || [...now].some(([id, place]) => placed?.get(id) !== place);
    };
    const finish = () => {
        frame = null;
        if (!before || (!addedOrRemoved && !moved()))
            return;
        const snapshot = before;
        clearPending();
        play(options, snapshot);
    };
    const containsItem = (node) => node instanceof Element && (node.matches(options.itemSelector) || !!node.querySelector(options.itemSelector));
    const observer = new MutationObserver((records) => {
        if (!before)
            return;
        if (records.some((record) => [...record.addedNodes, ...record.removedNodes].some(containsItem)))
            addedOrRemoved = true;
        if (frame === null)
            frame = requestAnimationFrame(finish);
    });
    observer.observe(options.host, { childList: true, attributes: true, subtree: true });
    return {
        prepare: (origin) => {
            clearPending();
            before = capture(options);
            if (origin)
                before.set(origin.itemId, origin.rect);
            placed = places(options);
            timer = setTimeout(clearPending, 2000);
        },
        dispose: () => {
            observer.disconnect();
            clearPending();
        },
    };
}
