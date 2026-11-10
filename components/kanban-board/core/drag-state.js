// Generated from drag-state.ts by `go tool task ts`: edit the TypeScript, not this file.
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
