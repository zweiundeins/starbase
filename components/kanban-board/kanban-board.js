// Generated from kanban-board.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import { rocket } from "datastar";
import { installFlip } from "./core/flip.js";
import { installFocusRecovery } from "./core/focus-recovery.js";
import { insertionBefore } from "./core/insertion-target.js";
import { keyboardBindings, keyboardItem, platformFocusKeys } from "./core/keyboard.js";
import { installKeyboardStaging } from "./core/keyboard-staging.js";
import { markRocketHost, ownsRocketElement } from "./core/ownership.js";
import { installPointerDrag } from "./core/pointer-drag.js";
import { installTargetIndicator } from "./core/visual-outlets.js";
import { kanbanContract, defaultKanbanKeyboard } from "./contracts/kanban.js";
rocket("sb-kanban-board", {
    mode: "light",
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
        const select = (card) => {
            if (!card)
                return;
            card.focus();
            host.dispatchEvent(new CustomEvent(kanbanContract.events.select, {
                bubbles: true,
                composed: true,
                detail: { cardId: card.dataset.kanbanCard ?? "" },
            }));
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
        const onKeyDown = (event) => {
            const card = keyboardItem(event, kanbanContract.selectors.card, owns, kanbanContract.selectors.cardMain);
            const itemId = card?.dataset.kanbanCard;
            if (!card || !itemId)
                return;
            if (keyboard.matches("cancel", event)) {
                if (staging.current) {
                    event.preventDefault();
                    staging.cancel();
                }
                clearDragging();
                return;
            }
            const focusDirection = keyboard.direction(event, "focus");
            if (focusDirection.y || keyboard.matches("focusFirst", event) || keyboard.matches("focusLast", event)) {
                const all = cards();
                const next = keyboard.matches("focusFirst", event)
                    ? all[0]
                    : keyboard.matches("focusLast", event)
                        ? all.at(-1)
                        : all[all.indexOf(card) + focusDirection.y];
                if (next && next !== card) {
                    event.preventDefault();
                    select(next);
                }
                return;
            }
            const laneDirection = focusDirection.x;
            if (laneDirection) {
                const allLanes = lanes();
                const lane = card.closest(kanbanContract.selectors.lane);
                const laneIndex = lane ? allLanes.indexOf(lane) : -1;
                const targetLane = allLanes[laneIndex + laneDirection];
                const targetCards = targetLane ? cardsIn(targetLane) : [];
                const row = lane ? cardsIn(lane).indexOf(card) : 0;
                const next = targetCards[Math.min(targetCards.length - 1, Math.max(0, row))];
                if (next) {
                    event.preventDefault();
                    select(next);
                }
                return;
            }
            const { x: direction, y: rowDirection } = keyboard.direction(event, "move");
            if (!direction && !rowDirection)
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
            let before = "";
            if (rowDirection) {
                const candidates = cardsIn(targetLane).filter((candidate) => candidate !== card);
                const position = stagedTarget
                    ? stagedTarget.before
                        ? candidates.findIndex((candidate) => candidate.dataset.kanbanCard === stagedTarget.before)
                        : candidates.length
                    : cardsIn(targetLane).indexOf(card);
                const nextPosition = Math.max(0, Math.min(candidates.length, position + rowDirection));
                if (nextPosition === position)
                    return;
                before = candidates[nextPosition]?.dataset.kanbanCard ?? "";
            }
            const target = { col: Number(targetLane.dataset.col ?? 0), before, lane: targetLane };
            event.preventDefault();
            staging.set(card, { itemId, target }, event);
            markTarget(target);
        };
        host.addEventListener("keydown", onKeyDown);
        cleanup(() => {
            staging.dispose();
            focus.dispose();
            pointerDispose();
            flip.dispose();
            host.removeEventListener("keydown", onKeyDown);
            clearMarks();
            clearDragging();
        });
    },
});
