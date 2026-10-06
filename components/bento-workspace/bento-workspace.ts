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
import { bentoContract, type BentoMoveDetail, type BentoPosition, type BentoResizeDetail } from "./contracts/bento.ts";
import { installFlip } from "./core/flip.ts";
import { installFocusRecovery } from "./core/focus-recovery.ts";
import {
  cancelKeys,
  gridKeys,
  keyboardBindings,
  keyboardItem,
  moveKeys,
  platformFocusKeys,
  resizeKeys,
} from "./core/keyboard.ts";
import { installKeyboardStaging } from "./core/keyboard-staging.ts";
import { markRocketHost, ownsRocketElement } from "./core/ownership.ts";
import { installPointerDrag } from "./core/pointer-drag.ts";
import { installTargetIndicator } from "./core/visual-outlets.ts";
import { nextBentoColumn, projectBentoLayout, type Cell, type GridLayout } from "./placement.ts";

type Target = Cell & { grid: HTMLElement; gridId: string };

rocket("sb-bento-workspace", {
  mode: "light",
  manifest: {
    events: [
      {
        name: bentoContract.events.move,
        kind: "custom-event",
        bubbles: true,
        composed: true,
        description:
          "A tile was dropped on another cell or grid, or a keyboard move was released. detail: { itemId, fromGrid, " +
          "toGrid, updates }, where updates holds the new place { itemId, grid, col, row, width, height } of every " +
          "tile that changes in the grid the tile lands in, the tile included.",
      },
      {
        name: bentoContract.events.resize,
        kind: "custom-event",
        bubbles: true,
        composed: true,
        description:
          "A tile was resized by its handle or the keyboard. detail: { itemId, grid, updates }, with updates as for " +
          "the move.",
      },
    ],
  },
  setup({ host, cleanup }: { host: HTMLElement; cleanup: (fn: () => void) => void }) {
    cleanup(markRocketHost(host));
    const { grid: gridSelector, item: itemSelector, resize: resizeSelector } = bentoContract.selectors;
    const owns = (element: HTMLElement) => ownsRocketElement(host, element);
    const keyboard = keyboardBindings(host, {
      ...platformFocusKeys(navigator.platform),
      ...moveKeys,
      ...resizeKeys,
      ...gridKeys,
      ...cancelKeys,
    });
    const focus = installFocusRecovery(host);
    const grids = () => [...host.querySelectorAll<HTMLElement>(gridSelector)].filter(owns);
    const itemId = (item: HTMLElement) => (owns(item) ? (item.dataset.bentoItem ?? null) : null);
    const cells = (item: HTMLElement): Cell => ({
      col: Number(item.dataset.bentoCol ?? 1),
      row: Number(item.dataset.bentoRow ?? 1),
      width: Number(item.dataset.bentoWidth ?? 1),
      height: Number(item.dataset.bentoHeight ?? 1),
    });
    const columns = (grid: HTMLElement) => Math.max(1, Number(grid.dataset.columns ?? 4));
    const metrics = (grid: HTMLElement) => {
      const style = getComputedStyle(grid);
      const rect = grid.getBoundingClientRect();
      const gapX = parseFloat(style.columnGap) || 0;
      const gapY = parseFloat(style.rowGap) || 0;
      const left = rect.left + (parseFloat(style.borderLeftWidth) || 0) + (parseFloat(style.paddingLeft) || 0);
      const top = rect.top + (parseFloat(style.borderTopWidth) || 0) + (parseFloat(style.paddingTop) || 0);
      const contentWidth =
        rect.width -
        (parseFloat(style.borderLeftWidth) || 0) -
        (parseFloat(style.borderRightWidth) || 0) -
        (parseFloat(style.paddingLeft) || 0) -
        (parseFloat(style.paddingRight) || 0);
      return {
        left,
        top,
        gapX,
        gapY,
        cellWidth: (contentWidth - gapX * (columns(grid) - 1)) / columns(grid),
        cellHeight: parseFloat(style.gridAutoRows) || 80,
      };
    };
    const flip = installFlip({ host, itemSelector, itemId });
    const indicator = installTargetIndicator(host);
    let marker: HTMLElement | null = null;
    let previewKey = "";
    let previewUpdates: BentoPosition[] = [];
    let pendingTimer: ReturnType<typeof setTimeout> | null = null;
    const originalTransforms = new Map<HTMLElement, string>();
    let stretchedGrid: { element: HTMLElement; minHeight: string } | null = null;
    let projectedItem: HTMLElement | null = null;
    const layout = (): GridLayout[] =>
      grids().map((grid) => ({
        id: grid.dataset.bentoGrid ?? "",
        columns: columns(grid),
        items: [...grid.querySelectorAll<HTMLElement>(itemSelector)].filter(owns).map((item) => ({
          id: item.dataset.bentoItem ?? "",
          ...cells(item),
        })),
      }));
    const clearProjection = () => {
      indicator.clear();
      if (pendingTimer) clearTimeout(pendingTimer);
      pendingTimer = null;
      marker?.remove();
      marker = null;
      previewKey = "";
      previewUpdates = [];
      for (const [item, transform] of originalTransforms) item.style.transform = transform;
      originalTransforms.clear();
      projectedItem?.removeAttribute("data-bento-projecting");
      projectedItem = null;
      if (stretchedGrid) stretchedGrid.element.style.minHeight = stretchedGrid.minHeight;
      stretchedGrid = null;
    };
    const mark = (target: Target | null, id?: string) => {
      if (!target || !id) {
        clearProjection();
        return;
      }
      const key = `${id}:${target.gridId}:${target.col}:${target.row}:${target.width}:${target.height}`;
      if (key === previewKey) return;
      clearProjection();
      previewKey = key;
      previewUpdates = projectBentoLayout(layout(), id, target.gridId, target);
      marker = document.createElement("div");
      marker.setAttribute("data-bento-target", "");
      marker.style.position = "relative";
      marker.style.gridColumn = `${target.col} / span ${target.width}`;
      marker.style.gridRow = `${target.row} / span ${target.height}`;
      target.grid.append(marker);
      indicator.show(marker, "cell");
      const { cellWidth, cellHeight, gapX, gapY } = metrics(target.grid);
      const itemsById = new Map(
        [...host.querySelectorAll<HTMLElement>(itemSelector)]
          .filter(owns)
          .map((item) => [item.dataset.bentoItem, item]),
      );
      for (const update of previewUpdates) {
        const item = itemsById.get(update.itemId);
        if (!item) continue;
        if (update.itemId === id) {
          item.setAttribute("data-bento-projecting", "");
          projectedItem = item;
          continue;
        }
        const original = cells(item);
        originalTransforms.set(item, item.style.transform);
        item.style.transform =
          `translate(${(update.col - original.col) * (cellWidth + gapX)}px, ${(update.row - original.row) * (cellHeight + gapY)}px) ${item.style.transform}`.trim();
      }
      const furthestRow = Math.max(
        target.row + target.height - 1,
        ...previewUpdates.map((update) => update.row + update.height - 1),
      );
      const style = getComputedStyle(target.grid);
      stretchedGrid = { element: target.grid, minHeight: target.grid.style.minHeight };
      target.grid.style.minHeight = `${Math.max(
        parseFloat(style.minHeight) || 0,
        furthestRow * (cellHeight + gapY) -
          gapY +
          (parseFloat(style.paddingTop) || 0) +
          (parseFloat(style.paddingBottom) || 0) +
          (parseFloat(style.borderTopWidth) || 0) +
          (parseFloat(style.borderBottomWidth) || 0),
      )}px`;
    };
    const gridFor = (item: HTMLElement) => item.closest<HTMLElement>(gridSelector);
    const targetAt = (x: number, y: number, id: string): Target | null => {
      const grid = document.elementFromPoint(x, y)?.closest<HTMLElement>(gridSelector);
      const item = [...host.querySelectorAll<HTMLElement>(itemSelector)].find((candidate) => itemId(candidate) === id);
      if (!grid || !owns(grid) || !item || grid.dataset.bentoGrid === undefined) return null;
      const { left, top, gapX, gapY, cellWidth, cellHeight } = metrics(grid);
      const size = cells(item);
      const width = Math.min(size.width, columns(grid));
      return {
        grid,
        gridId: grid.dataset.bentoGrid,
        col: Math.max(1, Math.min(columns(grid) - width + 1, Math.floor((x - left) / (cellWidth + gapX)) + 1)),
        row: Math.max(1, Math.floor((y - top) / (cellHeight + gapY)) + 1),
        width,
        height: size.height,
      };
    };
    const finishPreview = () => {
      if (pendingTimer) clearTimeout(pendingTimer);
      pendingTimer = setTimeout(clearProjection, 2000);
    };
    // A DOM patch that reuses tiles by position would leave the focus on whichever tile takes the old place.
    const expectFocus = (id: string, target: Target) => {
      const source = [...host.querySelectorAll<HTMLElement>(itemSelector)].find(
        (candidate) => itemId(candidate) === id,
      );
      if (source?.contains(document.activeElement))
        focus.expect(source, () => {
          const item = [...host.querySelectorAll<HTMLElement>(itemSelector)].find(
            (candidate) => itemId(candidate) === id,
          );
          if (!item || gridFor(item)?.dataset.bentoGrid !== target.gridId) return null;
          const position = cells(item);
          return position.col === target.col &&
            position.row === target.row &&
            position.width === target.width &&
            position.height === target.height
            ? item
            : null;
        });
    };
    const emitMove = (id: string, target: Target) => {
      const item = [...host.querySelectorAll<HTMLElement>(itemSelector)].find((candidate) => itemId(candidate) === id);
      const fromGrid = item && gridFor(item)?.dataset.bentoGrid;
      if (fromGrid === undefined) return;
      if (!previewUpdates.length) {
        clearProjection();
        return;
      }
      host.dispatchEvent(
        new CustomEvent<BentoMoveDetail>(bentoContract.events.move, {
          bubbles: true,
          composed: true,
          detail: {
            itemId: id,
            fromGrid,
            toGrid: target.gridId,
            updates: previewUpdates,
          },
        }),
      );
      finishPreview();
    };
    const emitResize = (id: string, target: Target) => {
      if (!previewUpdates.length) {
        clearProjection();
        return;
      }
      host.dispatchEvent(
        new CustomEvent<BentoResizeDetail>(bentoContract.events.resize, {
          bubbles: true,
          composed: true,
          detail: { itemId: id, grid: target.gridId, updates: previewUpdates },
        }),
      );
      finishPreview();
    };
    const pointerDispose = installPointerDrag({
      host,
      itemSelector,
      itemId,
      targetAt,
      sameTarget: (a, b) =>
        a?.grid === b?.grid &&
        a?.col === b?.col &&
        a?.row === b?.row &&
        a?.width === b?.width &&
        a?.height === b?.height,
      mark,
      retainPreviewOnCommit: true,
      canStart: (event) => !(event.target as HTMLElement).closest(resizeSelector),
      beforeCommit: (id, rect) => flip.prepare({ itemId: id, rect }),
      commit: (id, target) => {
        expectFocus(id, target);
        emitMove(id, target);
      },
    });

    let resizing: {
      item: HTMLElement;
      pointerId: number;
      x: number;
      y: number;
      origin: Target;
      target: Target;
    } | null = null;
    const clearResize = () => {
      if (resizing) {
        resizing.item.removeAttribute("data-bento-resizing");
        try {
          host.releasePointerCapture?.(resizing.pointerId);
        } catch {
          /* Already released. */
        }
      }
      resizing = null;
      host.removeAttribute("data-resize-active");
      mark(null);
    };
    const onPointerDown = (event: PointerEvent) => {
      if (!owns(event.target as HTMLElement)) return;
      if (staging.current) staging.cancel();
      const handle = (event.target as HTMLElement).closest<HTMLElement>(resizeSelector);
      const item = handle?.closest<HTMLElement>(itemSelector);
      const grid = item && gridFor(item);
      if (!handle || !item || !grid || !owns(item) || event.button !== 0 || resizing) return;
      event.preventDefault();
      const target = { grid, gridId: grid.dataset.bentoGrid ?? "", ...cells(item) };
      resizing = { item, pointerId: event.pointerId, x: event.clientX, y: event.clientY, origin: target, target };
      item.setAttribute("data-bento-resizing", "");
      host.setAttribute("data-resize-active", "");
      host.setPointerCapture?.(event.pointerId);
      mark(target, item.dataset.bentoItem);
    };
    const onPointerMove = (event: PointerEvent) => {
      if (!resizing || resizing.pointerId !== event.pointerId) return;
      const { cellWidth, cellHeight, gapX, gapY } = metrics(resizing.origin.grid);
      resizing.target = {
        ...resizing.origin,
        width: Math.max(
          1,
          Math.min(
            columns(resizing.origin.grid) - resizing.origin.col + 1,
            resizing.origin.width + Math.round((event.clientX - resizing.x) / (cellWidth + gapX)),
          ),
        ),
        height: Math.max(
          1,
          Math.min(5, resizing.origin.height + Math.round((event.clientY - resizing.y) / (cellHeight + gapY))),
        ),
      };
      mark(resizing.target, resizing.item.dataset.bentoItem);
    };
    const onPointerUp = (event: PointerEvent) => {
      if (!resizing || resizing.pointerId !== event.pointerId) return;
      const { item, origin, target } = resizing;
      if (origin.width === target.width && origin.height === target.height) {
        clearResize();
        return;
      }
      resizing.item.removeAttribute("data-bento-resizing");
      try {
        host.releasePointerCapture?.(resizing.pointerId);
      } catch {
        /* Already released. */
      }
      resizing = null;
      host.removeAttribute("data-resize-active");
      expectFocus(item.dataset.bentoItem ?? "", target);
      flip.prepare();
      emitResize(item.dataset.bentoItem ?? "", target);
    };
    const onPointerCancel = (event: PointerEvent) => {
      if (resizing?.pointerId === event.pointerId) clearResize();
    };

    const staging = installKeyboardStaging<{ id: string; target: Target; kind: "move" | "resize" }>({
      host,
      owns,
      onCancel: () => mark(null),
      onCommit: ({ id, target, kind }) => {
        expectFocus(id, target);
        flip.prepare();
        if (kind === "move") emitMove(id, target);
        else emitResize(id, target);
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
        if (event.target !== item) return;
        const items = [...host.querySelectorAll<HTMLElement>(itemSelector)].filter(owns);
        let next = keyboard.matches("focusFirst", event)
          ? items[0]
          : keyboard.matches("focusLast", event)
            ? items.at(-1)
            : null;
        const direction =
          focusDirection.x < 0
            ? { x: -1, y: 0 }
            : focusDirection.x > 0
              ? { x: 1, y: 0 }
              : focusDirection.y < 0
                ? { x: 0, y: -1 }
                : focusDirection.y > 0
                  ? { x: 0, y: 1 }
                  : null;
        if (direction) {
          const rect = item.getBoundingClientRect();
          const x = rect.left + rect.width / 2,
            y = rect.top + rect.height / 2;
          const nearest = (candidates: HTMLElement[], crossBoard = false) =>
            candidates
              .map((candidate) => {
                const box = candidate.getBoundingClientRect();
                const dx = box.left + box.width / 2 - x,
                  dy = box.top + box.height / 2 - y;
                const along = dx * direction.x + dy * direction.y;
                const across = Math.abs(dx * direction.y - dy * direction.x);
                return { candidate, along, across, score: (crossBoard ? 0 : along) + across * 2 };
              })
              .filter((option) => crossBoard || (option.along > 1 && option.across <= option.along))
              .sort((a, b) => a.score - b.score)[0]?.candidate;
          const currentGrid = gridFor(item);
          next = nearest(items.filter((candidate) => candidate !== item && gridFor(candidate) === currentGrid));
          if (!next && currentGrid) {
            const allGrids = grids();
            const step = direction.x || direction.y;
            const neighbor = allGrids[allGrids.indexOf(currentGrid) + step];
            if (neighbor)
              next = nearest(
                items.filter((candidate) => gridFor(candidate) === neighbor),
                true,
              );
          }
        }
        if (next) {
          event.preventDefault();
          next.focus();
        }
        return;
      }
      const moveDirection = keyboard.direction(event, "move");
      const resizeDirection = keyboard.direction(event, "resize");
      const gridDirection = keyboard.matches("gridPrevious", event) ? -1 : keyboard.matches("gridNext", event) ? 1 : 0;
      const move = !!(moveDirection.x || moveDirection.y || gridDirection);
      const resize = !!(resizeDirection.x || resizeDirection.y);
      if (!move && !resize) return;
      const { x: dx, y: dy } = move ? moveDirection : resizeDirection;
      if (!dx && !dy && !gridDirection) return;
      const sourceGrid = gridFor(item);
      const current =
        staging.current?.id === id && staging.current.kind === (move ? "move" : "resize")
          ? staging.current.target
          : null;
      const currentGrid = current?.grid ?? sourceGrid;
      const allGrids = grids();
      let targetGrid = currentGrid && allGrids[allGrids.indexOf(currentGrid) + gridDirection];
      let crossing = !!gridDirection;
      if (move && dx && currentGrid && !gridDirection) {
        const currentCol = current?.col ?? cells(item).col;
        const currentWidth = current?.width ?? cells(item).width;
        if (nextBentoColumn(currentCol, currentWidth, columns(currentGrid), dx) === null) {
          targetGrid = allGrids[allGrids.indexOf(currentGrid) + dx];
          crossing = !!targetGrid;
        }
      }
      if (!targetGrid) return;
      const origin = current ?? { grid: targetGrid, gridId: targetGrid.dataset.bentoGrid ?? "", ...cells(item) };
      const width = resize
        ? Math.max(1, Math.min(columns(targetGrid) - origin.col + 1, origin.width + dx))
        : Math.min(origin.width, columns(targetGrid));
      const target: Target = {
        grid: targetGrid,
        gridId: targetGrid.dataset.bentoGrid ?? "",
        col: resize
          ? origin.col
          : crossing && dx
            ? dx > 0
              ? 1
              : columns(targetGrid) - width + 1
            : Math.max(1, Math.min(columns(targetGrid) - width + 1, origin.col + dx)),
        row: resize ? origin.row : Math.max(1, origin.row + dy),
        width,
        height: resize ? Math.max(1, Math.min(5, origin.height + dy)) : origin.height,
      };
      if (gridDirection) {
        target.col = 1;
        target.row = 1;
      }
      if (
        !crossing &&
        target.col === origin.col &&
        target.row === origin.row &&
        target.width === origin.width &&
        target.height === origin.height
      )
        return;
      event.preventDefault();
      staging.set(item, { id, target, kind: move ? "move" : "resize" }, event);
      mark(target, id);
    };
    host.addEventListener("pointerdown", onPointerDown);
    host.addEventListener("pointermove", onPointerMove);
    host.addEventListener("pointerup", onPointerUp);
    host.addEventListener("pointercancel", onPointerCancel);
    host.addEventListener("keydown", onKeyDown);
    cleanup(() => {
      staging.dispose();
      clearResize();
      clearProjection();
      pointerDispose();
      flip.dispose();
      focus.dispose();
      host.removeEventListener("pointerdown", onPointerDown);
      host.removeEventListener("pointermove", onPointerMove);
      host.removeEventListener("pointerup", onPointerUp);
      host.removeEventListener("pointercancel", onPointerCancel);
      host.removeEventListener("keydown", onKeyDown);
    });
  },
});
