// Generated from visual-outlets.ts by `go tool task ts`: edit the TypeScript, not this file.
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
/** Use a server-rendered fragment for the floating pointer preview, or clone the item by default. */
export function dragPreviewFor(item) {
    const template = item.querySelector(":scope > template[data-sb-preview]");
    if (!template)
        return item.cloneNode(true);
    const preview = document.createElement("div");
    preview.className = template.className;
    preview.append(template.content.cloneNode(true));
    return preview;
}
/** Optional decoration outlet; the surface still owns the target and its geometry. */
export function installTargetIndicator(host) {
    let indicator = null;
    const clear = () => {
        indicator?.remove();
        indicator = null;
    };
    return {
        show(target, kind) {
            clear();
            if (!target)
                return;
            const template = host.querySelector(`:scope > template[data-sb-target="${kind}"]`) ??
                host.querySelector(':scope > template[data-sb-target=""]');
            if (!template)
                return;
            indicator = document.createElement("span");
            indicator.setAttribute("data-sb-target-indicator", kind);
            indicator.setAttribute("aria-hidden", "true");
            indicator.inert = true;
            indicator.style.position = "absolute";
            indicator.style.pointerEvents = "none";
            indicator.append(template.content.cloneNode(true));
            target.append(indicator);
        },
        clear,
    };
}
