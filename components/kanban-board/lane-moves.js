// Generated from lane-moves.ts by `go tool task ts`: edit the TypeScript, not this file.
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
import { dragPreviewFor } from "./core/visual-outlets.js";
import { kanbanContract } from "./contracts/kanban.js";
/** Where a lane lands when it takes `target`'s place: before that data-col, "" for the end, or null for no move. */
export function laneBefore(lanes, lane, target) {
    const from = lanes.indexOf(lane);
    const to = lanes.indexOf(target);
    if (from < 0 || to < 0 || from === to)
        return null;
    return from > to ? (target.dataset.col ?? "") : (lanes[to + 1]?.dataset.col ?? "");
}
/** Opt-in lane moves: a pointer drag by a lane's grip, and its step buttons. The page renders both. */
export function installLaneMoves(options) {
    const { host, owns } = options;
    const { lane: laneSelector, laneGrip, laneStep } = kanbanContract.selectors;
    let press = null;
    let drag = null;
    const mark = (source, target) => {
        for (const lane of options.lanes()) {
            lane.toggleAttribute("data-lane-dragging", lane === source);
            lane.toggleAttribute("data-lane-drop-target", lane === target && lane !== source);
        }
    };
    const nearest = (x, y, source) => {
        const box = host.getBoundingClientRect();
        if (x < box.left || x > box.right || y < box.top || y > box.bottom)
            return source;
        let best = source;
        let distance = Infinity;
        for (const lane of options.lanes()) {
            const rect = lane.getBoundingClientRect();
            const next = Math.hypot(Math.max(rect.left - x, 0, x - rect.right), Math.max(rect.top - y, 0, y - rect.bottom));
            if (next < distance) {
                distance = next;
                best = lane;
            }
        }
        return best;
    };
    const start = (pointer, lane, from) => {
        const rect = lane.getBoundingClientRect();
        const custom = !!lane.querySelector(":scope > template[data-sb-preview]");
        const preview = dragPreviewFor(lane);
        preview.removeAttribute("id");
        preview.querySelectorAll("[id]").forEach((element) => element.removeAttribute("id"));
        preview.setAttribute("data-drag-preview", "true");
        preview.setAttribute("aria-hidden", "true");
        preview.inert = true;
        preview.style.position = "fixed";
        preview.style.left = `${rect.left}px`;
        preview.style.top = `${rect.top}px`;
        preview.style.boxSizing = "border-box";
        if (!custom) {
            preview.style.width = `${rect.width}px`;
            preview.style.height = `${rect.height}px`;
        }
        preview.style.pointerEvents = "none";
        preview.style.zIndex = "1000";
        document.body.append(preview);
        drag = { preview, offset: { x: from.x - rect.left, y: from.y - rect.top }, target: lane };
        host.setPointerCapture?.(pointer.pointerId);
        mark(lane, lane);
    };
    const finish = (commit) => {
        const current = drag;
        const pressed = press;
        press = null;
        drag = null;
        if (!current || !pressed)
            return false;
        mark(null, null);
        const rect = { left: parseFloat(current.preview.style.left), top: parseFloat(current.preview.style.top) };
        current.preview.remove();
        try {
            host.releasePointerCapture?.(pressed.pointerId);
        }
        catch {
            // Pointer capture may already have been released by the browser.
        }
        if (commit && current.target !== pressed.lane)
            options.commit(pressed.lane, current.target, rect);
        return true;
    };
    const onDown = (event) => {
        if (event.button !== 0 || press || !(event.target instanceof HTMLElement))
            return;
        const grip = event.target.closest(laneGrip);
        const lane = grip?.closest(laneSelector);
        if (!grip || !lane || !owns(grip) || !owns(lane))
            return;
        press = { pointerId: event.pointerId, x: event.clientX, y: event.clientY, lane };
    };
    const onMove = (event) => {
        if (!press || event.pointerId !== press.pointerId)
            return;
        if (!drag) {
            if (Math.hypot(event.clientX - press.x, event.clientY - press.y) < 5)
                return;
            start(event, press.lane, press);
        }
        if (!drag)
            return;
        drag.preview.style.left = `${event.clientX - drag.offset.x}px`;
        drag.preview.style.top = `${event.clientY - drag.offset.y}px`;
        const target = nearest(event.clientX, event.clientY, press.lane);
        if (target === drag.target)
            return;
        drag.target = target;
        mark(press.lane, target);
    };
    const onUp = (event) => {
        if (press?.pointerId === event.pointerId)
            finish(true);
    };
    const onCancel = (event) => {
        if (press?.pointerId === event.pointerId)
            finish(false);
    };
    const onDragStart = (event) => {
        if (press)
            event.preventDefault();
    };
    const onClick = (event) => {
        if (!(event.target instanceof HTMLElement))
            return;
        const button = event.target.closest(laneStep);
        const lane = button?.closest(laneSelector);
        if (!button || !lane || !owns(button) || !owns(lane) || button.matches(":disabled"))
            return;
        const step = Number(button.dataset.kanbanLaneStep);
        const lanes = options.lanes();
        const target = step === -1 || step === 1 ? lanes[lanes.indexOf(lane) + step] : undefined;
        if (target)
            options.commit(lane, target);
    };
    host.addEventListener("pointerdown", onDown);
    host.addEventListener("dragstart", onDragStart);
    host.addEventListener("click", onClick);
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onCancel);
    return {
        mark,
        /** Ends a pointer drag without a move; true when one was running. */
        cancel: () => finish(false),
        dispose() {
            finish(false);
            host.removeEventListener("pointerdown", onDown);
            host.removeEventListener("dragstart", onDragStart);
            host.removeEventListener("click", onClick);
            window.removeEventListener("pointermove", onMove);
            window.removeEventListener("pointerup", onUp);
            window.removeEventListener("pointercancel", onCancel);
        },
    };
}
