// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export const contextMenuContract = {
  tag: "sb-context-menu",
  selectors: { trigger: "[data-menu-for]", item: '[role="menuitem"], [role="menuitemradio"]' },
  events: { action: "sb-menu-action", scope: "sb-menu-scope" },
} as const;

export type MenuActionDetail = { action: string; contextId: string };
export type MenuScopeDetail = { root: HTMLElement; active: boolean };
export type ContextMenuHost = HTMLElement & {
  openFor(trigger: HTMLElement, point?: { x: number; y: number }, context?: Record<string, string>): void;
  closeMenu(refocus?: boolean): void;
  isOpen(): boolean;
};
