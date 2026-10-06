// Generated from keyboard-staging.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Modifier-held preview lifecycle; surfaces own their target and semantic commit. */
export function installKeyboardStaging(options) {
    let staged = null;
    const cancel = () => {
        if (!staged)
            return;
        staged = null;
        options.host.removeAttribute("data-key-staging");
        options.onCancel();
    };
    const commit = () => {
        if (!staged)
            return;
        const value = staged.value;
        staged = null;
        options.host.removeAttribute("data-key-staging");
        options.onCommit(value);
    };
    const onKeyUp = (event) => {
        if (staged?.releaseKey === event.key)
            commit();
    };
    const onPointerDown = (event) => {
        if (event.target instanceof HTMLElement && options.owns(event.target))
            cancel();
    };
    const onFocus = (event) => {
        if (staged && event.target !== staged.source)
            cancel();
    };
    window.addEventListener("keyup", onKeyUp);
    window.addEventListener("blur", commit);
    options.host.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("focusin", onFocus, true);
    return {
        get current() {
            return staged?.value ?? null;
        },
        set(source, value, event, releaseKey) {
            staged = {
                source,
                value,
                releaseKey: releaseKey ??
                    (event.altKey
                        ? "Alt"
                        : event.shiftKey
                            ? "Shift"
                            : event.ctrlKey
                                ? "Control"
                                : event.metaKey
                                    ? "Meta"
                                    : event.key),
            };
            options.host.setAttribute("data-key-staging", "");
        },
        cancel,
        dispose() {
            cancel();
            window.removeEventListener("keyup", onKeyUp);
            window.removeEventListener("blur", commit);
            options.host.removeEventListener("pointerdown", onPointerDown);
            document.removeEventListener("focusin", onFocus, true);
        },
    };
}
