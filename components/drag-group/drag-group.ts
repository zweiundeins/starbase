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
import { dragGroupContract, type DragGroupMoveDetail } from "./contracts/drag-group.ts";
import { installFlip } from "./core/flip.ts";
import { installFocusRecovery } from "./core/focus-recovery.ts";
import { insertionBefore } from "./core/insertion-target.ts";
import { cancelKeys, keyboardBindings, keyboardItem, moveKeys, platformFocusKeys } from "./core/keyboard.ts";
import { installKeyboardStaging } from "./core/keyboard-staging.ts";
import { markRocketHost, ownsRocketElement } from "./core/ownership.ts";
import { installPointerDrag } from "./core/pointer-drag.ts";
import { installTargetIndicator } from "./core/visual-outlets.ts";

type Target = { list: HTMLElement; toList: string; before: string };

rocket("sb-drag-group", {
  mode: "light",
  setup({ host, cleanup }: { host: HTMLElement; cleanup: (fn: () => void) => void }) {
    cleanup(markRocketHost(host));
    const { list: listSelector, item: itemSelector } = dragGroupContract.selectors;
    const owns = (element: HTMLElement): boolean => ownsRocketElement(host, element);
    const keyboard = keyboardBindings(host, { ...platformFocusKeys(navigator.platform), ...moveKeys, ...cancelKeys });
    const focus = installFocusRecovery(host);
    const itemId = (item: HTMLElement): string | null =>
      owns(item) && item.closest(listSelector) ? (item.dataset.dragItem ?? null) : null;
    const lists = () => [...host.querySelectorAll<HTMLElement>(listSelector)].filter(owns);
    const itemsIn = (list: HTMLElement) =>
      [...list.querySelectorAll<HTMLElement>(itemSelector)].filter(
        (item) => owns(item) && item.closest(listSelector) === list,
      );
    const flip = installFlip({ host, itemSelector, itemId });
    const indicator = installTargetIndicator(host);

    const targetAt = (x: number, y: number, sourceId: string): Target | null => {
      const hit = document.elementFromPoint(x, y);
      const list = hit?.closest<HTMLElement>(listSelector);
      if (!list || !owns(list) || list.dataset.dropList === undefined) return null;
      const candidates = itemsIn(list)
        .filter((item) => item.dataset.dragItem !== sourceId)
        .map((item) => {
          const rect = item.getBoundingClientRect();
          return { id: item.dataset.dragItem ?? "", top: rect.top, bottom: rect.bottom };
        });
      return { list, toList: list.dataset.dropList, before: insertionBefore(candidates, y) };
    };
    const mark = (target: Target | null): void => {
      indicator.clear();
      lists().forEach((list) => {
        itemsIn(list).forEach((item) => item.removeAttribute("data-drop-before"));
        list.removeAttribute("data-drop-active");
        list.removeAttribute("data-drop-end");
      });
      if (!target) return;
      target.list.setAttribute("data-drop-active", "");
      if (target.before) {
        const beforeItem = itemsIn(target.list).find((item) => item.dataset.dragItem === target.before);
        beforeItem?.setAttribute("data-drop-before", "");
        indicator.show(beforeItem ?? null, "before");
      } else {
        target.list.setAttribute("data-drop-end", "");
        indicator.show(target.list, "end");
      }
    };
    const emitMove = (id: string, target: Target) => {
      const source = [...host.querySelectorAll<HTMLElement>(itemSelector)].find((item) => itemId(item) === id);
      const fromList = source?.closest<HTMLElement>(listSelector)?.dataset.dropList;
      if (fromList === undefined) return;
      host.dispatchEvent(
        new CustomEvent<DragGroupMoveDetail>(dragGroupContract.events.move, {
          bubbles: true,
          composed: true,
          detail: { itemId: id, fromList, toList: target.toList, before: target.before },
        }),
      );
    };
    const staging = installKeyboardStaging<{ itemId: string; target: Target }>({
      host,
      owns,
      onCancel: () => mark(null),
      onCommit: ({ itemId: id, target }) => {
        mark(null);
        const source = [...host.querySelectorAll<HTMLElement>(itemSelector)].find((item) => itemId(item) === id);
        if (source)
          focus.expect(source, () => {
            const destination = lists().find((list) => list.dataset.dropList === target.toList);
            const items = destination ? itemsIn(destination) : [];
            const index = items.findIndex((item) => itemId(item) === id);
            return index >= 0 && (items[index + 1]?.dataset.dragItem ?? "") === target.before ? items[index]! : null;
          });
        flip.prepare();
        emitMove(id, target);
      },
    });
    const onKeyDown = (event: KeyboardEvent) => {
      const item = keyboardItem(event, itemSelector, owns);
      const id = item && itemId(item);
      if (!item || !id) return;
      if (keyboard.matches("cancel", event)) {
        if (staging.current) {
          event.preventDefault();
          staging.cancel();
        }
        return;
      }
      const focusDirection = keyboard.direction(event, "focus");
      if (
        focusDirection.x ||
        focusDirection.y ||
        keyboard.matches("focusFirst", event) ||
        keyboard.matches("focusLast", event)
      ) {
        const source = item.closest<HTMLElement>(listSelector);
        const groupLists = lists();
        const siblings = source ? itemsIn(source) : [];
        const index = siblings.indexOf(item);
        const listIndex = source ? groupLists.indexOf(source) : -1;
        const nextList = focusDirection.x ? groupLists[listIndex + focusDirection.x] : null;
        const neighbors = nextList ? itemsIn(nextList) : [];
        const next =
          focusDirection.y < 0
            ? siblings[index - 1]
            : focusDirection.y > 0
              ? siblings[index + 1]
              : keyboard.matches("focusFirst", event)
                ? siblings[0]
                : keyboard.matches("focusLast", event)
                  ? siblings.at(-1)
                  : neighbors[Math.min(index, neighbors.length - 1)];
        if (next) {
          event.preventDefault();
          next.focus();
        }
        return;
      }
      const { x: horizontal, y: vertical } = keyboard.direction(event, "move");
      if (!horizontal && !vertical) return;
      const sourceList = item.closest<HTMLElement>(listSelector);
      const stagedTarget = staging.current?.itemId === id ? staging.current.target : null;
      const currentList = stagedTarget?.list ?? sourceList;
      const allLists = lists();
      const list = currentList && allLists[allLists.indexOf(currentList) + horizontal];
      if (!list) return;
      let before = "";
      if (vertical) {
        const candidates = itemsIn(list).filter((candidate) => candidate !== item);
        const position = stagedTarget
          ? stagedTarget.before
            ? candidates.findIndex((candidate) => candidate.dataset.dragItem === stagedTarget.before)
            : candidates.length
          : itemsIn(list).indexOf(item);
        const nextPosition = Math.max(0, Math.min(candidates.length, position + vertical));
        if (nextPosition === position) return;
        before = candidates[nextPosition]?.dataset.dragItem ?? "";
      }
      const target = { list, toList: list.dataset.dropList ?? "", before };
      event.preventDefault();
      staging.set(item, { itemId: id, target }, event);
      mark(target);
    };
    const dispose = installPointerDrag({
      host,
      itemSelector,
      itemId,
      targetAt,
      sameTarget: (a, b) => a?.list === b?.list && a?.before === b?.before,
      mark,
      beforeCommit: (id, rect) => flip.prepare({ itemId: id, rect }),
      commit: emitMove,
    });
    host.addEventListener("keydown", onKeyDown);
    cleanup(() => {
      staging.dispose();
      focus.dispose();
      host.removeEventListener("keydown", onKeyDown);
      dispose();
      flip.dispose();
      indicator.clear();
    });
  },
});
