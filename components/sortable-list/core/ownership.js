// Generated from ownership.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Registered hosts define gesture ownership without coupling core to surface tags or DOM attributes. */
const hosts = new WeakSet();
export function markRocketHost(host) {
    hosts.add(host);
    return () => hosts.delete(host);
}
export function ownsRocketElement(host, element) {
    for (let current = element; current; current = current.parentElement) {
        if (hosts.has(current))
            return current === host;
    }
    return false;
}
