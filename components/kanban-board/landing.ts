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

import { kanbanContract, type KanbanLandingEndDetail } from "./contracts/kanban.ts";

type Rect = { left: number; top: number; width: number; height: number };

/** Where a pending landing's marker goes now, the room it opens there, and the element whose scrolling clips it. */
export type LandingPlace = {
  rect: Rect;
  gap: { element: HTMLElement; side: "top" | "bottom"; size: number } | null;
  clip: HTMLElement;
};

/** A move the host hasn't confirmed: the marker shows the person's drop until the move arrives or is released. */
export type LandingSpec = {
  key: string;
  detail: { cardId?: string; col: number };
  marker: HTMLElement;
  origin: { left: number; top: number };
  place: () => LandingPlace | null;
  arrived: () => boolean;
  /** Arms the move's animation with the marker as its starting point; `group` is the animation it arms. */
  prepare: (rect: Rect) => void;
  group: object;
};

type Pending = {
  spec: LandingSpec;
  timer: ReturnType<typeof setTimeout>;
  placed: boolean;
  gap: { element: HTMLElement; side: "top" | "bottom"; size: number; animation: Animation } | null;
};

const reducedMotion = () => matchMedia("(prefers-reduced-motion: reduce)").matches;
const easing = "cubic-bezier(.2, 0, 0, 1)";

/** The marker: positioned inline, its look from custom properties a page can set, with a forced-colours default. */
export function landingMarker(from: HTMLElement | null, kind: "card" | "lane"): HTMLElement {
  const marker = from ? (from.cloneNode(true) as HTMLElement) : document.createElement("div");
  marker.removeAttribute("id");
  marker.querySelectorAll("[id]").forEach((element) => element.removeAttribute("id"));
  for (const name of ["tabindex", "data-dragging", "data-drop-before", "data-lane-dragging", "data-lane-drop-target"])
    marker.removeAttribute(name);
  marker.setAttribute("data-landing-marker", kind);
  marker.setAttribute("aria-hidden", "true");
  marker.inert = true;
  marker.style.position = "fixed";
  marker.style.boxSizing = "border-box";
  marker.style.margin = "0";
  marker.style.pointerEvents = "none";
  const forced = matchMedia("(forced-colors: active)").matches;
  marker.style.setProperty("opacity", `var(--sb-landing-opacity, ${forced ? 1 : 0.5})`);
  marker.style.setProperty("outline", `var(--sb-landing-outline, 2px dashed ${forced ? "Highlight" : "currentColor"})`);
  marker.style.setProperty("outline-offset", "-2px");
  return marker;
}

/** The part of `element` its scrolling ancestors leave visible. */
function visibleArea(element: HTMLElement): Rect {
  let { left, top, right, bottom } = element.getBoundingClientRect();
  for (let current = element.parentElement; current && current !== document.documentElement;) {
    const style = getComputedStyle(current);
    if (style.overflowX !== "visible" || style.overflowY !== "visible") {
      const rect = current.getBoundingClientRect();
      left = Math.max(left, rect.left);
      top = Math.max(top, rect.top);
      right = Math.min(right, rect.right);
      bottom = Math.min(bottom, rect.bottom);
    }
    current = current.parentElement;
  }
  return { left, top, width: Math.max(0, right - left), height: Math.max(0, bottom - top) };
}

/** Pending landings: each marker lives in <body>, out of reach of the morphs that patch the board. */
export function installLandings(options: { host: HTMLElement; onEnd: (detail: KanbanLandingEndDetail) => void }) {
  const pending = new Map<string, Pending>();
  let frame: number | null = null;

  const setGap = (entry: Pending, want: LandingPlace["gap"]): void => {
    const current = entry.gap;
    if (current && want && current.element === want.element && current.side === want.side && current.size === want.size)
      return;
    current?.animation.cancel();
    entry.gap = null;
    if (!want) return;
    const property = want.side === "top" ? "marginTop" : "marginBottom";
    const base = parseFloat(getComputedStyle(want.element)[property]) || 0;
    const animation = want.element.animate([{ [property]: `${base}px` }, { [property]: `${base + want.size}px` }], {
      duration: current || reducedMotion() ? 0 : 150,
      easing,
      fill: "forwards",
    });
    entry.gap = { ...want, animation };
  };
  const closeGap = (entry: Pending, smooth: boolean): void => {
    const animation = entry.gap?.animation;
    entry.gap = null;
    if (!animation) return;
    if (!smooth || reducedMotion()) {
      animation.cancel();
      return;
    }
    animation.reverse();
    animation.finished.then(() => animation.cancel()).catch(() => {});
  };
  const end = (entry: Pending, reason: KanbanLandingEndDetail["reason"], notify = true): void => {
    if (pending.get(entry.spec.key) !== entry) return;
    pending.delete(entry.spec.key);
    clearTimeout(entry.timer);
    entry.spec.marker.remove();
    closeGap(entry, reason !== "arrived");
    if (notify) options.onEnd({ ...entry.spec.detail, reason });
  };
  const place = (entry: Pending, where: LandingPlace): void => {
    const { marker } = entry.spec;
    // A morph of the whole page removes it with everything else the server didn't render: put it back.
    if (!marker.isConnected) document.body.append(marker);
    const { rect } = where;
    marker.style.left = `${rect.left}px`;
    marker.style.top = `${rect.top}px`;
    marker.style.width = `${rect.width}px`;
    marker.style.height = `${rect.height}px`;
    const view = visibleArea(where.clip);
    const clip = {
      top: Math.max(0, view.top - rect.top),
      right: Math.max(0, rect.left + rect.width - (view.left + view.width)),
      bottom: Math.max(0, rect.top + rect.height - (view.top + view.height)),
      left: Math.max(0, view.left - rect.left),
    };
    marker.style.clipPath =
      clip.top || clip.right || clip.bottom || clip.left
        ? `inset(${clip.top}px ${clip.right}px ${clip.bottom}px ${clip.left}px)`
        : "";
    marker.style.visibility = "";
    if (entry.placed) return;
    entry.placed = true;
    const dx = entry.spec.origin.left - rect.left;
    const dy = entry.spec.origin.top - rect.top;
    if (!reducedMotion() && (Math.abs(dx) >= 1 || Math.abs(dy) >= 1))
      marker.animate([{ transform: `translate(${dx}px, ${dy}px)` }, { transform: "none" }], { duration: 150, easing });
  };
  // Each frame: a move that arrived ends its landing, and the others follow their slot through morphs, scrolling and
  // resizing. Arming the move's animation every frame lets it start from the marker when the server's answer lands.
  const tick = (): void => {
    frame = null;
    const armed = new Set<object>();
    for (const entry of [...pending.values()]) {
      if (entry.spec.arrived()) {
        end(entry, "arrived");
        continue;
      }
      const where = entry.spec.place();
      setGap(entry, where?.gap ?? null);
      if (!where) {
        entry.spec.marker.style.visibility = "hidden";
        continue;
      }
      place(entry, where);
      if (armed.has(entry.spec.group)) continue;
      armed.add(entry.spec.group);
      entry.spec.prepare(where.rect);
    }
    if (pending.size) frame = requestAnimationFrame(tick);
  };

  return {
    add(spec: LandingSpec, timeout: number): void {
      const previous = pending.get(spec.key);
      if (previous) end(previous, "released");
      spec.marker.style.visibility = "hidden";
      document.body.append(spec.marker);
      const entry: Pending = {
        spec,
        placed: false,
        gap: null,
        timer: setTimeout(() => end(entry, "timeout"), timeout),
      };
      pending.set(spec.key, entry);
      if (frame === null) frame = requestAnimationFrame(tick);
    },
    /** Ends landings early: the one of a card ({ cardId }), of a lane ({ col }), or all; true when one was pending. */
    release(match?: { cardId?: string; col?: number }): boolean {
      const ended = [...pending.values()].filter(({ spec: { detail } }) =>
        match?.cardId !== undefined
          ? detail.cardId === match.cardId
          : match?.col !== undefined
            ? detail.cardId === undefined && detail.col === match.col
            : true,
      );
      ended.forEach((entry) => end(entry, "released"));
      return ended.length > 0;
    },
    dispose(): void {
      if (frame !== null) cancelAnimationFrame(frame);
      frame = null;
      [...pending.values()].forEach((entry) => end(entry, "released", false));
    },
  };
}

type Geometry = {
  lanes: () => HTMLElement[];
  cards: () => HTMLElement[];
  cardsIn: (lane: HTMLElement) => HTMLElement[];
};

const cardId = (card: HTMLElement | undefined) => card?.dataset.kanbanCard ?? "";

/** The landing of a card dropped into lane `col` before card `before` ("" for the end); null for a drop in place. */
export function cardLanding(
  geometry: Geometry,
  card: HTMLElement,
  target: { col: number; before: string },
  origin: { left: number; top: number },
  prepare: (id: string, rect: Rect) => void,
  group: object,
): LandingSpec | null {
  const id = cardId(card);
  const col = String(target.col);
  const lane = card.closest<HTMLElement>(kanbanContract.selectors.lane);
  const siblings = lane ? geometry.cardsIn(lane) : [];
  const from = { col: lane?.dataset.col, next: cardId(siblings[siblings.indexOf(card) + 1]) };
  if (from.col === col && from.next === target.before) return null;
  const height = card.getBoundingClientRect().height;
  return {
    key: `card:${id}`,
    detail: { cardId: id, col: target.col },
    marker: landingMarker(card, "card"),
    origin,
    place: () => {
      const lane = geometry.lanes().find((candidate) => candidate.dataset.col === col);
      if (!lane) return null;
      const list = lane.querySelector<HTMLElement>("[data-kanban-lane-cards]") ?? lane;
      const inLane = geometry.cardsIn(lane);
      const gap = parseFloat(getComputedStyle(list).rowGap) || 0;
      const before = target.before ? inLane.find((other) => cardId(other) === target.before) : undefined;
      if (before) {
        const previous = inLane[inLane.indexOf(before) - 1];
        const rect = before.getBoundingClientRect();
        const top = previous ? previous.getBoundingClientRect().bottom + gap : contentTop(list);
        return {
          rect: { left: rect.left, top, width: rect.width, height },
          gap: { element: before, side: "top", size: height + gap },
          clip: list,
        };
      }
      const last = inLane.filter((other) => cardId(other) !== id).at(-1);
      if (last) {
        const rect = last.getBoundingClientRect();
        return {
          rect: { left: rect.left, top: rect.bottom + gap, width: rect.width, height },
          gap: { element: last, side: "bottom", size: height + gap },
          clip: list,
        };
      }
      const box = contentBox(list);
      return { rect: { ...box, height }, gap: null, clip: list };
    },
    arrived: () => {
      const now = geometry.cards().find((other) => cardId(other) === id);
      const lane = now?.closest<HTMLElement>(kanbanContract.selectors.lane);
      if (!now || lane?.dataset.col !== col) return false;
      const inLane = geometry.cardsIn(lane);
      return from.col !== col || cardId(inLane[inLane.indexOf(now) + 1]) !== from.next;
    },
    prepare: (rect) => prepare(id, rect),
    group,
  };
}

/** The landing of a lane moved into `target`'s place, before lane `before` ("" for the end). */
export function laneLanding(
  geometry: Geometry,
  lane: HTMLElement,
  target: HTMLElement,
  before: string,
  origin: { left: number; top: number },
  prepare: (col: string, rect: Rect) => void,
  group: object,
): LandingSpec {
  const col = lane.dataset.col ?? "";
  const targetCol = target.dataset.col ?? "";
  return {
    key: `lane:${col}`,
    detail: { col: Number(col) },
    marker: landingMarker(null, "lane"),
    origin,
    place: () => {
      const slot = geometry.lanes().find((candidate) => candidate.dataset.col === targetCol);
      if (!slot?.parentElement) return null;
      const { left, top, width, height } = slot.getBoundingClientRect();
      return { rect: { left, top, width, height }, gap: null, clip: slot.parentElement };
    },
    arrived: () => {
      const all = geometry.lanes();
      const index = all.findIndex((candidate) => candidate.dataset.col === col);
      return index >= 0 && (all[index + 1]?.dataset.col ?? "") === before;
    },
    prepare: (rect) => prepare(col, rect),
    group,
  };
}

function contentBox(element: HTMLElement): Rect {
  const rect = element.getBoundingClientRect();
  const style = getComputedStyle(element);
  const left = rect.left + (parseFloat(style.borderLeftWidth) || 0) + (parseFloat(style.paddingLeft) || 0);
  const right = rect.right - (parseFloat(style.borderRightWidth) || 0) - (parseFloat(style.paddingRight) || 0);
  return { left, top: contentTop(element), width: Math.max(0, right - left), height: 0 };
}

function contentTop(element: HTMLElement): number {
  const style = getComputedStyle(element);
  return (
    element.getBoundingClientRect().top + (parseFloat(style.borderTopWidth) || 0) + (parseFloat(style.paddingTop) || 0)
  );
}
