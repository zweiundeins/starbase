// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
import { rocket } from "datastar";
import {
  inlineEditContract,
  type InlineEditCommitDetail,
  type InlineEditRequestDetail,
} from "./contracts/inline-edit.ts";

rocket("sb-inline-edit", {
  mode: "light",
  setup({ host, cleanup }: { host: HTMLElement; cleanup: (fn: () => void) => void }) {
    let lastPress = 0;
    let cancelling = false;
    const contextId = () => host.dataset.contextId ?? "";
    const emit = <T>(name: string, detail: T) =>
      host.dispatchEvent(new CustomEvent<T>(name, { bubbles: true, composed: true, detail }));
    const inputFor = (target: EventTarget | null): HTMLInputElement | null => {
      const candidate =
        target instanceof Element && target.closest<HTMLInputElement>(inlineEditContract.selectors.input);
      return candidate instanceof HTMLInputElement && host.contains(candidate) ? candidate : null;
    };
    const onPointerDown = (event: PointerEvent) => {
      if (event.button !== 0 || !(event.target instanceof Element)) return;
      const trigger = event.target.closest(inlineEditContract.selectors.trigger);
      if (!trigger || !host.contains(trigger)) return;
      const now = event.timeStamp;
      if (!lastPress || now - lastPress > 500) {
        lastPress = now;
        return;
      }
      lastPress = 0;
      emit<InlineEditRequestDetail>(inlineEditContract.events.request, { contextId: contextId() });
    };
    const onKeyDown = (event: KeyboardEvent) => {
      const input = inputFor(event.target);
      if (!input || event.defaultPrevented || event.isComposing) return;
      if (event.key === "Enter") {
        event.preventDefault();
        input.blur();
      } else if (event.key === "Escape") {
        event.preventDefault();
        cancelling = true;
        emit<InlineEditRequestDetail>(inlineEditContract.events.cancel, { contextId: contextId() });
        input.blur();
        cancelling = false;
      }
    };
    const onFocusOut = (event: FocusEvent) => {
      if (cancelling) return;
      const input = inputFor(event.target);
      if (!input) return;
      if (input.value === host.querySelector(inlineEditContract.selectors.value)?.textContent) {
        emit<InlineEditRequestDetail>(inlineEditContract.events.cancel, { contextId: contextId() });
      } else {
        emit<InlineEditCommitDetail>(inlineEditContract.events.commit, { contextId: contextId(), value: input.value });
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
