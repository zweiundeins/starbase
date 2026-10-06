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

import { rocket } from "datastar";
import {
  contextMenuContract,
  type MenuActionDetail,
  type ContextMenuHost,
  type MenuScopeDetail,
} from "./contracts/context-menu.ts";
import { bindTemplate } from "./core/template-bind.ts";
import { showPositionedPopover } from "./core/popover.ts";
import { cancelKeys, keyboardBindings, platformFocusKeys } from "./core/keyboard.ts";

type Level = { trigger: HTMLElement; menu: HTMLElement; unposition: () => void };
// Triggers that open the menu on click, the only ones that take aria-expanded.
const menuButton = "button, a";
type MenuHost = ContextMenuHost;

/** Install menu mechanics on a server-owned light-DOM host (including fetched fragments). */
export function installContextMenu(
  host: MenuHost,
  cleanup: (fn: () => void) => void,
  options: {
    inlineContent?: boolean;
    captureTriggers?: boolean;
    mobileQuery?: string;
    mobileSheet?: boolean;
    focusFirst?: boolean;
  } = {},
): void {
  const {
    inlineContent = false,
    captureTriggers = true,
    mobileQuery = "(max-width: 650px)",
    focusFirst = true,
  } = options;
  setupMenu(host, cleanup, {
    inlineContent,
    captureTriggers,
    mobileQuery,
    mobileSheet: options.mobileSheet ?? false,
    focusFirst,
  });
}

rocket("sb-context-menu", {
  mode: "light",
  manifest: {
    events: [
      {
        name: contextMenuContract.events.action,
        kind: "custom-event",
        bubbles: true,
        composed: true,
        description: "An item with data-action was chosen. detail: { action, contextId }.",
      },
      {
        name: contextMenuContract.events.scope,
        kind: "custom-event",
        bubbles: true,
        composed: false,
        description: "The menu opened (active: true) or closed. detail: { root, active }.",
      },
    ],
  },
  setup({ host, cleanup }: { host: HTMLElement; cleanup: (fn: () => void) => void }) {
    // The menu methods are installed below; Rocket hands setup a plain element.
    installContextMenu(host as MenuHost, cleanup);
  },
});

function setupMenu(
  host: MenuHost,
  cleanup: (fn: () => void) => void,
  options: {
    inlineContent: boolean;
    captureTriggers: boolean;
    mobileQuery: string;
    mobileSheet: boolean;
    focusFirst: boolean;
  },
) {
  let trigger: HTMLElement | null = null;
  let content: HTMLElement | null = null;
  let clonedContent = false;
  let contextId = "";
  let unpositionRoot: (() => void) | null = null;
  let pendingContextOpen = 0;
  let pressedTrigger: HTMLElement | null = null;
  const levels: Level[] = [];
  const keyboard = keyboardBindings(host, { ...platformFocusKeys(navigator.platform), ...cancelKeys });
  const mobile = matchMedia(options.mobileQuery);
  host.popover = "auto";
  host.setAttribute("role", "menu");

  const itemsIn = (menu: HTMLElement): HTMLElement[] =>
    [...menu.querySelectorAll<HTMLElement>(contextMenuContract.selectors.item)].filter(
      (item) => item.closest('[role="menu"]') === menu && !item.matches(":disabled, [aria-disabled='true']"),
    );
  const focusIn = (menu: HTMLElement) => ((options.focusFirst && itemsIn(menu)[0]) || menu).focus();
  const closeTo = (depth: number, refocus = false) => {
    while (levels.length > depth) {
      const level = levels.pop()!;
      if (level.menu.matches(":popover-open")) level.menu.hidePopover();
      level.unposition();
      level.trigger.setAttribute("aria-expanded", "false");
      if (refocus && levels.length === depth) level.trigger.focus();
    }
  };
  const close = (refocus = false) => {
    if (!content) return;
    // A morph can strip popover from the open host, so forget the menu first and hide only what is shown.
    const closing = content;
    const cloned = clonedContent;
    content = null;
    clonedContent = false;
    closeTo(0);
    if (host.matches(":popover-open")) host.hidePopover();
    unpositionRoot?.();
    unpositionRoot = null;
    if (cloned) closing.remove();
    host.dispatchEvent(
      new CustomEvent<MenuScopeDetail>(contextMenuContract.events.scope, {
        bubbles: true,
        detail: { root: host, active: false },
      }),
    );
    const previous = trigger;
    trigger = null;
    if (previous) {
      if (previous.matches(menuButton)) previous.setAttribute("aria-expanded", "false");
      if (refocus && previous.isConnected) previous.focus();
    }
  };
  const openSubmenu = (button: HTMLElement, focus = false) => {
    const id = button.dataset.submenu;
    const submenu = id && content?.querySelector<HTMLElement>(`[id="${CSS.escape(id)}"][role="menu"]`);
    if (!submenu) return;
    const parent = button.closest<HTMLElement>('[role="menu"]');
    const depth = parent === host ? 0 : levels.findIndex((level) => level.menu === parent) + 1;
    if (depth < 0 || (parent !== host && depth === 0)) return;
    if (levels[depth]?.menu === submenu) {
      if (!submenu.matches(":popover-open")) {
        levels[depth]!.unposition();
        levels[depth]!.unposition =
          options.mobileSheet && mobile.matches ? showSheet(submenu) : showPositionedPopover(submenu, button, "beside");
      }
      if (focus) focusIn(submenu);
      return;
    }
    closeTo(depth);
    submenu.popover = "auto";
    submenu.tabIndex = -1;
    submenu.toggleAttribute("data-mobile-sheet", mobile.matches);
    const unposition =
      options.mobileSheet && mobile.matches ? showSheet(submenu) : showPositionedPopover(submenu, button, "beside");
    levels.push({ trigger: button, menu: submenu, unposition });
    button.setAttribute("aria-expanded", "true");
    if (focus) focusIn(submenu);
  };
  const openFor = (source: HTMLElement, point?: { x: number; y: number }, context: Record<string, string> = {}) => {
    const template = host.querySelector<HTMLTemplateElement>(":scope > template[data-sb-menu]");
    const live = host.querySelector<HTMLElement>(":scope > [data-sb-menu-content]");
    if (!template && !live && !options.inlineContent) return;
    close();
    trigger = source;
    contextId = source.closest<HTMLElement>("[data-context-id]")?.dataset.contextId ?? source.dataset.contextId ?? "";
    clonedContent = !!template;
    if (template) {
      content = document.createElement("div");
      content.setAttribute("data-menu-content", "");
      content.append(bindTemplate(template, { ...triggerParams(source), ...context, contextId }));
      disableItems(content, source);
      host.append(content);
    } else content = live ?? host;
    // Submenus stay hidden until they open.
    for (const submenu of content.querySelectorAll<HTMLElement>('[role="menu"]')) submenu.popover = "auto";
    // A server morph may replace setup-only attributes with the original HTML.
    host.popover = "auto";
    host.setAttribute("role", "menu");
    host.tabIndex = -1;
    host.toggleAttribute("data-mobile-sheet", mobile.matches);
    if (source.matches(menuButton)) source.setAttribute("aria-expanded", "true");
    unpositionRoot =
      options.mobileSheet && mobile.matches
        ? showSheet(host)
        : showPositionedPopover(host, point ?? source, point ? "point" : "below");
    focusIn(host);
    host.dispatchEvent(
      new CustomEvent<MenuScopeDetail>(contextMenuContract.events.scope, {
        bubbles: true,
        detail: { root: host, active: true },
      }),
    );
  };
  host.openFor = openFor;
  host.closeMenu = close;
  host.isOpen = () => !!content && host.matches(":popover-open");
  const ownsTrigger = (element: Element): HTMLElement | null => {
    const found = element.closest<HTMLElement>(contextMenuContract.selectors.trigger);
    return found?.dataset.menuFor === host.id ? found : null;
  };
  const onContextMenu = (event: MouseEvent) => {
    const source = event.target instanceof Element && ownsTrigger(event.target);
    if (!source) return;
    event.preventDefault();
    const point = { x: event.clientX, y: event.clientY };
    cancelAnimationFrame(pendingContextOpen);
    pendingContextOpen = requestAnimationFrame(() => {
      pendingContextOpen = 0;
      if (source.isConnected) openFor(source, point);
    });
  };
  // Light dismiss hides the menu before the click on its trigger arrives, so note at pointerdown that it was open.
  const onPointerDown = (event: PointerEvent) => {
    pressedTrigger = trigger && event.target instanceof Node && trigger.contains(event.target) ? trigger : null;
  };
  const onTriggerClick = (event: MouseEvent) => {
    const source = event.target instanceof Element && ownsTrigger(event.target);
    const again = source === pressedTrigger && event.detail > 0;
    pressedTrigger = null;
    if (source && source.matches(menuButton)) {
      event.preventDefault();
      if (again) close(true);
      else openFor(source);
    }
  };
  const onClick = (event: MouseEvent) => {
    const item = (event.target as Element).closest<HTMLElement>(contextMenuContract.selectors.item);
    if ((event.target as Element).closest("[data-submenu-back]")) {
      closeTo(levels.length - 1, true);
      return;
    }
    if (!item || !content?.contains(item) || item.matches(":disabled, [aria-disabled='true']")) return;
    if (item.dataset.submenu) return openSubmenu(item, true);
    if (item.dataset.action) {
      host.dispatchEvent(
        new CustomEvent<MenuActionDetail>(contextMenuContract.events.action, {
          bubbles: true,
          composed: true,
          detail: { action: item.dataset.action, contextId },
        }),
      );
    }
    close(true);
  };
  const onHover = (event: MouseEvent) => {
    if (mobile.matches) return;
    const item = (event.target as Element).closest<HTMLElement>(contextMenuContract.selectors.item);
    if (!item || !content?.contains(item)) return;
    if (item.dataset.submenu) openSubmenu(item);
    else {
      const parent = item.closest<HTMLElement>('[role="menu"]');
      const depth = parent === host ? 0 : levels.findIndex((level) => level.menu === parent) + 1;
      if (depth >= 0) closeTo(depth);
    }
  };
  const onKey = (event: KeyboardEvent) => {
    if (
      event.defaultPrevented ||
      event.isComposing ||
      (event.target as Element).closest("input, textarea, [contenteditable]")
    )
      return;
    if (event.key === "Tab") {
      // Hand the focus back to the trigger, and the browser moves on from there.
      close(true);
      return;
    }
    const active = document.activeElement as HTMLElement;
    const scope = active.closest<HTMLElement>('[role="menu"]') ?? host;
    if (keyboard.matches("cancel", event)) {
      event.preventDefault();
      event.stopPropagation();
      if (levels.length) closeTo(levels.length - 1, true);
      else close(true);
      return;
    }
    if (
      (keyboard.matches("focusLeft", event) ||
        (event.key === "Backspace" && !event.ctrlKey && !event.metaKey && !event.altKey)) &&
      levels.length
    ) {
      event.preventDefault();
      closeTo(levels.length - 1, true);
      return;
    }
    if (keyboard.matches("focusRight", event) && active.dataset.submenu) {
      event.preventDefault();
      openSubmenu(active, true);
      return;
    }
    const items = itemsIn(scope);
    if (!items.length) return;
    const direction = keyboard.matches("focusNext", event) ? 1 : keyboard.matches("focusPrevious", event) ? -1 : 0;
    if (!direction && !keyboard.matches("focusFirst", event) && !keyboard.matches("focusLast", event)) return;
    event.preventDefault();
    const index = items.indexOf(active);
    const next = keyboard.matches("focusFirst", event)
      ? 0
      : keyboard.matches("focusLast", event)
        ? items.length - 1
        : index < 0
          ? direction > 0
            ? 0
            : items.length - 1
          : (index + direction + items.length) % items.length;
    if (scope === host) closeTo(0);
    else closeTo(levels.findIndex((level) => level.menu === scope) + 1);
    items[next]?.focus();
  };
  const onToggle = (event: Event) => {
    if (event.target === host && (event as ToggleEvent).newState === "closed") close();
  };
  if (options.captureTriggers) {
    document.addEventListener("contextmenu", onContextMenu);
    document.addEventListener("click", onTriggerClick);
    document.addEventListener("pointerdown", onPointerDown, true);
  }
  host.addEventListener("click", onClick);
  host.addEventListener("mouseover", onHover);
  host.addEventListener("keydown", onKey);
  host.addEventListener("toggle", onToggle);
  cleanup(() => {
    cancelAnimationFrame(pendingContextOpen);
    close();
    delete (host as Partial<ContextMenuHost>).openFor;
    delete (host as Partial<ContextMenuHost>).closeMenu;
    delete (host as Partial<ContextMenuHost>).isOpen;
    if (options.captureTriggers) {
      document.removeEventListener("contextmenu", onContextMenu);
      document.removeEventListener("click", onTriggerClick);
      document.removeEventListener("pointerdown", onPointerDown, true);
    }
    host.removeEventListener("click", onClick);
    host.removeEventListener("mouseover", onHover);
    host.removeEventListener("keydown", onKey);
    host.removeEventListener("toggle", onToggle);
  });
}

/** The data-menu-param-<name> values of a trigger and of the element that carries its context id. */
function triggerParams(trigger: HTMLElement): Record<string, string> {
  const params: Record<string, string> = {};
  for (const element of [trigger.closest<HTMLElement>("[data-context-id]"), trigger])
    for (const [key, value] of Object.entries(element?.dataset ?? {}))
      if (/^menuParam[A-Z]/.test(key) && value !== undefined) params[key[9]!.toLowerCase() + key.slice(10)] = value;
  return params;
}

/** Turn off the copied items whose data-action the trigger, or its context element, lists in data-menu-disabled. */
function disableItems(content: HTMLElement, trigger: HTMLElement): void {
  const actions = [trigger.closest<HTMLElement>("[data-context-id]"), trigger].flatMap(
    (element) => element?.dataset.menuDisabled?.split(" ").filter(Boolean) ?? [],
  );
  for (const item of content.querySelectorAll<HTMLElement>("[data-action]"))
    if (actions.includes(item.dataset.action!)) item.setAttribute("aria-disabled", "true");
}

function showSheet(menu: HTMLElement): () => void {
  menu.showPopover();
  return () => {};
}
