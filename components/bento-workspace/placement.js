// Generated from placement.ts by `go tool task ts`: edit the TypeScript, not this file.
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
/** The last valid anchor is columns - width + 1 (coordinates are one-based). */
export function nextBentoColumn(col, width, columns, direction) {
    const next = col + direction;
    return next >= 1 && next + width - 1 <= columns ? next : null;
}
export function overlaps(a, b) {
    return a.col < b.col + b.width && b.col < a.col + a.width && a.row < b.row + b.height && b.row < a.row + a.height;
}
/** Anchor the changed item, then shift displaced items down to the next free row. */
export function placeWithPush(items, moved, columns) {
    const width = Math.min(Math.max(1, moved.width), columns);
    const result = [
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
        while (result.some((existing) => overlaps(placed, existing)))
            placed.row++;
        result.push(placed);
    }
    return result;
}
/** Project from confirmed positions, never from an earlier transient preview. */
export function projectBentoLayout(grids, itemId, targetGrid, target) {
    const source = grids.find((grid) => grid.items.some((item) => item.id === itemId));
    const destination = grids.find((grid) => grid.id === targetGrid);
    if (!source || !destination)
        return [];
    const projected = placeWithPush(destination.items, { id: itemId, ...target }, destination.columns);
    const previousById = new Map(destination.items.map((item) => [item.id, item]));
    return projected.flatMap((item) => {
        const previous = previousById.get(item.id);
        if (previous &&
            previous.col === item.col &&
            previous.row === item.row &&
            previous.width === item.width &&
            previous.height === item.height)
            return [];
        return [
            { itemId: item.id, grid: destination.id, col: item.col, row: item.row, width: item.width, height: item.height },
        ];
    });
}
