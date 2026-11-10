// Generated from landing.ts by `go tool task ts`: edit the TypeScript, not this file.
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
import { kanbanContract } from "./contracts/kanban.js";
const reducedMotion = () => matchMedia("(prefers-reduced-motion: reduce)").matches;
const easing = "cubic-bezier(.2, 0, 0, 1)";
/** The marker: positioned inline, its look from custom properties a page can set, with a forced-colours default. */
export function landingMarker(from, kind) {
    const marker = from ? from.cloneNode(true) : document.createElement("div");
    marker.removeAttribute("id");
    marker.querySelectorAll("[id]").forEach((element) => element.removeAttribute("id"));
    for (const name of ["tabindex", "data-dragging", "data-drop-before", "data-lane-dragging", "data-lane-drop-target"])
        marker.removeAttribute(name);
    marker.setAttribute("data-landing-marker", kind);
    marker.setAttribute("aria-hidden", "true");
    marker.inert = true;
    marker.style.position = "fixed";
    marker.style.boxSizing = "border-box";
    marker.style.margin = "0";
    marker.style.pointerEvents = "none";
    const forced = matchMedia("(forced-colors: active)").matches;
    marker.style.setProperty("opacity", `var(--sb-landing-opacity, ${forced ? 1 : 0.5})`);
    marker.style.setProperty("outline", `var(--sb-landing-outline, 2px dashed ${forced ? "Highlight" : "currentColor"})`);
    marker.style.setProperty("outline-offset", "-2px");
    return marker;
}
/** The ancestors of `element` that clip it when they scroll. */
function scrollers(element) {
    const result = [];
    for (let current = element.parentElement; current && current !== document.documentElement;) {
        const style = getComputedStyle(current);
        if (style.overflowX !== "visible" || style.overflowY !== "visible")
            result.push(current);
        current = current.parentElement;
    }
    return result;
}
/** The part of `element` its scrolling ancestors leave visible. */
function visibleArea(element, ancestors) {
    let { left, top, right, bottom } = element.getBoundingClientRect();
    for (const ancestor of ancestors) {
        const rect = ancestor.getBoundingClientRect();
        left = Math.max(left, rect.left);
        top = Math.max(top, rect.top);
        right = Math.min(right, rect.right);
        bottom = Math.min(bottom, rect.bottom);
    }
    return { left, top, width: Math.max(0, right - left), height: Math.max(0, bottom - top) };
}
/**
 * Pending landings: each marker lives in <body>, out of reach of the morphs that patch the board. A landing does its
 * work only when something can have moved its slot: a change in the board, scrolling, resizing, or an animation there.
 */
export function installLandings(options) {
    const { host } = options;
    const pending = new Map();
    let frame = null;
    let rearm = null;
    let dirty = false;
    let settled = true;
    let armedAt = 0;
    const schedule = () => {
        if (frame === null && pending.size)
            frame = requestAnimationFrame(tick);
    };
    const wake = () => {
        dirty = true;
        schedule();
    };
    const changes = new MutationObserver(wake);
    const sizes = new ResizeObserver(wake);
    let watching = false;
    const watch = (on) => {
        if (on === watching)
            return;
        watching = on;
        if (on) {
            changes.observe(host, { childList: true, subtree: true, attributes: true, characterData: true });
            sizes.observe(host);
            addEventListener("scroll", wake, { capture: true, passive: true });
            addEventListener("resize", wake);
            return;
        }
        changes.disconnect();
        sizes.disconnect();
        removeEventListener("scroll", wake, { capture: true });
        removeEventListener("resize", wake);
        if (rearm !== null)
            clearTimeout(rearm);
        rearm = null;
    };
    const setGap = (entry, want) => {
        const current = entry.gap;
        if (current && want && current.element === want.element && current.side === want.side && current.size === want.size)
            return;
        current?.animation.cancel();
        entry.gap = null;
        if (!want)
            return;
        const property = want.side === "top" ? "marginTop" : "marginBottom";
        const base = parseFloat(getComputedStyle(want.element)[property]) || 0;
        const animation = want.element.animate([{ [property]: `${base}px` }, { [property]: `${base + want.size}px` }], {
            duration: current || reducedMotion() ? 0 : 150,
            easing,
            fill: "forwards",
        });
        entry.gap = { ...want, animation };
    };
    const closeGap = (entry, smooth) => {
        const animation = entry.gap?.animation;
        entry.gap = null;
        if (!animation)
            return;
        if (!smooth || reducedMotion()) {
            animation.cancel();
            return;
        }
        animation.reverse();
        animation.finished.then(() => animation.cancel()).catch(() => { });
    };
    const end = (entry, reason, notify = true) => {
        if (pending.get(entry.key) !== entry)
            return;
        pending.delete(entry.key);
        clearTimeout(entry.timer);
        entry.spec?.marker.remove();
        closeGap(entry, reason !== "arrived");
        if (!pending.size)
            watch(false);
        if (notify)
            options.onEnd({ ...entry.detail, reason });
    };
    const place = (entry, marker, rect, view) => {
        // A morph of the whole page removes it with everything else the server didn't render: put it back.
        if (!marker.isConnected)
            document.body.append(marker);
        marker.style.left = `${rect.left}px`;
        marker.style.top = `${rect.top}px`;
        marker.style.width = `${rect.width}px`;
        marker.style.height = `${rect.height}px`;
        const clip = {
            top: Math.max(0, view.top - rect.top),
            right: Math.max(0, rect.left + rect.width - (view.left + view.width)),
            bottom: Math.max(0, rect.top + rect.height - (view.top + view.height)),
            left: Math.max(0, view.left - rect.left),
        };
        marker.style.clipPath =
            clip.top || clip.right || clip.bottom || clip.left
                ? `inset(${clip.top}px ${clip.right}px ${clip.bottom}px ${clip.left}px)`
                : "";
        marker.style.visibility = "";
        if (entry.placed || !entry.spec)
            return;
        entry.placed = true;
        const dx = entry.spec.origin.left - rect.left;
        const dy = entry.spec.origin.top - rect.top;
        if (!reducedMotion() && (Math.abs(dx) >= 1 || Math.abs(dy) >= 1))
            marker.animate([{ transform: `translate(${dx}px, ${dy}px)` }, { transform: "none" }], { duration: 150, easing });
    };
    // Reads first, then writes, so a tick forces at most the one layout the frame needs anyway. A move that arrived ends
    // its landing; the others follow their slot. Arming the move's animation lets it start from the marker when the
    // server's answer lands: again after each change, once animations settle, and each second for its timer.
    const tick = () => {
        frame = null;
        if (rearm !== null)
            clearTimeout(rearm);
        rearm = null;
        const changed = dirty;
        dirty = false;
        const animating = host.getAnimations({ subtree: true }).some((animation) => animation.playState === "running");
        const arm = changed || (!animating && !settled) || performance.now() - armedAt > 1000;
        settled = !animating;
        const plans = [];
        const armed = new Set();
        for (const entry of pending.values()) {
            if (!entry.ready)
                continue;
            const fresh = !entry.spec;
            if (fresh) {
                entry.spec = entry.build();
                if (!entry.spec || entry.spec.arrived()) {
                    plans.push({ entry, arrived: true });
                    continue;
                }
            }
            const spec = entry.spec;
            if (!fresh && changed && spec.arrived()) {
                plans.push({ entry, arrived: true });
                armed.add(spec.group);
                continue;
            }
            if (!fresh && !changed && !animating && !arm)
                continue;
            const where = spec.place();
            let view;
            if (where) {
                if (entry.clip?.element !== where.clip)
                    entry.clip = { element: where.clip, ancestors: scrollers(where.clip) };
                view = visibleArea(where.clip, entry.clip.ancestors);
            }
            plans.push({ entry, where, view });
        }
        for (const { entry, where } of plans) {
            // A new landing that opens a gap arms the animation once the gap is open; the drop armed it until then.
            if (!where || !arm || !entry.spec || armed.has(entry.spec.group) || (!entry.placed && where.gap))
                continue;
            armed.add(entry.spec.group);
            entry.spec.prepare(where.rect);
            armedAt = performance.now();
        }
        for (const { entry, arrived, where, view } of plans) {
            if (arrived) {
                end(entry, "arrived");
                continue;
            }
            const marker = entry.spec.marker;
            if (where === undefined)
                continue;
            if (!where || !view) {
                if (!marker.isConnected)
                    document.body.append(marker);
                marker.style.visibility = "hidden";
                setGap(entry, null);
                continue;
            }
            const opened = !entry.gap && where.gap;
            setGap(entry, where.gap);
            place(entry, marker, where.rect, view);
            if (opened)
                settled = false;
        }
        if (!pending.size)
            return;
        if (animating || !settled)
            schedule();
        else
            rearm = setTimeout(schedule, 1000);
    };
    return {
        /** A landing for `key`, built a frame after the drop: the drop's own frame stays as light as without one. */
        add(key, detail, build, timeout) {
            const previous = pending.get(key);
            if (previous)
                end(previous, "released");
            const entry = {
                key,
                detail,
                build,
                spec: null,
                ready: false,
                placed: false,
                gap: null,
                clip: null,
                timer: setTimeout(() => end(entry, "timeout"), timeout),
            };
            pending.set(key, entry);
            requestAnimationFrame(() => {
                if (pending.get(key) !== entry)
                    return;
                entry.ready = true;
                watch(true);
                wake();
            });
        },
        /** Ends landings early: the one of a card ({ cardId }), of a lane ({ col }), or all; true when one was pending. */
        release(match) {
            const ended = [...pending.values()].filter(({ detail }) => match?.cardId !== undefined
                ? detail.cardId === match.cardId
                : match?.col !== undefined
                    ? detail.cardId === undefined && detail.col === match.col
                    : true);
            ended.forEach((entry) => end(entry, "released"));
            return ended.length > 0;
        },
        dispose() {
            if (frame !== null)
                cancelAnimationFrame(frame);
            frame = null;
            [...pending.values()].forEach((entry) => end(entry, "released", false));
            watch(false);
        },
    };
}
const cardId = (card) => card?.dataset.kanbanCard ?? "";
/** Where a card is: its lane's data-col and the card after it ("" for the last). */
export function cardPlace(geometry, card) {
    const lane = card.closest(kanbanContract.selectors.lane);
    const siblings = lane ? geometry.cardsIn(lane) : [];
    return { col: lane?.dataset.col, next: cardId(siblings[siblings.indexOf(card) + 1]) };
}
/** The landing of a card dropped from `from` into lane `col` before card `before` ("" for the end); null in place. */
export function cardLanding(geometry, card, from, target, origin, prepare, group) {
    const id = cardId(card);
    const col = String(target.col);
    if (from.col === col && from.next === target.before)
        return null;
    const height = card.getBoundingClientRect().height;
    return {
        marker: landingMarker(card, "card"),
        origin,
        place: () => {
            const lane = geometry.lanes().find((candidate) => candidate.dataset.col === col);
            if (!lane)
                return null;
            const list = lane.querySelector("[data-kanban-lane-cards]") ?? lane;
            const inLane = geometry.cardsIn(lane);
            const gap = parseFloat(getComputedStyle(list).rowGap) || 0;
            const before = target.before ? inLane.find((other) => cardId(other) === target.before) : undefined;
            if (before) {
                const previous = inLane[inLane.indexOf(before) - 1];
                const rect = before.getBoundingClientRect();
                const top = previous ? previous.getBoundingClientRect().bottom + gap : contentTop(list);
                return {
                    rect: { left: rect.left, top, width: rect.width, height },
                    gap: { element: before, side: "top", size: height + gap },
                    clip: list,
                };
            }
            const last = inLane.filter((other) => cardId(other) !== id).at(-1);
            if (last) {
                const rect = last.getBoundingClientRect();
                return {
                    rect: { left: rect.left, top: rect.bottom + gap, width: rect.width, height },
                    gap: { element: last, side: "bottom", size: height + gap },
                    clip: list,
                };
            }
            const box = contentBox(list);
            return { rect: { ...box, height }, gap: null, clip: list };
        },
        arrived: () => {
            const now = geometry.card(id);
            const lane = now?.closest(kanbanContract.selectors.lane);
            if (!now || lane?.dataset.col !== col)
                return false;
            const inLane = geometry.cardsIn(lane);
            return from.col !== col || cardId(inLane[inLane.indexOf(now) + 1]) !== from.next;
        },
        prepare: (rect) => prepare(id, rect),
        group,
    };
}
/** The landing of a lane moved into `target`'s place, before lane `before` ("" for the end). */
export function laneLanding(geometry, lane, target, before, origin, prepare, group) {
    const col = lane.dataset.col ?? "";
    const targetCol = target.dataset.col ?? "";
    return {
        marker: landingMarker(null, "lane"),
        origin,
        place: () => {
            const slot = geometry.lanes().find((candidate) => candidate.dataset.col === targetCol);
            if (!slot?.parentElement)
                return null;
            const { left, top, width, height } = slot.getBoundingClientRect();
            return { rect: { left, top, width, height }, gap: null, clip: slot.parentElement };
        },
        arrived: () => {
            const all = geometry.lanes();
            const index = all.findIndex((candidate) => candidate.dataset.col === col);
            return index >= 0 && (all[index + 1]?.dataset.col ?? "") === before;
        },
        prepare: (rect) => prepare(col, rect),
        group,
    };
}
function contentBox(element) {
    const rect = element.getBoundingClientRect();
    const style = getComputedStyle(element);
    const left = rect.left + (parseFloat(style.borderLeftWidth) || 0) + (parseFloat(style.paddingLeft) || 0);
    const right = rect.right - (parseFloat(style.borderRightWidth) || 0) - (parseFloat(style.paddingRight) || 0);
    return { left, top: contentTop(element), width: Math.max(0, right - left), height: 0 };
}
function contentTop(element) {
    const style = getComputedStyle(element);
    return (element.getBoundingClientRect().top + (parseFloat(style.borderTopWidth) || 0) + (parseFloat(style.paddingTop) || 0));
}
