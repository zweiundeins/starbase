// Generated from drag-state.ts by `go tool task ts`: edit the TypeScript, not this file.
export function reduceDragState(state, event) {
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
export function createDragState() {
    let state = { kind: "idle" };
    return {
        read: () => state,
        send: (event) => {
            state = reduceDragState(state, event);
            return state;
        },
        reset: () => {
            state = { kind: "idle" };
            return state;
        },
    };
}
