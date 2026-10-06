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
/** Restore a keyboard-moved item after a later DOM patch, unless the user chose another focus target. */
export function installFocusRecovery(host: HTMLElement): {
  expect: (source: HTMLElement, findConfirmed: () => HTMLElement | null) => void;
  dispose: () => void;
} {
  let pending: { source: HTMLElement; findConfirmed: () => HTMLElement | null } | null = null;
  let timeout: ReturnType<typeof setTimeout> | null = null;
  let frame: number | null = null;
  const clear = () => {
    pending = null;
    if (timeout) clearTimeout(timeout);
    if (frame !== null) cancelAnimationFrame(frame);
    timeout = null;
    frame = null;
  };
  const restore = () => {
    frame = null;
    if (!pending) return;
    const item = pending.findConfirmed();
    if (!item) return;
    if (item !== document.activeElement) item.focus({ preventScroll: true });
    clear();
  };
  const observer = new MutationObserver(() => {
    if (pending && frame === null) frame = requestAnimationFrame(restore);
  });
  observer.observe(host, { childList: true, attributes: true, subtree: true });
  const onFocus = (event: FocusEvent) => {
    if (pending && event.target !== pending.source) clear();
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
