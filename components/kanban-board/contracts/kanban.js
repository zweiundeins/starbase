// Generated from kanban.ts by `go tool task ts`: edit the TypeScript, not this file.
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
    moveFirst: ["Alt+Home"],
    moveLast: ["Alt+End"],
    ...cancelKeys,
};
export function kanbanKeyboardConfig(overrides = {}) {
    return {
        ...defaultKanbanKeyboard,
        ...overrides,
    };
}
