// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import { rocket } from "datastar";
import { installFlip } from "./core/flip.ts";
import { installFocusRecovery } from "./core/focus-recovery.ts";
import { insertionBefore } from "./core/insertion-target.ts";
import { cancelKeys, keyboardBindings, keyboardItem, moveKeys, platformFocusKeys } from "./core/keyboard.ts";
import { installKeyboardStaging } from "./core/keyboard-staging.ts";
import { markRocketHost, ownsRocketElement } from "./core/ownership.ts";
import { installPointerDrag } from "./core/pointer-drag.ts";
import { installTargetIndicator } from "./core/visual-outlets.ts";
import { sortableListContract, type SortableMoveDetail } from "./contracts/sortable-list.ts";

rocket("sb-sortable-list", {
  mode: "light",
  setup({ host, cleanup }: { host: HTMLElement; cleanup: (fn: () => void) => void }) {
    cleanup(markRocketHost(host));
    const owns = (item: HTMLElement) => ownsRocketElement(host, item);
    const { focusNext, focusPrevious, focusFirst, focusLast } = platformFocusKeys(navigator.platform);
    const keyboard = keyboardBindings(host, {
      focusNext,
      focusPrevious,
      focusFirst,
      focusLast,
      moveUp: moveKeys.moveUp,
      moveDown: moveKeys.moveDown,
      cancel: cancelKeys.cancel,
    });
    const itemId = (item: HTMLElement): string | null => (owns(item) ? (item.dataset.sortableItem ?? null) : null);
    const focus = installFocusRecovery(host);
    const items = () => [...host.querySelectorAll<HTMLElement>(sortableListContract.selectors.item)].filter(owns);
    const flip = installFlip({ host, itemSelector: sortableListContract.selectors.item, itemId });
    const indicator = installTargetIndicator(host);
    const targetAt = (x: number, y: number, sourceId: string) => {
      const hit = document.elementFromPoint(x, y);
      if (!hit || !owns(hit as HTMLElement)) return null;
      const candidates = items()
        .filter((item) => item.dataset.sortableItem !== sourceId)
        .map((item) => {
          const rect = item.getBoundingClientRect();
          return { id: item.dataset.sortableItem ?? "", top: rect.top, bottom: rect.bottom };
        });
      return { before: insertionBefore(candidates, y) };
    };
    const mark = (target: { before: string } | null) => {
      let beforeItem: HTMLElement | null = null;
      items().forEach((item) => {
        item.toggleAttribute("data-drop-before", target?.before !== "" && item.dataset.sortableItem === target?.before);
        if (target?.before === item.dataset.sortableItem) beforeItem = item;
      });
      host.toggleAttribute("data-drop-end", target !== null && target.before === "");
      indicator.show(target ? (target.before ? beforeItem : host) : null, target?.before ? "before" : "end");
    };
    const emitMove = (id: string, target: { before: string }) => {
      host.dispatchEvent(
        new CustomEvent<SortableMoveDetail>(sortableListContract.events.move, {
          bubbles: true,
          composed: true,
          detail: { itemId: id, before: target.before },
        }),
      );
    };
    const staging = installKeyboardStaging<{ id: string; before: string }>({
      host,
      owns,
      onCancel: () => mark(null),
      onCommit: ({ id, before }) => {
        mark(null);
        const source = items().find((item) => itemId(item) === id);
        if (source)
          focus.expect(source, () => {
            const all = items();
            const index = all.findIndex((item) => itemId(item) === id);
            return index >= 0 && (all[index + 1]?.dataset.sortableItem ?? "") === before ? all[index]! : null;
          });
        flip.prepare();
        emitMove(id, { before });
      },
    });
    const dispose = installPointerDrag({
      host,
      itemSelector: sortableListContract.selectors.item,
      itemId,
      targetAt,
      sameTarget: (a, b) => a?.before === b?.before,
      mark,
      beforeCommit: (id, rect) => flip.prepare({ itemId: id, rect }),
      commit: emitMove,
    });
    const onKeyDown = (event: KeyboardEvent) => {
      const item = keyboardItem(event, sortableListContract.selectors.item, owns);
      if (!item) return;
      if (keyboard.matches("cancel", event)) {
        if (staging.current) {
          event.preventDefault();
          staging.cancel();
        }
        return;
      }
      const all = items();
      const index = all.indexOf(item);
      const focusDirection = keyboard.direction(event, "focus").y;
      if (focusDirection || keyboard.matches("focusFirst", event) || keyboard.matches("focusLast", event)) {
        const next =
          focusDirection > 0
            ? all[index + 1]
            : focusDirection < 0
              ? all[index - 1]
              : keyboard.matches("focusFirst", event)
                ? all[0]
                : keyboard.matches("focusLast", event)
                  ? all.at(-1)
                  : null;
        if (next) {
          event.preventDefault();
          next.focus();
        }
        return;
      }
      const direction = keyboard.direction(event, "move").y;
      if (!direction) return;
      const id = itemId(item);
      if (!id) return;
      const candidates = all.filter((candidate) => candidate !== item);
      const position =
        staging.current?.id === id
          ? staging.current.before
            ? candidates.findIndex((candidate) => itemId(candidate) === staging.current?.before)
            : candidates.length
          : index;
      const nextPosition = Math.max(0, Math.min(candidates.length, position + direction));
      if (nextPosition === position) return;
      const target = { id, before: candidates[nextPosition]?.dataset.sortableItem ?? "" };
      event.preventDefault();
      staging.set(item, target, event);
      mark(target);
    };
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
