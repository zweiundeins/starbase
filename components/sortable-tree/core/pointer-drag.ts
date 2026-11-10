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

import { createDragState } from "./drag-state.ts";
import { ownsRocketElement } from "./ownership.ts";
import { dragPreviewFor } from "./visual-outlets.ts";

export type PointerDragOptions<ItemId, Target> = {
  host: HTMLElement;
  itemSelector: string;
  itemId: (item: HTMLElement) => ItemId | null;
  interactiveHandle?: string;
  canStart?: (event: PointerEvent, item: HTMLElement) => boolean;
  targetAt: (x: number, y: number, itemId: ItemId) => Target | null;
  sameTarget?: (previous: Target | null, next: Target | null) => boolean;
  mark: (target: Target | null, itemId?: ItemId) => void;
  retainPreviewOnCommit?: boolean;
  beforeCommit?: (itemId: ItemId, rect: { left: number; top: number }) => void;
  commit: (itemId: ItemId, target: Target) => void;
};

/** Shared pointer lifecycle; target geometry and semantic commit stay domain-owned. */
export function installPointerDrag<ItemId, Target>(options: PointerDragOptions<ItemId, Target>): () => void {
  const state = createDragState<ItemId, Target>();
  let active: { itemId: ItemId; pointerId: number } | null = null;
  let preview: HTMLElement | null = null;
  let previewOffset = { x: 0, y: 0 };
  let start: { x: number; y: number; item: HTMLElement } | null = null;
  let dragging = false;
  let previewTarget: Target | null = null;
  let hasPreviewTarget = false;

  const movePreview = (event: PointerEvent): void => {
    if (!preview) return;
    preview.style.left = `${event.clientX - previewOffset.x}px`;
    preview.style.top = `${event.clientY - previewOffset.y}px`;
  };

  const clearPreview = (): void => {
    preview?.remove();
    preview = null;
  };

  const finish = (target: Target | null, cancelled: boolean): void => {
    const current = active;
    const previewPosition = preview && { left: parseFloat(preview.style.left), top: parseFloat(preview.style.top) };
    active = null;
    start = null;
    dragging = false;
    previewTarget = null;
    hasPreviewTarget = false;
    if (!target || cancelled || !options.retainPreviewOnCommit) options.mark(null, current?.itemId);
    clearPreview();
    options.host.removeAttribute("data-drag-active");
    options.host.querySelectorAll<HTMLElement>("[data-dragging]").forEach((item) => {
      if (ownsRocketElement(options.host, item)) item.removeAttribute("data-dragging");
    });
    if (!current) return;
    try {
      options.host.releasePointerCapture?.(current.pointerId);
    } catch {
      // Pointer capture may already have been released by the browser.
    }
    if (cancelled) {
      state.send({ type: "cancel" });
      return;
    }
    if (target) {
      state.send({ type: "preview", target });
      state.send({ type: "commit", target });
      if (previewPosition) options.beforeCommit?.(current.itemId, previewPosition);
      options.commit(current.itemId, target);
    } else {
      state.send({ type: "cancel" });
    }
  };

  const onMove = (event: Event): void => {
    if (!active || (event as PointerEvent).pointerId !== active.pointerId) return;
    const pointer = event as PointerEvent;
    if (!dragging) {
      if (!start || Math.hypot(pointer.clientX - start.x, pointer.clientY - start.y) < 5) return;
      dragging = true;
      beginPreview(start.item, pointer);
    }
    movePreview(pointer);
    const target = options.targetAt(pointer.clientX, pointer.clientY, active.itemId);
    if (hasPreviewTarget && options.sameTarget?.(previewTarget, target)) return;
    previewTarget = target;
    hasPreviewTarget = true;
    state.send({ type: "preview", target });
    options.mark(target, active.itemId);
  };
  const onUp = (event: Event): void => {
    if (!active || (event as PointerEvent).pointerId !== active.pointerId) return;
    const pointer = event as PointerEvent;
    if (!dragging) {
      finish(null, true);
      return;
    }
    movePreview(pointer);
    finish(options.targetAt(pointer.clientX, pointer.clientY, active.itemId), false);
  };
  const onCancel = (event: Event): void => {
    if (!active || (event as PointerEvent).pointerId !== active.pointerId) return;
    finish(null, true);
  };
  const beginPreview = (item: HTMLElement, pointer: PointerEvent): void => {
    options.host.setAttribute("data-drag-active", "");
    item.setAttribute("data-dragging", "true");
    const rect = item.getBoundingClientRect();
    previewOffset = { x: start!.x - rect.left, y: start!.y - rect.top };
    const customPreview = !!item.querySelector(":scope > template[data-sb-preview]");
    preview = dragPreviewFor(item);
    preview.removeAttribute("id");
    preview.setAttribute("data-drag-preview", "true");
    preview.setAttribute("aria-hidden", "true");
    preview.inert = true;
    preview.style.position = "fixed";
    preview.style.left = `${rect.left}px`;
    preview.style.top = `${rect.top}px`;
    preview.style.boxSizing = "border-box";
    preview.style.setProperty("--sb-source-width", `${rect.width}px`);
    preview.style.setProperty("--sb-source-height", `${rect.height}px`);
    if (!customPreview) {
      preview.style.width = `${rect.width}px`;
      preview.style.height = `${rect.height}px`;
    }
    preview.style.pointerEvents = "none";
    preview.style.zIndex = "1000";
    document.body.append(preview);
    if (customPreview) {
      previewOffset = {
        x: Math.min(1, Math.max(0, previewOffset.x / rect.width)) * preview.offsetWidth,
        y: Math.min(1, Math.max(0, previewOffset.y / rect.height)) * preview.offsetHeight,
      };
    }
    options.host.setPointerCapture?.(pointer.pointerId);
  };
  // Links and images are draggable: the browser's own drag would send pointercancel and end this one.
  const onDragStart = (event: Event): void => {
    if (active) event.preventDefault();
  };
  const onDown = (event: Event): void => {
    const pointer = event as PointerEvent;
    if (pointer.button !== 0 || active) return;
    const target = pointer.target as HTMLElement;
    if (!ownsRocketElement(options.host, target)) return;
    if (
      target.closest("button, a, input, select, textarea, [contenteditable]:not([contenteditable='false'])") &&
      !(options.interactiveHandle && target.closest(options.interactiveHandle))
    )
      return;
    const item = target.closest<HTMLElement>(options.itemSelector);
    const itemId = item && ownsRocketElement(options.host, item) ? options.itemId(item) : null;
    if (!item || itemId === null || (options.canStart && !options.canStart(pointer, item))) return;
    active = { itemId, pointerId: pointer.pointerId };
    start = { x: pointer.clientX, y: pointer.clientY, item };
    state.send({ type: "begin", itemId });
  };

  options.host.addEventListener("pointerdown", onDown);
  options.host.addEventListener("dragstart", onDragStart);
  window.addEventListener("pointermove", onMove);
  window.addEventListener("pointerup", onUp);
  window.addEventListener("pointercancel", onCancel);
  return () => {
    finish(null, true);
    options.host.removeEventListener("pointerdown", onDown);
    options.host.removeEventListener("dragstart", onDragStart);
    window.removeEventListener("pointermove", onMove);
    window.removeEventListener("pointerup", onUp);
    window.removeEventListener("pointercancel", onCancel);
  };
}
