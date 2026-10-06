// Generated from context-menu.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import { rocket } from "datastar";
import { contextMenuContract, } from "./contracts/context-menu.js";
import { bindTemplate } from "./core/template-bind.js";
import { showPositionedPopover } from "./core/popover.js";
import { cancelKeys, keyboardBindings, platformFocusKeys } from "./core/keyboard.js";
/** Install menu mechanics on a server-owned light-DOM host (including fetched fragments). */
export function installContextMenu(host, cleanup, options = {}) {
    const { inlineContent = false, captureTriggers = true, mobileQuery = "(max-width: 650px)", focusFirst = false, } = options;
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
    setup({ host, cleanup }) {
        installContextMenu(host, cleanup);
    },
});
function setupMenu(host, cleanup, options) {
    let trigger = null;
    let content = null;
    let clonedContent = false;
    let contextId = "";
    let unpositionRoot = null;
    let pendingContextOpen = 0;
    const levels = [];
    const keyboard = keyboardBindings(host, { ...platformFocusKeys(navigator.platform), ...cancelKeys });
    const mobile = matchMedia(options.mobileQuery);
    host.popover = "auto";
    host.setAttribute("role", "menu");
    const itemsIn = (menu) => [...menu.querySelectorAll(contextMenuContract.selectors.item)].filter((item) => item.closest('[role="menu"]') === menu && !item.matches(":disabled, [aria-disabled='true']"));
    const closeTo = (depth, refocus = false) => {
        while (levels.length > depth) {
            const level = levels.pop();
            level.menu.hidePopover();
            level.unposition();
            level.trigger.setAttribute("aria-expanded", "false");
            if (refocus && levels.length === depth)
                level.trigger.focus();
        }
    };
    const close = (refocus = false) => {
        if (!content)
            return;
        closeTo(0);
        host.hidePopover();
        unpositionRoot?.();
        unpositionRoot = null;
        if (clonedContent)
            content.remove();
        content = null;
        clonedContent = false;
        host.dispatchEvent(new CustomEvent(contextMenuContract.events.scope, {
            bubbles: true,
            detail: { root: host, active: false },
        }));
        const previous = trigger;
        trigger = null;
        if (previous) {
            previous.setAttribute("aria-expanded", "false");
            if (refocus && previous.isConnected)
                previous.focus();
        }
    };
    const openSubmenu = (button, focus = false) => {
        const id = button.dataset.submenu;
        const submenu = id && content?.querySelector(`[id="${CSS.escape(id)}"][role="menu"]`);
        if (!submenu)
            return;
        const parent = button.closest('[role="menu"]');
        const depth = parent === host ? 0 : levels.findIndex((level) => level.menu === parent) + 1;
        if (depth < 0 || (parent !== host && depth === 0))
            return;
        if (levels[depth]?.menu === submenu) {
            if (!submenu.matches(":popover-open")) {
                levels[depth].unposition();
                levels[depth].unposition =
                    options.mobileSheet && mobile.matches ? showSheet(submenu) : showPositionedPopover(submenu, button, "beside");
            }
            if (focus)
                (options.focusFirst ? itemsIn(submenu)[0] : submenu)?.focus();
            return;
        }
        closeTo(depth);
        submenu.popover = "auto";
        submenu.tabIndex = -1;
        submenu.toggleAttribute("data-mobile-sheet", mobile.matches);
        const unposition = options.mobileSheet && mobile.matches ? showSheet(submenu) : showPositionedPopover(submenu, button, "beside");
        levels.push({ trigger: button, menu: submenu, unposition });
        button.setAttribute("aria-expanded", "true");
        if (focus)
            (options.focusFirst ? itemsIn(submenu)[0] : submenu)?.focus();
    };
    const openFor = (source, point, context = {}) => {
        const template = host.querySelector(":scope > template[data-sb-menu]");
        const live = host.querySelector(":scope > [data-sb-menu-content]");
        if (!template && !live && !options.inlineContent)
            return;
        close();
        trigger = source;
        contextId = source.closest("[data-context-id]")?.dataset.contextId ?? source.dataset.contextId ?? "";
        clonedContent = !!template;
        if (template) {
            content = document.createElement("div");
            content.setAttribute("data-menu-content", "");
            content.append(bindTemplate(template, { ...context, contextId }));
            host.append(content);
        }
        else
            content = live ?? host;
        // A server morph may replace setup-only attributes with the original HTML.
        host.popover = "auto";
        host.setAttribute("role", "menu");
        host.tabIndex = -1;
        host.toggleAttribute("data-mobile-sheet", mobile.matches);
        source.setAttribute("aria-expanded", "true");
        unpositionRoot =
            options.mobileSheet && mobile.matches
                ? showSheet(host)
                : showPositionedPopover(host, point ?? source, point ? "point" : "below");
        (options.focusFirst ? itemsIn(host)[0] : host)?.focus();
        host.dispatchEvent(new CustomEvent(contextMenuContract.events.scope, {
            bubbles: true,
            detail: { root: host, active: true },
        }));
    };
    host.openFor = openFor;
    host.closeMenu = close;
    host.isOpen = () => !!content;
    const ownsTrigger = (element) => {
        const found = element.closest(contextMenuContract.selectors.trigger);
        return found?.dataset.menuFor === host.id ? found : null;
    };
    const onContextMenu = (event) => {
        const source = event.target instanceof Element && ownsTrigger(event.target);
        if (!source)
            return;
        event.preventDefault();
        const point = { x: event.clientX, y: event.clientY };
        cancelAnimationFrame(pendingContextOpen);
        pendingContextOpen = requestAnimationFrame(() => {
            pendingContextOpen = 0;
            if (source.isConnected)
                openFor(source, point);
        });
    };
    const onTriggerClick = (event) => {
        const source = event.target instanceof Element && ownsTrigger(event.target);
        if (source && source.matches("button, a")) {
            event.preventDefault();
            openFor(source);
        }
    };
    const onClick = (event) => {
        const item = event.target.closest(contextMenuContract.selectors.item);
        if (event.target.closest("[data-submenu-back]")) {
            closeTo(levels.length - 1, true);
            return;
        }
        if (!item || !content?.contains(item) || item.matches(":disabled, [aria-disabled='true']"))
            return;
        if (item.dataset.submenu)
            return openSubmenu(item, true);
        if (item.dataset.action) {
            host.dispatchEvent(new CustomEvent(contextMenuContract.events.action, {
                bubbles: true,
                composed: true,
                detail: { action: item.dataset.action, contextId },
            }));
        }
        close(true);
    };
    const onHover = (event) => {
        if (mobile.matches)
            return;
        const item = event.target.closest(contextMenuContract.selectors.item);
        if (!item || !content?.contains(item))
            return;
        if (item.dataset.submenu)
            openSubmenu(item);
        else {
            const parent = item.closest('[role="menu"]');
            const depth = parent === host ? 0 : levels.findIndex((level) => level.menu === parent) + 1;
            if (depth >= 0)
                closeTo(depth);
        }
    };
    const onKey = (event) => {
        if (event.defaultPrevented ||
            event.isComposing ||
            event.target.closest("input, textarea, [contenteditable]"))
            return;
        const active = document.activeElement;
        const scope = active.closest('[role="menu"]') ?? host;
        if (keyboard.matches("cancel", event)) {
            event.preventDefault();
            event.stopPropagation();
            if (levels.length)
                closeTo(levels.length - 1, true);
            else
                close(true);
            return;
        }
        if ((keyboard.matches("focusLeft", event) ||
            (event.key === "Backspace" && !event.ctrlKey && !event.metaKey && !event.altKey)) &&
            levels.length) {
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
        if (!items.length)
            return;
        if ((event.key === "Enter" || event.key === " ") && active === scope) {
            event.preventDefault();
            items[0]?.click();
            return;
        }
        const direction = keyboard.matches("focusNext", event) ? 1 : keyboard.matches("focusPrevious", event) ? -1 : 0;
        if (!direction && !keyboard.matches("focusFirst", event) && !keyboard.matches("focusLast", event))
            return;
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
        if (scope === host)
            closeTo(0);
        else
            closeTo(levels.findIndex((level) => level.menu === scope) + 1);
        items[next]?.focus();
    };
    const onToggle = (event) => {
        if (event.target === host && event.newState === "closed")
            close();
    };
    if (options.captureTriggers) {
        document.addEventListener("contextmenu", onContextMenu);
        document.addEventListener("click", onTriggerClick);
    }
    host.addEventListener("click", onClick);
    host.addEventListener("mouseover", onHover);
    host.addEventListener("keydown", onKey);
    host.addEventListener("toggle", onToggle);
    cleanup(() => {
        cancelAnimationFrame(pendingContextOpen);
        close();
        delete host.openFor;
        delete host.closeMenu;
        delete host.isOpen;
        if (options.captureTriggers) {
            document.removeEventListener("contextmenu", onContextMenu);
            document.removeEventListener("click", onTriggerClick);
        }
        host.removeEventListener("click", onClick);
        host.removeEventListener("mouseover", onHover);
        host.removeEventListener("keydown", onKey);
        host.removeEventListener("toggle", onToggle);
    });
}
function showSheet(menu) {
    menu.showPopover();
    return () => { };
}
