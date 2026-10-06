// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
type KeyEvent = Pick<KeyboardEvent, "key" | "code" | "altKey" | "shiftKey" | "metaKey" | "ctrlKey">;

export const focusKeys = {
  focusNext: ["ArrowDown", "j"],
  focusPrevious: ["ArrowUp", "k"],
  focusLeft: ["ArrowLeft", "h"],
  focusRight: ["ArrowRight", "l"],
  focusFirst: ["Home"],
  focusLast: ["End"],
} as const;

/** Keep browser Control shortcuts on non-Mac platforms; macOS uses Command for those. */
export function platformFocusKeys(platform: string): Record<keyof typeof focusKeys, readonly string[]> {
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
} as const;

export const cancelKeys = { cancel: ["Escape"] } as const;
export const resizeKeys = {
  resizeUp: ["Shift+ArrowUp"],
  resizeDown: ["Shift+ArrowDown"],
  resizeLeft: ["Shift+ArrowLeft"],
  resizeRight: ["Shift+ArrowRight"],
} as const;
export const gridKeys = { gridPrevious: ["Alt+PageUp"], gridNext: ["Alt+PageDown"] } as const;

export type KeyboardDirection = { x: number; y: number };

function datasetKey(slot: string): keyof DOMStringMap {
  return `key${slot[0]!.toUpperCase()}${slot.slice(1)}` as keyof DOMStringMap;
}

/** Resolve per-host bindings once; an empty attribute disables its intent. */
export function keyboardBindings<Intent extends string>(
  host: HTMLElement,
  defaults: Record<Intent, readonly string[]>,
  aliases: Partial<Record<Intent, string>> = {},
) {
  const bindings = Object.fromEntries(
    Object.entries(defaults).map(([slot, keys]) => {
      const alias = aliases[slot as Intent];
      const override = host.dataset[datasetKey(slot)] ?? (alias ? host.dataset[datasetKey(alias)] : undefined);
      return [slot, override === undefined ? keys : override.split(" ").filter(Boolean)];
    }),
  ) as Record<Intent, readonly string[]>;
  const matches = (slot: Intent, event: KeyboardEvent): boolean =>
    bindings[slot]?.some(
      (key) =>
        keyMatches(key, event) ||
        (slot === "cancel" && key === "Escape" && event.key === "Escape" && !event.ctrlKey && !event.metaKey),
    ) ?? false;
  const direction = (event: KeyboardEvent, kind: "focus" | "move" | "resize"): KeyboardDirection => ({
    x: matches(`${kind}Left` as Intent, event) ? -1 : matches(`${kind}Right` as Intent, event) ? 1 : 0,
    y: matches(`${kind}${kind === "focus" ? "Previous" : "Up"}` as Intent, event)
      ? -1
      : matches(`${kind}${kind === "focus" ? "Next" : "Down"}` as Intent, event)
        ? 1
        : 0,
  });
  return { bindings, matches, direction };
}

/** Respect earlier page handlers and leave editable controls to the browser. */
export function keyboardItem(
  event: KeyboardEvent,
  itemSelector: string,
  owns: (element: HTMLElement) => boolean,
  interactiveHandle?: string,
): HTMLElement | null {
  if (event.defaultPrevented || event.isComposing || !(event.target instanceof HTMLElement) || !owns(event.target))
    return null;
  const interactive = event.target.closest<HTMLElement>(
    "input, textarea, select, button, a, [contenteditable]:not([contenteditable='false']), [role='textbox']",
  );
  if (interactive && !(interactiveHandle && interactive.matches(interactiveHandle))) return null;
  const item = event.target.closest<HTMLElement>(itemSelector);
  return item && owns(item) ? item : null;
}

/** Match configurable shortcuts, including Option-letter characters on macOS. */
export function keyMatches(value: string, event: KeyEvent): boolean {
  const parts = value.split("+");
  const base = parts.pop() ?? "";
  const alt = parts.some((part) => part.toLowerCase() === "alt");
  const shift = parts.some((part) => part.toLowerCase() === "shift");
  const ctrl = parts.some((part) => part.toLowerCase() === "ctrl");
  const meta = parts.some((part) => ["cmd", "meta"].includes(part.toLowerCase()));
  const letterCodeMatches = alt && /^[a-z]$/i.test(base) && event.code === `Key${base.toUpperCase()}`;
  return (
    (base === event.key || letterCodeMatches) &&
    event.altKey === alt &&
    event.shiftKey === shift &&
    event.ctrlKey === ctrl &&
    event.metaKey === meta
  );
}
