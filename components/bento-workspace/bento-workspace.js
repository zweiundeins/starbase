// Generated from bento-workspace.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import { rocket } from "datastar";
import { bentoContract } from "./contracts/bento.js";
import { installFlip } from "./core/flip.js";
import { installFocusRecovery } from "./core/focus-recovery.js";
import { cancelKeys, gridKeys, keyboardBindings, keyboardItem, moveKeys, platformFocusKeys, resizeKeys, } from "./core/keyboard.js";
import { installKeyboardStaging } from "./core/keyboard-staging.js";
import { markRocketHost, ownsRocketElement } from "./core/ownership.js";
import { installPointerDrag } from "./core/pointer-drag.js";
import { installTargetIndicator } from "./core/visual-outlets.js";
import { nextBentoColumn, projectBentoLayout } from "./placement.js";
rocket("sb-bento-workspace", {
    mode: "light",
    setup({ host, cleanup }) {
        cleanup(markRocketHost(host));
        const { grid: gridSelector, item: itemSelector, resize: resizeSelector } = bentoContract.selectors;
        const owns = (element) => ownsRocketElement(host, element);
        const keyboard = keyboardBindings(host, {
            ...platformFocusKeys(navigator.platform),
            ...moveKeys,
            ...resizeKeys,
            ...gridKeys,
            ...cancelKeys,
        });
        const focus = installFocusRecovery(host);
        const grids = () => [...host.querySelectorAll(gridSelector)].filter(owns);
        const itemId = (item) => (owns(item) ? (item.dataset.bentoItem ?? null) : null);
        const cells = (item) => ({
            col: Number(item.dataset.bentoCol ?? 1),
            row: Number(item.dataset.bentoRow ?? 1),
            width: Number(item.dataset.bentoWidth ?? 1),
            height: Number(item.dataset.bentoHeight ?? 1),
        });
        const columns = (grid) => Math.max(1, Number(grid.dataset.columns ?? 4));
        const metrics = (grid) => {
            const style = getComputedStyle(grid);
            const rect = grid.getBoundingClientRect();
            const gapX = parseFloat(style.columnGap) || 0;
            const gapY = parseFloat(style.rowGap) || 0;
            const left = rect.left + (parseFloat(style.borderLeftWidth) || 0) + (parseFloat(style.paddingLeft) || 0);
            const top = rect.top + (parseFloat(style.borderTopWidth) || 0) + (parseFloat(style.paddingTop) || 0);
            const contentWidth = rect.width -
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
        let marker = null;
        let previewKey = "";
        let previewUpdates = [];
        let pendingTimer = null;
        const originalTransforms = new Map();
        let stretchedGrid = null;
        let projectedItem = null;
        const layout = () => grids().map((grid) => ({
            id: grid.dataset.bentoGrid ?? "",
            columns: columns(grid),
            items: [...grid.querySelectorAll(itemSelector)].filter(owns).map((item) => ({
                id: item.dataset.bentoItem ?? "",
                ...cells(item),
            })),
        }));
        const clearProjection = () => {
            indicator.clear();
            if (pendingTimer)
                clearTimeout(pendingTimer);
            pendingTimer = null;
            marker?.remove();
            marker = null;
            previewKey = "";
            previewUpdates = [];
            for (const [item, transform] of originalTransforms)
                item.style.transform = transform;
            originalTransforms.clear();
            projectedItem?.removeAttribute("data-bento-projecting");
            projectedItem = null;
            if (stretchedGrid)
                stretchedGrid.element.style.minHeight = stretchedGrid.minHeight;
            stretchedGrid = null;
        };
        const mark = (target, id) => {
            if (!target || !id) {
                clearProjection();
                return;
            }
            const key = `${id}:${target.gridId}:${target.col}:${target.row}:${target.width}:${target.height}`;
            if (key === previewKey)
                return;
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
            const itemsById = new Map([...host.querySelectorAll(itemSelector)]
                .filter(owns)
                .map((item) => [item.dataset.bentoItem, item]));
            for (const update of previewUpdates) {
                const item = itemsById.get(update.itemId);
                if (!item)
                    continue;
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
            const furthestRow = Math.max(target.row + target.height - 1, ...previewUpdates.map((update) => update.row + update.height - 1));
            const style = getComputedStyle(target.grid);
            stretchedGrid = { element: target.grid, minHeight: target.grid.style.minHeight };
            target.grid.style.minHeight = `${Math.max(parseFloat(style.minHeight) || 0, furthestRow * (cellHeight + gapY) -
                gapY +
                (parseFloat(style.paddingTop) || 0) +
                (parseFloat(style.paddingBottom) || 0) +
                (parseFloat(style.borderTopWidth) || 0) +
                (parseFloat(style.borderBottomWidth) || 0))}px`;
        };
        const gridFor = (item) => item.closest(gridSelector);
        const targetAt = (x, y, id) => {
            const grid = document.elementFromPoint(x, y)?.closest(gridSelector);
            const item = [...host.querySelectorAll(itemSelector)].find((candidate) => itemId(candidate) === id);
            if (!grid || !owns(grid) || !item || grid.dataset.bentoGrid === undefined)
                return null;
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
            if (pendingTimer)
                clearTimeout(pendingTimer);
            pendingTimer = setTimeout(clearProjection, 2000);
        };
        const emitMove = (id, target) => {
            const item = [...host.querySelectorAll(itemSelector)].find((candidate) => itemId(candidate) === id);
            const fromGrid = item && gridFor(item)?.dataset.bentoGrid;
            if (fromGrid === undefined)
                return;
            if (!previewUpdates.length) {
                clearProjection();
                return;
            }
            host.dispatchEvent(new CustomEvent(bentoContract.events.move, {
                bubbles: true,
                composed: true,
                detail: {
                    itemId: id,
                    fromGrid,
                    toGrid: target.gridId,
                    updates: previewUpdates,
                },
            }));
            finishPreview();
        };
        const emitResize = (id, target) => {
            if (!previewUpdates.length) {
                clearProjection();
                return;
            }
            host.dispatchEvent(new CustomEvent(bentoContract.events.resize, {
                bubbles: true,
                composed: true,
                detail: { itemId: id, grid: target.gridId, updates: previewUpdates },
            }));
            finishPreview();
        };
        const pointerDispose = installPointerDrag({
            host,
            itemSelector,
            itemId,
            targetAt,
            sameTarget: (a, b) => a?.grid === b?.grid &&
                a?.col === b?.col &&
                a?.row === b?.row &&
                a?.width === b?.width &&
                a?.height === b?.height,
            mark,
            retainPreviewOnCommit: true,
            canStart: (event) => !event.target.closest(resizeSelector),
            beforeCommit: (id, rect) => flip.prepare({ itemId: id, rect }),
            commit: emitMove,
        });
        let resizing = null;
        const clearResize = () => {
            if (resizing) {
                resizing.item.removeAttribute("data-bento-resizing");
                try {
                    host.releasePointerCapture?.(resizing.pointerId);
                }
                catch {
                    /* Already released. */
                }
            }
            resizing = null;
            host.removeAttribute("data-resize-active");
            mark(null);
        };
        const onPointerDown = (event) => {
            if (!owns(event.target))
                return;
            if (staging.current)
                staging.cancel();
            const handle = event.target.closest(resizeSelector);
            const item = handle?.closest(itemSelector);
            const grid = item && gridFor(item);
            if (!handle || !item || !grid || !owns(item) || event.button !== 0 || resizing)
                return;
            event.preventDefault();
            const target = { grid, gridId: grid.dataset.bentoGrid ?? "", ...cells(item) };
            resizing = { item, pointerId: event.pointerId, x: event.clientX, y: event.clientY, origin: target, target };
            item.setAttribute("data-bento-resizing", "");
            host.setAttribute("data-resize-active", "");
            host.setPointerCapture?.(event.pointerId);
            mark(target, item.dataset.bentoItem);
        };
        const onPointerMove = (event) => {
            if (!resizing || resizing.pointerId !== event.pointerId)
                return;
            const { cellWidth, cellHeight, gapX, gapY } = metrics(resizing.origin.grid);
            resizing.target = {
                ...resizing.origin,
                width: Math.max(1, Math.min(columns(resizing.origin.grid) - resizing.origin.col + 1, resizing.origin.width + Math.round((event.clientX - resizing.x) / (cellWidth + gapX)))),
                height: Math.max(1, Math.min(5, resizing.origin.height + Math.round((event.clientY - resizing.y) / (cellHeight + gapY)))),
            };
            mark(resizing.target, resizing.item.dataset.bentoItem);
        };
        const onPointerUp = (event) => {
            if (!resizing || resizing.pointerId !== event.pointerId)
                return;
            const { item, origin, target } = resizing;
            if (origin.width === target.width && origin.height === target.height) {
                clearResize();
                return;
            }
            resizing.item.removeAttribute("data-bento-resizing");
            try {
                host.releasePointerCapture?.(resizing.pointerId);
            }
            catch {
                /* Already released. */
            }
            resizing = null;
            host.removeAttribute("data-resize-active");
            flip.prepare();
            emitResize(item.dataset.bentoItem ?? "", target);
        };
        const onPointerCancel = (event) => {
            if (resizing?.pointerId === event.pointerId)
                clearResize();
        };
        const staging = installKeyboardStaging({
            host,
            owns,
            onCancel: () => mark(null),
            onCommit: ({ id, target, kind }) => {
                const source = [...host.querySelectorAll(itemSelector)].find((candidate) => itemId(candidate) === id);
                if (source)
                    focus.expect(source, () => {
                        const item = [...host.querySelectorAll(itemSelector)].find((candidate) => itemId(candidate) === id);
                        if (!item || gridFor(item)?.dataset.bentoGrid !== target.gridId)
                            return null;
                        const position = cells(item);
                        return position.col === target.col &&
                            position.row === target.row &&
                            position.width === target.width &&
                            position.height === target.height
                            ? item
                            : null;
                    });
                flip.prepare();
                if (kind === "move")
                    emitMove(id, target);
                else
                    emitResize(id, target);
            },
        });
        const onKeyDown = (event) => {
            const item = keyboardItem(event, itemSelector, owns);
            const id = item && itemId(item);
            if (!item || !id)
                return;
            if (keyboard.matches("cancel", event)) {
                if (staging.current) {
                    event.preventDefault();
                    staging.cancel();
                }
                return;
            }
            const focusDirection = keyboard.direction(event, "focus");
            if (focusDirection.x ||
                focusDirection.y ||
                keyboard.matches("focusFirst", event) ||
                keyboard.matches("focusLast", event)) {
                if (event.target !== item)
                    return;
                const items = [...host.querySelectorAll(itemSelector)].filter(owns);
                let next = keyboard.matches("focusFirst", event)
                    ? items[0]
                    : keyboard.matches("focusLast", event)
                        ? items.at(-1)
                        : null;
                const direction = focusDirection.x < 0
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
                    const x = rect.left + rect.width / 2, y = rect.top + rect.height / 2;
                    const nearest = (candidates, crossBoard = false) => candidates
                        .map((candidate) => {
                        const box = candidate.getBoundingClientRect();
                        const dx = box.left + box.width / 2 - x, dy = box.top + box.height / 2 - y;
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
                            next = nearest(items.filter((candidate) => gridFor(candidate) === neighbor), true);
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
            if (!move && !resize)
                return;
            const { x: dx, y: dy } = move ? moveDirection : resizeDirection;
            if (!dx && !dy && !gridDirection)
                return;
            const sourceGrid = gridFor(item);
            const current = staging.current?.id === id && staging.current.kind === (move ? "move" : "resize")
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
            if (!targetGrid)
                return;
            const origin = current ?? { grid: targetGrid, gridId: targetGrid.dataset.bentoGrid ?? "", ...cells(item) };
            const width = resize
                ? Math.max(1, Math.min(columns(targetGrid) - origin.col + 1, origin.width + dx))
                : Math.min(origin.width, columns(targetGrid));
            const target = {
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
            if (!crossing &&
                target.col === origin.col &&
                target.row === origin.row &&
                target.width === origin.width &&
                target.height === origin.height)
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
