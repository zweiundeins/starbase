// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export const sortableListContract = {
  tag: "sb-sortable-list",
  selectors: { item: "[data-sortable-item]" },
  events: { move: "sb-sortable-move" },
} as const;

export type SortableMoveDetail = { itemId: string; before: string };
