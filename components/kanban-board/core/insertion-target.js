// Generated from insertion-target.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Resolve a one-dimensional insert point; an empty `before` appends. */
export function insertionBefore(items, y) {
    for (const item of items) {
        if (y < (item.top + item.bottom) / 2)
            return item.id;
    }
    return "";
}
