// Generated from keyboard-staging.ts by `go tool task ts`: edit the TypeScript, not this file.
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
