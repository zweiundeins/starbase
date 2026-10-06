// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import type { BentoPosition } from "./contracts/bento.ts";

export type Cell = { col: number; row: number; width: number; height: number };
export type PlacedItem = Cell & { id: string };
export type GridLayout = { id: string; columns: number; items: readonly PlacedItem[] };

/** The last valid anchor is columns - width + 1 (coordinates are one-based). */
export function nextBentoColumn(col: number, width: number, columns: number, direction: number): number | null {
  const next = col + direction;
  return next >= 1 && next + width - 1 <= columns ? next : null;
}

export function overlaps(a: Cell, b: Cell): boolean {
  return a.col < b.col + b.width && b.col < a.col + a.width && a.row < b.row + b.height && b.row < a.row + a.height;
}

/** Anchor the changed item, then shift displaced items down to the next free row. */
export function placeWithPush(items: readonly PlacedItem[], moved: PlacedItem, columns: number): PlacedItem[] {
  const width = Math.min(Math.max(1, moved.width), columns);
  const result: PlacedItem[] = [
    {
      ...moved,
      col: Math.min(Math.max(1, moved.col), columns - width + 1),
      row: Math.max(1, moved.row),
      width,
      height: Math.max(1, moved.height),
    },
  ];
  for (const item of items.filter((item) => item.id !== moved.id).sort((a, b) => a.row - b.row || a.col - b.col)) {
    const placed = { ...item, width: Math.min(Math.max(1, item.width), columns), row: Math.max(1, item.row) };
    placed.col = Math.min(Math.max(1, placed.col), columns - placed.width + 1);
    while (result.some((existing) => overlaps(placed, existing))) placed.row++;
    result.push(placed);
  }
  return result;
}

/** Project from confirmed positions, never from an earlier transient preview. */
export function projectBentoLayout(
  grids: readonly GridLayout[],
  itemId: string,
  targetGrid: string,
  target: Cell,
): BentoPosition[] {
  const source = grids.find((grid) => grid.items.some((item) => item.id === itemId));
  const destination = grids.find((grid) => grid.id === targetGrid);
  if (!source || !destination) return [];
  const projected = placeWithPush(destination.items, { id: itemId, ...target }, destination.columns);
  const previousById = new Map(destination.items.map((item) => [item.id, item]));
  return projected.flatMap((item) => {
    const previous = previousById.get(item.id);
    if (
      previous &&
      previous.col === item.col &&
      previous.row === item.row &&
      previous.width === item.width &&
      previous.height === item.height
    )
      return [];
    return [
      { itemId: item.id, grid: destination.id, col: item.col, row: item.row, width: item.width, height: item.height },
    ];
  });
}
