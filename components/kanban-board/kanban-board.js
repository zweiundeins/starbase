// Generated from kanban-board.ts by `go tool task ts`: edit the TypeScript, not this file.
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
import { rocket } from "datastar";
import { installFlip } from "./core/flip.js";
import { installFocusRecovery } from "./core/focus-recovery.js";
import { insertionBefore } from "./core/insertion-target.js";
import { keyboardBindings, keyboardItem, platformFocusKeys } from "./core/keyboard.js";
import { installKeyboardStaging } from "./core/keyboard-staging.js";
import { markRocketHost, ownsRocketElement } from "./core/ownership.js";
import { installPointerDrag } from "./core/pointer-drag.js";
import { installTargetIndicator } from "./core/visual-outlets.js";
import { kanbanContract, defaultKanbanKeyboard, } from "./contracts/kanban.js";
import { installLaneMoves, laneBefore } from "./lane-moves.js";
rocket("sb-kanban-board", {
    mode: "light",
    manifest: {
        events: [
            {
                name: kanbanContract.events.move,
                kind: "custom-event",
                bubbles: true,
                composed: true,
                description: "A card was dropped, or a keyboard move committed. detail: { cardId, col, before }: col is the lane's " +
                    'data-col as a number, before the id of the card it now precedes in that lane, or "" for the end.',
            },
            {
                name: kanbanContract.events.select,
                kind: "custom-event",
                bubbles: true,
                composed: true,
                description: "A card was selected: arrow focus moved to it, or the pointer pressed it. detail: { cardId }.",
            },
            {
                name: kanbanContract.events.laneMove,
                kind: "custom-event",
                bubbles: true,
                composed: true,
                description: "A lane was dropped by its grip, moved with Alt and the arrows, or stepped. detail: { col, before }: " +
                    'col is its data-col as a number, before the data-col of the lane it now precedes, or "" for the end.',
            },
        ],
    },
    setup({ host, cleanup }) {
        cleanup(markRocketHost(host));
        const owns = (element) => ownsRocketElement(host, element);
        const { focusNext, focusPrevious, focusFirst, focusLast } = platformFocusKeys(navigator.platform);
        const keyboard = keyboardBindings(host, {
            focusNext,
            focusPrevious,
            focusLeft: defaultKanbanKeyboard.selectLeft,
            focusRight: defaultKanbanKeyboard.selectRight,
            focusFirst,
            focusLast,
            moveUp: defaultKanbanKeyboard.moveUp,
            moveDown: defaultKanbanKeyboard.moveDown,
            moveLeft: defaultKanbanKeyboard.moveLeft,
            moveRight: defaultKanbanKeyboard.moveRight,
            moveFirst: defaultKanbanKeyboard.moveFirst,
            moveLast: defaultKanbanKeyboard.moveLast,
            cancel: defaultKanbanKeyboard.cancel,
        }, {
            focusNext: "selectNext",
            focusPrevious: "selectPrevious",
            focusLeft: "selectLeft",
            focusRight: "selectRight",
        });
        const cards = () => [...host.querySelectorAll(kanbanContract.selectors.card)].filter(owns);
        const lanes = () => [...host.querySelectorAll(kanbanContract.selectors.lane)].filter(owns);
        const cardsIn = (lane) => [...lane.querySelectorAll(kanbanContract.selectors.card)].filter(owns);
        const focus = installFocusRecovery(host);
        const flip = installFlip({
            host,
            itemSelector: kanbanContract.selectors.card,
            itemId: (card) => (owns(card) ? (card.dataset.kanbanCard ?? null) : null),
        });
        const indicator = installTargetIndicator(host);
        const clearMarks = () => {
            indicator.clear();
            cards().forEach((card) => card.removeAttribute("data-drop-before"));
            lanes().forEach((lane) => {
                lane.removeAttribute("data-drop-active");
                lane.querySelector("[data-kanban-lane-cards]")?.removeAttribute("data-drop-end");
            });
        };
        const clearDragging = () => cards().forEach((card) => card.removeAttribute("data-dragging"));
        const markTarget = (target) => {
            clearMarks();
            if (!target)
                return;
            target.lane.setAttribute("data-drop-active", "true");
            if (target.before) {
                const beforeCard = cardsIn(target.lane).find((card) => card.dataset.kanbanCard === target.before);
                beforeCard?.setAttribute("data-drop-before", "");
                indicator.show(beforeCard ?? null, "before");
            }
            else {
                const end = target.lane.querySelector("[data-kanban-lane-cards]");
                end?.setAttribute("data-drop-end", "");
                indicator.show(end ?? null, "end");
            }
        };
        const isCard = (element) => element.matches(kanbanContract.selectors.card);
        // A lane the page made focusable (tabindex) is a focus stop while it has no cards.
        const emptyStop = (lane) => lane.hasAttribute("tabindex") && cardsIn(lane).length === 0;
        const emitSelect = (card) => {
            host.dispatchEvent(new CustomEvent(kanbanContract.events.select, {
                bubbles: true,
                composed: true,
                detail: { cardId: card.dataset.kanbanCard ?? "" },
            }));
        };
        const select = (stop) => {
            if (!stop)
                return;
            stop.focus();
            if (isCard(stop))
                emitSelect(stop);
        };
        const onPointerDown = (event) => {
            const target = event.target;
            if (event.button !== 0 || !(target instanceof HTMLElement) || !owns(target))
                return;
            if (target.closest("button, a, input, select, textarea, [contenteditable]:not([contenteditable='false'])") &&
                !target.closest(kanbanContract.selectors.cardMain))
                return;
            const card = target.closest(kanbanContract.selectors.card);
            if (card && owns(card))
                emitSelect(card);
        };
        const targetAt = (x, y, itemId) => {
            const lane = document.elementFromPoint(x, y)?.closest(kanbanContract.selectors.lane);
            if (!lane || !owns(lane))
                return null;
            const candidates = cardsIn(lane)
                .filter((card) => card.dataset.kanbanCard !== itemId)
                .map((card) => {
                const rect = card.getBoundingClientRect();
                return { id: card.dataset.kanbanCard ?? "", top: rect.top, bottom: rect.bottom };
            });
            const before = insertionBefore(candidates, y);
            return { col: Number(lane.dataset.col ?? 0), before, lane };
        };
        const emitMove = (itemId, target) => {
            host.dispatchEvent(new CustomEvent(kanbanContract.events.move, {
                bubbles: true,
                composed: true,
                detail: { cardId: itemId, col: target.col, before: target.before },
            }));
        };
        const staging = installKeyboardStaging({
            host,
            owns,
            onCancel: clearMarks,
            onCommit: ({ itemId, target }) => {
                clearMarks();
                const source = cards().find((card) => card.dataset.kanbanCard === itemId);
                if (source)
                    focus.expect(source, () => {
                        const card = cards().find((candidate) => candidate.dataset.kanbanCard === itemId);
                        const lane = card?.closest(kanbanContract.selectors.lane);
                        if (!card || !lane || Number(lane.dataset.col) !== target.col)
                            return null;
                        const siblings = cardsIn(lane);
                        return (siblings[siblings.indexOf(card) + 1]?.dataset.kanbanCard ?? "") === target.before ? card : null;
                    });
                flip.prepare();
                emitMove(itemId, target);
            },
        });
        const pointerDispose = installPointerDrag({
            host,
            itemSelector: kanbanContract.selectors.card,
            interactiveHandle: kanbanContract.selectors.cardMain,
            itemId: (card) => (owns(card) ? (card.dataset.kanbanCard ?? null) : null),
            targetAt,
            sameTarget: (a, b) => a?.lane === b?.lane && a?.before === b?.before,
            mark: markTarget,
            beforeCommit: (id, rect) => flip.prepare({ itemId: id, rect }),
            commit: emitMove,
        });
        const laneFlip = installFlip({
            host,
            itemSelector: kanbanContract.selectors.lane,
            itemId: (lane) => (owns(lane) ? (lane.dataset.col ?? null) : null),
        });
        const emitLaneMove = (lane, target, rect) => {
            const col = lane.dataset.col ?? "";
            const before = laneBefore(lanes(), lane, target);
            if (before === null || col === "" || !Number.isFinite(Number(col)))
                return false;
            laneFlip.prepare(rect ? { itemId: col, rect } : undefined);
            host.dispatchEvent(new CustomEvent(kanbanContract.events.laneMove, {
                bubbles: true,
                composed: true,
                detail: { col: Number(col), before },
            }));
            return true;
        };
        const laneMoves = installLaneMoves({ host, lanes, owns, commit: emitLaneMove });
        const laneStaging = installKeyboardStaging({
            host,
            owns,
            onCancel: () => laneMoves.mark(null, null),
            onCommit: ({ lane, target, grip }) => {
                laneMoves.mark(null, null);
                const col = lane.dataset.col;
                const before = laneBefore(lanes(), lane, target);
                if (before === null)
                    return;
                focus.expect(grip, () => {
                    const all = lanes();
                    const moved = all.find((candidate) => candidate.dataset.col === col);
                    if (!moved || (all[all.indexOf(moved) + 1]?.dataset.col ?? "") !== before)
                        return null;
                    return moved.querySelector(kanbanContract.selectors.laneGrip);
                });
                emitLaneMove(lane, target);
            },
        });
        // Alt and the arrows on a grip stage a lane move, committed when Alt is released.
        const onGripKey = (event, grip) => {
            const lane = grip.closest(kanbanContract.selectors.lane);
            if (!lane || !owns(lane))
                return;
            if (keyboard.matches("cancel", event)) {
                const staged = !!laneStaging.current;
                laneStaging.cancel();
                if (laneMoves.cancel() || staged)
                    event.preventDefault();
                return;
            }
            const { x: direction, y } = keyboard.direction(event, "move");
            const edge = keyboard.matches("moveFirst", event) ? -1 : keyboard.matches("moveLast", event) ? 1 : 0;
            if (!direction && !y && !edge)
                return;
            event.preventDefault();
            const all = lanes();
            const at = laneStaging.current?.lane === lane ? laneStaging.current.target : lane;
            const target = edge < 0 ? all[0] : edge > 0 ? all.at(-1) : direction ? all[all.indexOf(at) + direction] : undefined;
            if (!target)
                return;
            laneStaging.set(grip, { lane, target, grip }, event);
            laneMoves.mark(lane, target);
        };
        const onKeyDown = (event) => {
            const grip = event.target instanceof HTMLElement && !event.defaultPrevented && !event.isComposing
                ? event.target.closest(kanbanContract.selectors.laneGrip)
                : null;
            if (grip && owns(grip)) {
                onGripKey(event, grip);
                return;
            }
            const card = keyboardItem(event, kanbanContract.selectors.card, owns, kanbanContract.selectors.cardMain);
            const itemId = card?.dataset.kanbanCard;
            const laneStop = card ? null : keyboardItem(event, kanbanContract.selectors.lane, owns);
            const current = card && itemId ? card : laneStop === event.target ? laneStop : null;
            if (!current)
                return;
            if (keyboard.matches("cancel", event)) {
                if (staging.current) {
                    event.preventDefault();
                    staging.cancel();
                }
                if (laneMoves.cancel())
                    event.preventDefault();
                clearDragging();
                return;
            }
            const focusDirection = keyboard.direction(event, "focus");
            if (focusDirection.y || keyboard.matches("focusFirst", event) || keyboard.matches("focusLast", event)) {
                const all = [
                    ...host.querySelectorAll(`${kanbanContract.selectors.lane}, ${kanbanContract.selectors.card}`),
                ].filter((stop) => owns(stop) && (stop === current || isCard(stop) || emptyStop(stop)));
                const next = keyboard.matches("focusFirst", event)
                    ? all[0]
                    : keyboard.matches("focusLast", event)
                        ? all.at(-1)
                        : all[all.indexOf(current) + focusDirection.y];
                if (next && next !== current) {
                    event.preventDefault();
                    select(next);
                }
                return;
            }
            const laneDirection = focusDirection.x;
            if (laneDirection) {
                const allLanes = lanes();
                const lane = current.closest(kanbanContract.selectors.lane);
                const row = card && lane ? cardsIn(lane).indexOf(card) : 0;
                const from = lane ? allLanes.indexOf(lane) : -1;
                // The nearest lane that way with a stop: empty lanes that take no focus are passed over.
                const ahead = laneDirection > 0 ? allLanes.slice(from + 1) : allLanes.slice(0, Math.max(0, from)).reverse();
                const next = ahead
                    .map((targetLane) => {
                    const targetCards = cardsIn(targetLane);
                    const nearest = targetCards[Math.min(targetCards.length - 1, Math.max(0, row))];
                    return nearest ?? (emptyStop(targetLane) ? targetLane : undefined);
                })
                    .find((stop) => stop);
                if (next) {
                    event.preventDefault();
                    select(next);
                }
                return;
            }
            const { x: direction, y: rowDirection } = keyboard.direction(event, "move");
            const edge = keyboard.matches("moveFirst", event) ? -1 : keyboard.matches("moveLast", event) ? 1 : 0;
            if (!direction && !rowDirection && !edge)
                return;
            event.preventDefault();
            if (!card || !itemId)
                return;
            const lane = card.closest(kanbanContract.selectors.lane);
            const allLanes = lanes();
            const sourceIndex = lane ? allLanes.indexOf(lane) : -1;
            const stagedTarget = staging.current?.itemId === itemId ? staging.current.target : null;
            const currentLane = stagedTarget?.lane ?? lane;
            const currentIndex = currentLane ? allLanes.indexOf(currentLane) : sourceIndex;
            const targetLane = allLanes[currentIndex + direction];
            if (!targetLane)
                return;
            const others = (inLane) => cardsIn(inLane).filter((candidate) => candidate !== card);
            const position = stagedTarget
                ? stagedTarget.before
                    ? others(stagedTarget.lane).findIndex((candidate) => candidate.dataset.kanbanCard === stagedTarget.before)
                    : others(stagedTarget.lane).length
                : lane
                    ? cardsIn(lane).indexOf(card)
                    : 0;
            // Across lanes the card keeps its row, clamped to the target lane's length.
            const candidates = others(targetLane);
            const nextPosition = edge
                ? edge < 0
                    ? 0
                    : candidates.length
                : Math.max(0, Math.min(candidates.length, position + rowDirection));
            if (!direction && nextPosition === position)
                return;
            const before = candidates[nextPosition]?.dataset.kanbanCard ?? "";
            const target = { col: Number(targetLane.dataset.col ?? 0), before, lane: targetLane };
            staging.set(card, { itemId, target }, event);
            markTarget(target);
        };
        host.addEventListener("keydown", onKeyDown);
        host.addEventListener("pointerdown", onPointerDown);
        cleanup(() => {
            staging.dispose();
            focus.dispose();
            pointerDispose();
            flip.dispose();
            laneStaging.dispose();
            laneMoves.dispose();
            laneFlip.dispose();
            host.removeEventListener("keydown", onKeyDown);
            host.removeEventListener("pointerdown", onPointerDown);
            clearMarks();
            clearDragging();
        });
    },
});
