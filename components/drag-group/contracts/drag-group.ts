// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export const dragGroupContract = {
  tag: "sb-drag-group",
  selectors: {
    list: "[data-drop-list]",
    item: "[data-drag-item]",
  },
  events: { move: "sb-drag-group-move" },
} as const;

/** Item IDs are unique within a group; list IDs identify server-rendered destinations. */
export type DragGroupMoveDetail = { itemId: string; fromList: string; toList: string; before: string };
