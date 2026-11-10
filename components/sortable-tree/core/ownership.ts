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

/**
 * Registered hosts define gesture ownership without coupling core to surface tags or DOM attributes.
 * The registry lives on globalThis, so surfaces bundled with their own copy of core still see each other when nested.
 */
const registry = globalThis as typeof globalThis & { [key: symbol]: WeakSet<HTMLElement> | undefined };
const hosts = (registry[Symbol.for("sb-rockets.hosts")] ??= new WeakSet<HTMLElement>());

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
