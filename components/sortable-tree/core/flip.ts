// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export type FlipOptions = {
  host: HTMLElement;
  itemSelector: string;
  itemId: (item: HTMLElement) => string | null;
};

type Rect = { left: number; top: number };
export type FlipOrigin = { itemId: string; rect: Rect };

function capture(options: FlipOptions): Map<string, Rect> {
  const result = new Map<string, Rect>();
  options.host.querySelectorAll<HTMLElement>(options.itemSelector).forEach((item) => {
    const id = options.itemId(item);
    if (id) {
      const rect = item.getBoundingClientRect();
      result.set(id, { left: rect.left, top: rect.top });
    }
  });
  return result;
}

function play(options: FlipOptions, before: Map<string, Rect>): void {
  options.host.querySelectorAll<HTMLElement>(options.itemSelector).forEach((item) => {
    const id = options.itemId(item);
    const first = id ? before.get(id) : undefined;
    if (!first) return;
    const last = item.getBoundingClientRect();
    const dx = first.left - last.left;
    const dy = first.top - last.top;
    if (Math.abs(dx) < 1 && Math.abs(dy) < 1) return;
    item.animate([{ transform: `translate(${dx}px, ${dy}px)` }, { transform: "translate(0, 0)" }], {
      duration: 180,
      easing: "cubic-bezier(.2, 0, 0, 1)",
    });
  });
}

/** Watches a host for the DOM change caused by a semantic move, including a later SSE morph. */
export function installFlip(options: FlipOptions): { prepare: (origin?: FlipOrigin) => void; dispose: () => void } {
  let before: Map<string, Rect> | null = null;
  let timer: ReturnType<typeof setTimeout> | null = null;
  let frame: number | null = null;

  const clearPending = (): void => {
    before = null;
    if (timer !== null) clearTimeout(timer);
    if (frame !== null) cancelAnimationFrame(frame);
    timer = null;
    frame = null;
  };

  const finish = (): void => {
    if (!before) return;
    const snapshot = before;
    clearPending();
    play(options, snapshot);
  };

  const containsItem = (node: Node): boolean =>
    node instanceof Element && (node.matches(options.itemSelector) || !!node.querySelector(options.itemSelector));
  const observer = new MutationObserver((records) => {
    if (
      !before ||
      frame !== null ||
      !records.some((record) => [...record.addedNodes, ...record.removedNodes].some(containsItem))
    )
      return;
    frame = requestAnimationFrame(finish);
  });
  observer.observe(options.host, { childList: true, subtree: true });

  return {
    prepare: (origin) => {
      clearPending();
      before = capture(options);
      if (origin) before.set(origin.itemId, origin.rect);
      timer = setTimeout(clearPending, 2000);
    },
    dispose: () => {
      observer.disconnect();
      clearPending();
    },
  };
}
