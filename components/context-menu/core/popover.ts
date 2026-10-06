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

type Box = { left: number; top: number; right: number; bottom: number; width: number; height: number };

/** Position in the viewport, flipping submenus before clamping. */
export function popoverPosition(
  anchor: Box | { x: number; y: number },
  size: { width: number; height: number },
  viewport: { width: number; height: number },
  side: "below" | "beside" | "point",
): { left: number; top: number } {
  const gap = side === "point" ? 0 : 6;
  const x = "x" in anchor ? anchor.x : anchor.left;
  const y = "y" in anchor ? anchor.y : anchor.bottom;
  const left =
    side === "beside" && "right" in anchor
      ? anchor.right + gap + size.width > viewport.width - 8
        ? anchor.left - size.width - gap
        : anchor.right + gap
      : x;
  const top =
    side === "below" && y + size.height > viewport.height - 8 && "top" in anchor
      ? anchor.top - size.height - gap
      : side === "beside" && "top" in anchor
        ? anchor.top
        : y + gap;
  return {
    left: Math.max(8, Math.min(left, viewport.width - size.width - 8)),
    top: Math.max(8, Math.min(top, viewport.height - size.height - 8)),
  };
}

let anchorSequence = 0;

export function showPositionedPopover(
  popover: HTMLElement,
  anchor: HTMLElement | { x: number; y: number },
  side: "below" | "beside" | "point",
): () => void {
  if (CSS.supports("position-anchor: --sb-menu-anchor") && CSS.supports("top: anchor(bottom)")) {
    const name = `--sb-menu-anchor-${++anchorSequence}`;
    const point = anchor instanceof HTMLElement ? null : document.createElement("span");
    const target = point ?? (anchor as HTMLElement);
    const previousAnchor = target.style.getPropertyValue("anchor-name");
    const properties = ["position", "inset", "left", "top", "position-anchor", "position-try-fallbacks"] as const;
    const previous = properties.map((property) => popover.style.getPropertyValue(property));
    if (point && !(anchor instanceof HTMLElement)) {
      point.setAttribute("aria-hidden", "true");
      point.style.cssText = `position:fixed;left:${anchor.x}px;top:${anchor.y}px;width:0;height:0;pointer-events:none`;
      document.body.append(point);
    }
    target.style.setProperty("anchor-name", name);
    popover.style.position = "fixed";
    popover.style.inset = "auto";
    popover.style.setProperty("position-anchor", name);
    popover.style.left = side === "beside" ? "calc(anchor(right) + 6px)" : "anchor(left)";
    popover.style.top =
      side === "beside" ? "anchor(top)" : side === "point" ? "anchor(bottom)" : "calc(anchor(bottom) + 6px)";
    popover.style.setProperty("position-try-fallbacks", "flip-inline, flip-block, flip-inline flip-block");
    popover.showPopover();
    const bounds = popover.getBoundingClientRect();
    if (bounds.left < 8 || bounds.right > innerWidth - 8 || bounds.top < 8 || bounds.bottom > innerHeight - 8) {
      const rect = anchor instanceof HTMLElement ? anchor.getBoundingClientRect() : anchor;
      const position = popoverPosition(rect, bounds, { width: innerWidth, height: innerHeight }, side);
      if (bounds.left < 8 || bounds.right > innerWidth - 8) popover.style.left = `${position.left}px`;
      if (bounds.top < 8 || bounds.bottom > innerHeight - 8) popover.style.top = `${position.top}px`;
    }
    return () => {
      if (previousAnchor) target.style.setProperty("anchor-name", previousAnchor);
      else target.style.removeProperty("anchor-name");
      point?.remove();
      properties.forEach((property, index) => {
        if (previous[index]) popover.style.setProperty(property, previous[index]!);
        else popover.style.removeProperty(property);
      });
    };
  }
  popover.showPopover();
  const size = popover.getBoundingClientRect();
  const rect = anchor instanceof HTMLElement ? anchor.getBoundingClientRect() : anchor;
  const { left, top } = popoverPosition(rect, size, { width: innerWidth, height: innerHeight }, side);
  popover.style.position = "fixed";
  popover.style.inset = "auto";
  popover.style.left = `${left}px`;
  popover.style.top = `${top}px`;
  return () => {};
}
