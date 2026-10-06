// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export const bentoContract = {
  tag: "sb-bento-workspace",
  selectors: {
    grid: "[data-bento-grid]",
    item: "[data-bento-item]",
    resize: "[data-bento-resize]",
  },
  events: { move: "sb-bento-move", resize: "sb-bento-resize" },
} as const;

export type BentoMoveDetail = {
  itemId: string;
  fromGrid: string;
  toGrid: string;
  updates: BentoPosition[];
};

export type BentoPosition = { itemId: string; grid: string; col: number; row: number; width: number; height: number };

export type BentoResizeDetail = { itemId: string; grid: string; updates: BentoPosition[] };
