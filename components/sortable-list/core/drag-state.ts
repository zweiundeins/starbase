// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
export type DragState<ItemId, Target> =
  | { kind: "idle" }
  | { kind: "previewing"; itemId: ItemId; target: Target | null }
  | { kind: "committing"; itemId: ItemId; target: Target };

export type DragEvent<ItemId, Target> =
  | { type: "begin"; itemId: ItemId }
  | { type: "preview"; target: Target | null }
  | { type: "commit"; target: Target }
  | { type: "cancel" };

export function reduceDragState<ItemId, Target>(
  state: DragState<ItemId, Target>,
  event: DragEvent<ItemId, Target>,
): DragState<ItemId, Target> {
  switch (event.type) {
    case "begin":
      return { kind: "previewing", itemId: event.itemId, target: null };
    case "preview":
      return state.kind === "previewing" ? { ...state, target: event.target } : state;
    case "commit":
      return state.kind === "previewing" && state.target !== null
        ? { kind: "committing", itemId: state.itemId, target: event.target }
        : state;
    case "cancel":
      return { kind: "idle" };
  }
}

export function createDragState<ItemId, Target>() {
  let state: DragState<ItemId, Target> = { kind: "idle" };
  return {
    read: () => state,
    send: (event: DragEvent<ItemId, Target>) => {
      state = reduceDragState(state, event);
      return state;
    },
    reset: () => {
      state = { kind: "idle" };
      return state;
    },
  };
}
