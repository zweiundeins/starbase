// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Registered hosts define gesture ownership without coupling core to surface tags or DOM attributes. */
const hosts = new WeakSet<HTMLElement>();

export function markRocketHost(host: HTMLElement): () => void {
  hosts.add(host);
  return () => hosts.delete(host);
}

export function ownsRocketElement(host: HTMLElement, element: Element): boolean {
  for (let current: Element | null = element; current; current = current.parentElement) {
    if (hosts.has(current as HTMLElement)) return current === host;
  }
  return false;
}
