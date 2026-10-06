// Generated from inline-edit.ts by `go tool task ts`: edit the TypeScript, not this file.
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
import { rocket } from "datastar";
import { inlineEditContract, } from "./contracts/inline-edit.js";
rocket("sb-inline-edit", {
    mode: "light",
    manifest: {
        events: [
            {
                name: inlineEditContract.events.request,
                kind: "custom-event",
                bubbles: true,
                composed: true,
                description: "A double press on the trigger, or Enter or F2 while it has the focus. detail: { contextId }. " +
                    "Render the field.",
            },
            {
                name: inlineEditContract.events.commit,
                kind: "custom-event",
                bubbles: true,
                composed: true,
                description: "Enter in the field, or the field lost the focus, with a text other than the saved one it started from. " +
                    "detail: { contextId, value }. Save it and render the title, or refuse it.",
            },
            {
                name: inlineEditContract.events.cancel,
                kind: "custom-event",
                bubbles: true,
                composed: true,
                description: "Escape in the field, or the field lost the focus with the saved text in it. detail: { contextId }. " +
                    "Render the title again.",
            },
        ],
    },
    setup({ host, cleanup }) {
        let lastPress = 0;
        let cancelling = false;
        // The saved text each field started from, read when it took the focus.
        const startedFrom = new WeakMap();
        const savedText = () => host.querySelector(inlineEditContract.selectors.value)?.textContent;
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
            if (event.defaultPrevented || event.isComposing)
                return;
            const input = inputFor(event.target);
            if (!input) {
                const onTrigger = event.target instanceof Element &&
                    event.target.matches(inlineEditContract.selectors.trigger) &&
                    host.contains(event.target);
                if (onTrigger && (event.key === "Enter" || event.key === "F2")) {
                    event.preventDefault();
                    if (event.repeat)
                        return;
                    emit(inlineEditContract.events.request, { contextId: contextId() });
                }
                return;
            }
            if (event.key === "Enter") {
                event.preventDefault();
                if (!event.repeat)
                    input.blur();
            }
            else if (event.key === "Escape") {
                event.preventDefault();
                cancelling = true;
                startedFrom.delete(input);
                emit(inlineEditContract.events.cancel, { contextId: contextId() });
                input.blur();
                cancelling = false;
            }
        };
        const onFocusIn = (event) => {
            const input = inputFor(event.target);
            if (input && !startedFrom.has(input))
                startedFrom.set(input, savedText());
        };
        const onFocusOut = (event) => {
            if (cancelling)
                return;
            const input = inputFor(event.target);
            if (!input)
                return;
            // A morph that removes or moves the field blurs it on the way: decide once the page has settled.
            queueMicrotask(() => {
                const root = input.getRootNode();
                const focused = (root instanceof Document || root instanceof ShadowRoot) && root.activeElement === input;
                if (!input.isConnected || focused)
                    return;
                const saved = startedFrom.has(input) ? startedFrom.get(input) : savedText();
                startedFrom.delete(input);
                if (input.value === saved) {
                    emit(inlineEditContract.events.cancel, { contextId: contextId() });
                }
                else {
                    emit(inlineEditContract.events.commit, {
                        contextId: contextId(),
                        value: input.value,
                    });
                }
            });
        };
        host.addEventListener("pointerdown", onPointerDown);
        host.addEventListener("keydown", onKeyDown);
        host.addEventListener("focusin", onFocusIn);
        host.addEventListener("focusout", onFocusOut);
        cleanup(() => {
            host.removeEventListener("pointerdown", onPointerDown);
            host.removeEventListener("keydown", onKeyDown);
            host.removeEventListener("focusin", onFocusIn);
            host.removeEventListener("focusout", onFocusOut);
        });
    },
});
