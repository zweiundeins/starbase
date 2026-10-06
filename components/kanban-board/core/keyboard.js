// Generated from keyboard.ts by `go tool task ts`: edit the TypeScript, not this file.
export const focusKeys = {
    focusNext: ["ArrowDown", "j"],
    focusPrevious: ["ArrowUp", "k"],
    focusLeft: ["ArrowLeft", "h"],
    focusRight: ["ArrowRight", "l"],
    focusFirst: ["Home"],
    focusLast: ["End"],
};
/** Keep browser Control shortcuts on non-Mac platforms; macOS uses Command for those. */
export function platformFocusKeys(platform) {
    return platform.startsWith("Mac")
        ? {
            ...focusKeys,
            focusNext: [...focusKeys.focusNext, "Ctrl+n"],
            focusPrevious: [...focusKeys.focusPrevious, "Ctrl+p"],
        }
        : focusKeys;
}
export const moveKeys = {
    moveUp: ["Alt+ArrowUp", "Alt+k"],
    moveDown: ["Alt+ArrowDown", "Alt+j"],
    moveLeft: ["Alt+ArrowLeft", "Alt+h"],
    moveRight: ["Alt+ArrowRight", "Alt+l"],
};
export const cancelKeys = { cancel: ["Escape"] };
export const resizeKeys = {
    resizeUp: ["Shift+ArrowUp"],
    resizeDown: ["Shift+ArrowDown"],
    resizeLeft: ["Shift+ArrowLeft"],
    resizeRight: ["Shift+ArrowRight"],
};
export const gridKeys = { gridPrevious: ["Alt+PageUp"], gridNext: ["Alt+PageDown"] };
function datasetKey(slot) {
    return `key${slot[0].toUpperCase()}${slot.slice(1)}`;
}
/** Resolve per-host bindings once; an empty attribute disables its intent. */
export function keyboardBindings(host, defaults, aliases = {}) {
    const bindings = Object.fromEntries(Object.entries(defaults).map(([slot, keys]) => {
        const alias = aliases[slot];
        const override = host.dataset[datasetKey(slot)] ?? (alias ? host.dataset[datasetKey(alias)] : undefined);
        return [slot, override === undefined ? keys : override.split(" ").filter(Boolean)];
    }));
    const matches = (slot, event) => bindings[slot]?.some((key) => keyMatches(key, event) ||
        (slot === "cancel" && key === "Escape" && event.key === "Escape" && !event.ctrlKey && !event.metaKey)) ?? false;
    const direction = (event, kind) => ({
        x: matches(`${kind}Left`, event) ? -1 : matches(`${kind}Right`, event) ? 1 : 0,
        y: matches(`${kind}${kind === "focus" ? "Previous" : "Up"}`, event)
            ? -1
            : matches(`${kind}${kind === "focus" ? "Next" : "Down"}`, event)
                ? 1
                : 0,
    });
    return { bindings, matches, direction };
}
/** Respect earlier page handlers and leave editable controls to the browser. */
export function keyboardItem(event, itemSelector, owns, interactiveHandle) {
    if (event.defaultPrevented || event.isComposing || !(event.target instanceof HTMLElement) || !owns(event.target))
        return null;
    const interactive = event.target.closest("input, textarea, select, button, a, [contenteditable]:not([contenteditable='false']), [role='textbox']");
    if (interactive && !(interactiveHandle && interactive.matches(interactiveHandle)))
        return null;
    const item = event.target.closest(itemSelector);
    return item && owns(item) ? item : null;
}
/** Match configurable shortcuts, including Option-letter characters on macOS. */
export function keyMatches(value, event) {
    const parts = value.split("+");
    const base = parts.pop() ?? "";
    const alt = parts.some((part) => part.toLowerCase() === "alt");
    const shift = parts.some((part) => part.toLowerCase() === "shift");
    const ctrl = parts.some((part) => part.toLowerCase() === "ctrl");
    const meta = parts.some((part) => ["cmd", "meta"].includes(part.toLowerCase()));
    const letterCodeMatches = alt && /^[a-z]$/i.test(base) && event.code === `Key${base.toUpperCase()}`;
    return ((base === event.key || letterCodeMatches) &&
        event.altKey === alt &&
        event.shiftKey === shift &&
        event.ctrlKey === ctrl &&
        event.metaKey === meta);
}
