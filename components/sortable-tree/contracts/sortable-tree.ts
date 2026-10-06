// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export const sortableTreeContract = {
  tag: "sb-sortable-tree",
  selectors: {
    node: "[data-tree-node]",
    row: "[data-tree-row]",
    children: "[data-tree-children]",
  },
  events: { move: "sb-tree-move" },
} as const;

/** Empty parent IDs address the root list; before is empty for the end of a list. */
export type TreeMoveDetail = { itemId: string; fromParent: string; toParent: string; before: string };
