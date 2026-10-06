// Generated from kanban.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import { cancelKeys, focusKeys, moveKeys } from "../core/keyboard.js";
export const kanbanContract = {
    tag: "sb-kanban-board",
    selectors: {
        lane: "[data-kanban-lane]",
        card: "[data-kanban-card]",
        cardMain: "[data-kanban-card-main]",
    },
    events: {
        move: "sb-kanban-move",
        select: "sb-kanban-select",
    },
};
export const defaultKanbanKeyboard = {
    selectNext: focusKeys.focusNext,
    selectPrevious: focusKeys.focusPrevious,
    selectLeft: focusKeys.focusLeft,
    selectRight: focusKeys.focusRight,
    ...moveKeys,
    ...cancelKeys,
};
export function kanbanKeyboardConfig(overrides = {}) {
    return {
        ...defaultKanbanKeyboard,
        ...overrides,
    };
}
