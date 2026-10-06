// Generated from focus-recovery.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Restore a keyboard-moved item after a later DOM patch, unless the user chose another focus target. */
export function installFocusRecovery(host) {
    let pending = null;
    let timeout = null;
    let frame = null;
    const clear = () => {
        pending = null;
        if (timeout)
            clearTimeout(timeout);
        if (frame !== null)
            cancelAnimationFrame(frame);
        timeout = null;
        frame = null;
    };
    const restore = () => {
        frame = null;
        if (!pending)
            return;
        const item = pending.findConfirmed();
        if (!item)
            return;
        if (item !== document.activeElement)
            item.focus({ preventScroll: true });
        clear();
    };
    const observer = new MutationObserver(() => {
        if (pending && frame === null)
            frame = requestAnimationFrame(restore);
    });
    observer.observe(host, { childList: true, attributes: true, subtree: true });
    const onFocus = (event) => {
        if (pending && event.target !== pending.source)
            clear();
    };
    const onPointer = () => clear();
    document.addEventListener("focusin", onFocus, true);
    document.addEventListener("pointerdown", onPointer, true);
    return {
        expect: (source, findConfirmed) => {
            clear();
            pending = { source, findConfirmed };
            timeout = setTimeout(clear, 2000);
        },
        dispose: () => {
            clear();
            observer.disconnect();
            document.removeEventListener("focusin", onFocus, true);
            document.removeEventListener("pointerdown", onPointer, true);
        },
    };
}
