// Generated from context-menu.ts by `go tool task ts`: edit the TypeScript, not this file.
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
export const contextMenuContract = {
    tag: "sb-context-menu",
    selectors: { trigger: "[data-menu-for]", item: '[role="menuitem"], [role="menuitemradio"]' },
    events: { action: "sb-menu-action", scope: "sb-menu-scope" },
};
