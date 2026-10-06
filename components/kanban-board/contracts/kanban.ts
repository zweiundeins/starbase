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

import { cancelKeys, focusKeys, moveKeys } from "../core/keyboard.ts";

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
} as const;

export type KanbanMoveDetail = {
  cardId: string;
  col: number;
  before: string;
};

export type KanbanSelectDetail = { cardId: string };

export type KanbanKeySlot =
  | "selectNext"
  | "selectPrevious"
  | "selectLeft"
  | "selectRight"
  | "moveUp"
  | "moveDown"
  | "moveLeft"
  | "moveRight"
  | "moveFirst"
  | "moveLast"
  | "cancel";

export type KanbanKeyboard = Partial<Record<KanbanKeySlot, readonly string[]>>;

export const defaultKanbanKeyboard: Readonly<Record<KanbanKeySlot, readonly string[]>> = {
  selectNext: focusKeys.focusNext,
  selectPrevious: focusKeys.focusPrevious,
  selectLeft: focusKeys.focusLeft,
  selectRight: focusKeys.focusRight,
  ...moveKeys,
  moveFirst: ["Alt+Home"],
  moveLast: ["Alt+End"],
  ...cancelKeys,
};

export function kanbanKeyboardConfig(overrides: KanbanKeyboard = {}): Record<KanbanKeySlot, readonly string[]> {
  return {
    ...defaultKanbanKeyboard,
    ...overrides,
  };
}
