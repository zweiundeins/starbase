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
import { installFlip } from "./core/flip.ts";
import { installFocusRecovery } from "./core/focus-recovery.ts";
import { insertionBefore } from "./core/insertion-target.ts";
import { keyboardBindings, keyboardItem, platformFocusKeys } from "./core/keyboard.ts";
import { installKeyboardStaging } from "./core/keyboard-staging.ts";
import { markRocketHost, ownsRocketElement } from "./core/ownership.ts";
import { installPointerDrag } from "./core/pointer-drag.ts";
import { installTargetIndicator } from "./core/visual-outlets.ts";
import { kanbanContract, defaultKanbanKeyboard, type KanbanMoveDetail } from "./contracts/kanban.ts";

type Target = { col: number; before: string; lane: HTMLElement };

rocket("sb-kanban-board", {
  mode: "light",
  manifest: {
    events: [
      {
        name: kanbanContract.events.move,
        kind: "custom-event",
        bubbles: true,
        composed: true,
        description:
          "A card was dropped, or a keyboard move committed. detail: { cardId, col, before }: col is the lane's " +
          'data-col as a number, before the id of the card it now precedes in that lane, or "" for the end.',
      },
      {
        name: kanbanContract.events.select,
        kind: "custom-event",
        bubbles: true,
        composed: true,
        description: "A card was selected: arrow focus moved to it, or the pointer pressed it. detail: { cardId }.",
      },
    ],
  },
  setup({ host, cleanup }: { host: HTMLElement; cleanup: (fn: () => void) => void }) {
    cleanup(markRocketHost(host));
    const owns = (element: HTMLElement) => ownsRocketElement(host, element);
    const { focusNext, focusPrevious, focusFirst, focusLast } = platformFocusKeys(navigator.platform);
    const keyboard = keyboardBindings(
      host,
      {
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
      },
      {
        focusNext: "selectNext",
        focusPrevious: "selectPrevious",
        focusLeft: "selectLeft",
        focusRight: "selectRight",
      },
    );
    const cards = () => [...host.querySelectorAll<HTMLElement>(kanbanContract.selectors.card)].filter(owns);
    const lanes = () => [...host.querySelectorAll<HTMLElement>(kanbanContract.selectors.lane)].filter(owns);
    const cardsIn = (lane: HTMLElement) =>
      [...lane.querySelectorAll<HTMLElement>(kanbanContract.selectors.card)].filter(owns);
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
    const markTarget = (target: Target | null): void => {
      clearMarks();
      if (!target) return;
      target.lane.setAttribute("data-drop-active", "true");
      if (target.before) {
        const beforeCard = cardsIn(target.lane).find((card) => card.dataset.kanbanCard === target.before);
        beforeCard?.setAttribute("data-drop-before", "");
        indicator.show(beforeCard ?? null, "before");
      } else {
        const end = target.lane.querySelector<HTMLElement>("[data-kanban-lane-cards]");
        end?.setAttribute("data-drop-end", "");
        indicator.show(end ?? null, "end");
      }
    };
    const isCard = (element: HTMLElement) => element.matches(kanbanContract.selectors.card);
    // A lane the page made focusable (tabindex) is a focus stop while it has no cards.
    const emptyStop = (lane: HTMLElement) => lane.hasAttribute("tabindex") && cardsIn(lane).length === 0;
    const emitSelect = (card: HTMLElement): void => {
      host.dispatchEvent(
        new CustomEvent(kanbanContract.events.select, {
          bubbles: true,
          composed: true,
          detail: { cardId: card.dataset.kanbanCard ?? "" },
        }),
      );
    };
    const select = (stop: HTMLElement | undefined): void => {
      if (!stop) return;
      stop.focus();
      if (isCard(stop)) emitSelect(stop);
    };
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target;
      if (event.button !== 0 || !(target instanceof HTMLElement) || !owns(target)) return;
      if (
        target.closest("button, a, input, select, textarea, [contenteditable]:not([contenteditable='false'])") &&
        !target.closest(kanbanContract.selectors.cardMain)
      )
        return;
      const card = target.closest<HTMLElement>(kanbanContract.selectors.card);
      if (card && owns(card)) emitSelect(card);
    };
    const targetAt = (x: number, y: number, itemId: string): Target | null => {
      const lane = document.elementFromPoint(x, y)?.closest<HTMLElement>(kanbanContract.selectors.lane);
      if (!lane || !owns(lane)) return null;
      const candidates = cardsIn(lane)
        .filter((card) => card.dataset.kanbanCard !== itemId)
        .map((card) => {
          const rect = card.getBoundingClientRect();
          return { id: card.dataset.kanbanCard ?? "", top: rect.top, bottom: rect.bottom };
        });
      const before = insertionBefore(candidates, y);
      return { col: Number(lane.dataset.col ?? 0), before, lane };
    };
    const emitMove = (itemId: string, target: Target) => {
      host.dispatchEvent(
        new CustomEvent<KanbanMoveDetail>(kanbanContract.events.move, {
          bubbles: true,
          composed: true,
          detail: { cardId: itemId, col: target.col, before: target.before },
        }),
      );
    };
    const staging = installKeyboardStaging<{ itemId: string; target: Target }>({
      host,
      owns,
      onCancel: clearMarks,
      onCommit: ({ itemId, target }) => {
        clearMarks();
        const source = cards().find((card) => card.dataset.kanbanCard === itemId);
        if (source)
          focus.expect(source, () => {
            const card = cards().find((candidate) => candidate.dataset.kanbanCard === itemId);
            const lane = card?.closest<HTMLElement>(kanbanContract.selectors.lane);
            if (!card || !lane || Number(lane.dataset.col) !== target.col) return null;
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
    const onKeyDown = (event: KeyboardEvent) => {
      const card = keyboardItem(event, kanbanContract.selectors.card, owns, kanbanContract.selectors.cardMain);
      const itemId = card?.dataset.kanbanCard;
      const laneStop = card ? null : keyboardItem(event, kanbanContract.selectors.lane, owns);
      const current = card && itemId ? card : laneStop === event.target ? laneStop : null;
      if (!current) return;
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
        const all = [
          ...host.querySelectorAll<HTMLElement>(`${kanbanContract.selectors.lane}, ${kanbanContract.selectors.card}`),
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
        const lane = current.closest<HTMLElement>(kanbanContract.selectors.lane);
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
      if (!direction && !rowDirection && !edge) return;
      event.preventDefault();
      if (!card || !itemId) return;
      const lane = card.closest<HTMLElement>(kanbanContract.selectors.lane);
      const allLanes = lanes();
      const sourceIndex = lane ? allLanes.indexOf(lane) : -1;
      const stagedTarget = staging.current?.itemId === itemId ? staging.current.target : null;
      const currentLane = stagedTarget?.lane ?? lane;
      const currentIndex = currentLane ? allLanes.indexOf(currentLane) : sourceIndex;
      const targetLane = allLanes[currentIndex + direction];
      if (!targetLane) return;
      const others = (inLane: HTMLElement) => cardsIn(inLane).filter((candidate) => candidate !== card);
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
      if (!direction && nextPosition === position) return;
      const before = candidates[nextPosition]?.dataset.kanbanCard ?? "";
      const target: Target = { col: Number(targetLane.dataset.col ?? 0), before, lane: targetLane };
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
      host.removeEventListener("keydown", onKeyDown);
      host.removeEventListener("pointerdown", onPointerDown);
      clearMarks();
      clearDragging();
    });
  },
});
