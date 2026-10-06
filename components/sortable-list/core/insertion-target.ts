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
/** Resolve a one-dimensional insert point; an empty `before` appends. */
export function insertionBefore(items: readonly { id: string; top: number; bottom: number }[], y: number): string {
  for (const item of items) {
    if (y < (item.top + item.bottom) / 2) return item.id;
  }
  return "";
}
