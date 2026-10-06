// Generated from inline-edit.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import { rocket } from "datastar";
import { inlineEditContract, } from "./contracts/inline-edit.js";
rocket("sb-inline-edit", {
    mode: "light",
    setup({ host, cleanup }) {
        let lastPress = 0;
        let cancelling = false;
        const contextId = () => host.dataset.contextId ?? "";
        const emit = (name, detail) => host.dispatchEvent(new CustomEvent(name, { bubbles: true, composed: true, detail }));
        const inputFor = (target) => {
            const candidate = target instanceof Element && target.closest(inlineEditContract.selectors.input);
            return candidate instanceof HTMLInputElement && host.contains(candidate) ? candidate : null;
        };
        const onPointerDown = (event) => {
            if (event.button !== 0 || !(event.target instanceof Element))
                return;
            const trigger = event.target.closest(inlineEditContract.selectors.trigger);
            if (!trigger || !host.contains(trigger))
                return;
            const now = event.timeStamp;
            if (!lastPress || now - lastPress > 500) {
                lastPress = now;
                return;
            }
            lastPress = 0;
            emit(inlineEditContract.events.request, { contextId: contextId() });
        };
        const onKeyDown = (event) => {
            const input = inputFor(event.target);
            if (!input || event.defaultPrevented || event.isComposing)
                return;
            if (event.key === "Enter") {
                event.preventDefault();
                input.blur();
            }
            else if (event.key === "Escape") {
                event.preventDefault();
                cancelling = true;
                emit(inlineEditContract.events.cancel, { contextId: contextId() });
                input.blur();
                cancelling = false;
            }
        };
        const onFocusOut = (event) => {
            if (cancelling)
                return;
            const input = inputFor(event.target);
            if (!input)
                return;
            if (input.value === host.querySelector(inlineEditContract.selectors.value)?.textContent) {
                emit(inlineEditContract.events.cancel, { contextId: contextId() });
            }
            else {
                emit(inlineEditContract.events.commit, { contextId: contextId(), value: input.value });
            }
        };
        host.addEventListener("pointerdown", onPointerDown);
        host.addEventListener("keydown", onKeyDown);
        host.addEventListener("focusout", onFocusOut);
        cleanup(() => {
            host.removeEventListener("pointerdown", onPointerDown);
            host.removeEventListener("keydown", onKeyDown);
            host.removeEventListener("focusout", onFocusOut);
        });
    },
});
