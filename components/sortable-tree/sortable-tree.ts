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
import { sortableTreeContract, type TreeMoveDetail } from "./contracts/sortable-tree.ts";
import { installFlip } from "./core/flip.ts";
import { installFocusRecovery } from "./core/focus-recovery.ts";
import { cancelKeys, keyboardBindings, keyboardItem, moveKeys, platformFocusKeys } from "./core/keyboard.ts";
import { installKeyboardStaging } from "./core/keyboard-staging.ts";
import { markRocketHost, ownsRocketElement } from "./core/ownership.ts";
import { installPointerDrag } from "./core/pointer-drag.ts";
import { installTargetIndicator } from "./core/visual-outlets.ts";

type Target = { list: HTMLElement; parentId: string; before: string; into: boolean };

rocket("sb-sortable-tree", {
  mode: "light",
  manifest: {
    events: [
      {
        name: sortableTreeContract.events.move,
        kind: "custom-event",
        bubbles: true,
        composed: true,
        description:
          'A row was dropped, or a keyboard move committed. detail: { itemId, fromParent, toParent, before }: the parents are folder ids, "" for the top level; before is the id of the node it now precedes, "" for the end of the folder.',
      },
    ],
  },
  setup({ host, cleanup }: { host: HTMLElement; cleanup: (fn: () => void) => void }) {
    cleanup(markRocketHost(host));
    const { node: nodeSelector, row: rowSelector, children: childrenSelector } = sortableTreeContract.selectors;
    const owns = (element: HTMLElement) => ownsRocketElement(host, element);
    const keyboard = keyboardBindings(host, { ...platformFocusKeys(navigator.platform), ...moveKeys, ...cancelKeys });
    const focus = installFocusRecovery(host);
    const nodeFor = (row: HTMLElement) => row.closest<HTMLElement>(nodeSelector);
    const rowId = (row: HTMLElement) => (owns(row) ? (nodeFor(row)?.dataset.treeNode ?? null) : null);
    const childrenOf = (list: HTMLElement) =>
      [...list.children].filter(
        (child): child is HTMLElement => child instanceof HTMLElement && child.matches(nodeSelector),
      );
    const nodeById = (id: string) =>
      [...host.querySelectorAll<HTMLElement>(nodeSelector)].find((node) => owns(node) && node.dataset.treeNode === id);
    const rowOf = (node: HTMLElement) => node.querySelector<HTMLElement>(`:scope > ${rowSelector}`);
    const listFor = (node: HTMLElement) => node.parentElement?.closest<HTMLElement>(childrenSelector);
    const collapsed = new Set<string>();
    let pendingExpansion: { id: string; parentId: string } | null = null;
    const syncExpanded = () => {
      const moved = pendingExpansion && nodeById(pendingExpansion.id);
      if (moved && pendingExpansion && listFor(moved)?.dataset.treeParent === pendingExpansion.parentId) {
        collapsed.delete(pendingExpansion.parentId);
        pendingExpansion = null;
      }
      host.querySelectorAll<HTMLElement>(nodeSelector).forEach((node) => {
        if (!owns(node) || node.dataset.treeKind !== "folder") return;
        const list = node.querySelector<HTMLElement>(`:scope > ${childrenSelector}`);
        const row = rowOf(node);
        if (!list || !row) return;
        const closed = collapsed.has(node.dataset.treeNode ?? "");
        list.hidden = closed;
        row.setAttribute("aria-expanded", String(!closed));
      });
    };
    const observer = new MutationObserver(syncExpanded);
    observer.observe(host, { childList: true, subtree: true });
    const flip = installFlip({ host, itemSelector: rowSelector, itemId: rowId });
    const indicator = installTargetIndicator(host);

    const valid = (sourceId: string, target: Target): boolean => {
      const source = nodeById(sourceId);
      if (!source || source.contains(target.list)) return false;
      const parent = target.parentId && nodeById(target.parentId);
      if (parent && parent.dataset.treeKind !== "folder") return false;
      return target.before !== sourceId;
    };
    const targetAt = (x: number, y: number, id: string): Target | null => {
      const hit = document.elementFromPoint(x, y);
      const row = hit?.closest<HTMLElement>(rowSelector);
      const node = row && nodeFor(row);
      if (row && node && owns(row)) {
        const list = listFor(node);
        if (!list) return null;
        const ratio = (y - row.getBoundingClientRect().top) / row.getBoundingClientRect().height;
        if (node.dataset.treeKind === "folder" && ratio >= 0.3 && ratio <= 0.7) {
          const nested = node.querySelector<HTMLElement>(`:scope > ${childrenSelector}`);
          const target = nested && { list: nested, parentId: node.dataset.treeNode ?? "", before: "", into: true };
          return target && valid(id, target) ? target : null;
        }
        const before =
          ratio < 0.5 ? (node.dataset.treeNode ?? "") : (node.nextElementSibling?.getAttribute("data-tree-node") ?? "");
        const target = { list, parentId: list.dataset.treeParent ?? "", before, into: false };
        return valid(id, target) ? target : null;
      }
      const list = hit?.closest<HTMLElement>(childrenSelector);
      if (!list || !owns(list)) return null;
      const target = { list, parentId: list.dataset.treeParent ?? "", before: "", into: !!list.dataset.treeParent };
      return valid(id, target) ? target : null;
    };
    let marked: { element: HTMLElement; attribute: string } | null = null;
    const mark = (target: Target | null) => {
      if (marked) marked.element.removeAttribute(marked.attribute);
      marked = null;
      if (!target) {
        indicator.clear();
        return;
      }
      if (target.into) {
        const parent = nodeById(target.parentId);
        const row = parent && rowOf(parent);
        if (row) marked = { element: row, attribute: "data-tree-into" };
      } else if (target.before) {
        const node = childrenOf(target.list).find((candidate) => candidate.dataset.treeNode === target.before);
        const row = node && rowOf(node);
        if (row) marked = { element: row, attribute: "data-tree-before" };
      } else marked = { element: target.list, attribute: "data-tree-end" };
      if (marked) {
        marked.element.setAttribute(marked.attribute, "");
        indicator.show(marked.element, target.into ? "into" : target.before ? "before" : "end");
      }
    };
    const emitMove = (id: string, target: Target) => {
      const node = nodeById(id);
      const source = node && listFor(node);
      if (!node || !source || !valid(id, target)) return;
      const fromParent = source.dataset.treeParent ?? "";
      if (source === target.list && (node.nextElementSibling?.getAttribute("data-tree-node") ?? "") === target.before)
        return;
      if (collapsed.has(target.parentId)) pendingExpansion = { id, parentId: target.parentId };
      host.dispatchEvent(
        new CustomEvent<TreeMoveDetail>(sortableTreeContract.events.move, {
          bubbles: true,
          composed: true,
          detail: { itemId: id, fromParent, toParent: target.parentId, before: target.before },
        }),
      );
    };

    const staging = installKeyboardStaging<{ id: string; target: Target }>({
      host,
      owns,
      onCancel: () => mark(null),
      onCommit: ({ id, target }) => {
        mark(null);
        const source = nodeById(id);
        const sourceRow = source && rowOf(source);
        if (sourceRow)
          focus.expect(sourceRow, () => {
            const node = nodeById(id);
            if (!node || listFor(node)?.dataset.treeParent !== target.parentId) return null;
            if ((node.nextElementSibling?.getAttribute("data-tree-node") ?? "") !== target.before) return null;
            const destination = listFor(node);
            if (destination?.closest(`${childrenSelector}[hidden]`)) {
              // A confirmed move into a closed folder should remain keyboard-reachable.
              const parent = nodeById(target.parentId);
              if (parent) {
                collapsed.delete(target.parentId);
                syncExpanded();
              }
            }
            return rowOf(node);
          });
        flip.prepare();
        emitMove(id, target);
      },
    });
    const onKeyDown = (event: KeyboardEvent) => {
      const row = keyboardItem(event, rowSelector, owns);
      const id = row && rowId(row);
      if (!id) return;
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
        const rows = [...host.querySelectorAll<HTMLElement>(rowSelector)].filter(
          (candidate) => owns(candidate) && !candidate.closest(`${childrenSelector}[hidden]`),
        );
        const index = rows.indexOf(row!);
        const node = nodeById(id);
        const childList = node?.querySelector<HTMLElement>(`:scope > ${childrenSelector}`);
        const firstChild = childList && childrenOf(childList)[0];
        const parent = node?.parentElement?.closest<HTMLElement>(nodeSelector);
        if (childList && focusDirection.x < 0 && !childList.hidden) {
          event.preventDefault();
          collapsed.add(id);
          syncExpanded();
          return;
        }
        if (childList?.hidden && focusDirection.x > 0) {
          event.preventDefault();
          collapsed.delete(id);
          syncExpanded();
          return;
        }
        const next =
          focusDirection.y < 0
            ? rows[index - 1]
            : focusDirection.y > 0
              ? rows[index + 1]
              : keyboard.matches("focusFirst", event)
                ? rows[0]
                : keyboard.matches("focusLast", event)
                  ? rows.at(-1)
                  : focusDirection.x > 0 && firstChild
                    ? rowOf(firstChild)
                    : focusDirection.x < 0 && parent
                      ? rowOf(parent)
                      : null;
        if (next) {
          event.preventDefault();
          next.focus();
        }
        return;
      }
      const { x: dx, y: dy } = keyboard.direction(event, "move");
      if (!dx && !dy) return;
      const node = nodeById(id);
      const list = staging.current?.id === id ? staging.current.target.list : node && listFor(node);
      if (!node || !list) return;
      const siblings = childrenOf(list).filter((candidate) => candidate !== node);
      const currentBefore =
        staging.current?.id === id
          ? staging.current.target.before
          : (node.nextElementSibling?.getAttribute("data-tree-node") ?? "");
      const position = currentBefore
        ? siblings.findIndex((candidate) => candidate.dataset.treeNode === currentBefore)
        : siblings.length;
      let target: Target | null = null;
      if (dy) {
        const next = Math.max(0, Math.min(siblings.length, position + dy));
        if (next !== position) {
          target = {
            list,
            parentId: list.dataset.treeParent ?? "",
            before: siblings[next]?.dataset.treeNode ?? "",
            into: false,
          };
        } else {
          const parent = list.closest<HTMLElement>(nodeSelector);
          const outer = parent && listFor(parent);
          if (parent && outer)
            target = {
              list: outer,
              parentId: outer.dataset.treeParent ?? "",
              before:
                dy < 0
                  ? (parent.dataset.treeNode ?? "")
                  : (parent.nextElementSibling?.getAttribute("data-tree-node") ?? ""),
              into: false,
            };
        }
      } else if (dx > 0) {
        const previous = siblings
          .slice(0, position)
          .reverse()
          .find((candidate) => candidate.dataset.treeKind === "folder");
        const nested = previous?.querySelector<HTMLElement>(`:scope > ${childrenSelector}`);
        if (nested) target = { list: nested, parentId: previous?.dataset.treeNode ?? "", before: "", into: true };
      } else {
        const parent = list.closest<HTMLElement>(nodeSelector);
        const outer = parent && listFor(parent);
        if (parent && outer)
          target = {
            list: outer,
            parentId: outer.dataset.treeParent ?? "",
            before: parent.nextElementSibling?.getAttribute("data-tree-node") ?? "",
            into: false,
          };
      }
      if (!target || !valid(id, target)) return;
      event.preventDefault();
      staging.set(row, { id, target }, event);
      mark(target);
    };
    const onClick = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      const row = target.closest<HTMLElement>(rowSelector);
      const id = row && owns(target) ? rowId(row) : null;
      const node = row && id ? nodeFor(row) : null;
      const list =
        node?.dataset.treeKind === "folder" ? node.querySelector<HTMLElement>(`:scope > ${childrenSelector}`) : null;
      const control = target.closest(
        "button, a, input, select, textarea, [contenteditable]:not([contenteditable='false'])",
      );
      if (!id || !list || control || event.defaultPrevented) return;
      if (list.hidden) collapsed.delete(id);
      else collapsed.add(id);
      syncExpanded();
    };
    const dispose = installPointerDrag({
      host,
      itemSelector: rowSelector,
      itemId: rowId,
      targetAt,
      sameTarget: (a, b) => a?.list === b?.list && a?.before === b?.before && a?.into === b?.into,
      mark,
      beforeCommit: (id, rect) => flip.prepare({ itemId: id, rect }),
      commit: emitMove,
    });
    host.addEventListener("keydown", onKeyDown);
    host.addEventListener("click", onClick);
    cleanup(() => {
      observer.disconnect();
      pendingExpansion = null;
      staging.dispose();
      focus.dispose();
      host.removeEventListener("keydown", onKeyDown);
      host.removeEventListener("click", onClick);
      dispose();
      flip.dispose();
      indicator.clear();
    });
  },
});
