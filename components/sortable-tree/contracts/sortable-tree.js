// Generated from sortable-tree.ts by `go tool task ts`: edit the TypeScript, not this file.
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
};
