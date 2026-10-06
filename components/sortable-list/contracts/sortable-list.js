// Generated from sortable-list.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export const sortableListContract = {
    tag: "sb-sortable-list",
    selectors: { item: "[data-sortable-item]" },
    events: { move: "sb-sortable-move" },
};
