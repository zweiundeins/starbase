// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Resolve a one-dimensional insert point; an empty `before` appends. */
export function insertionBefore(items: readonly { id: string; top: number; bottom: number }[], y: number): string {
  for (const item of items) {
    if (y < (item.top + item.bottom) / 2) return item.id;
  }
  return "";
}
