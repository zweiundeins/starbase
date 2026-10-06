// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Use a server-rendered fragment for the floating pointer preview, or clone the item by default. */
export function dragPreviewFor(item: HTMLElement): HTMLElement {
  const template = item.querySelector<HTMLTemplateElement>(":scope > template[data-sb-preview]");
  if (!template) return item.cloneNode(true) as HTMLElement;
  const preview = document.createElement("div");
  preview.className = template.className;
  preview.append(template.content.cloneNode(true));
  return preview;
}

/** Optional decoration outlet; the surface still owns the target and its geometry. */
export function installTargetIndicator(host: HTMLElement) {
  let indicator: HTMLElement | null = null;
  const clear = () => {
    indicator?.remove();
    indicator = null;
  };
  return {
    show(target: HTMLElement | null, kind: "before" | "end" | "into" | "cell") {
      clear();
      if (!target) return;
      const template =
        host.querySelector<HTMLTemplateElement>(`:scope > template[data-sb-target="${kind}"]`) ??
        host.querySelector<HTMLTemplateElement>(':scope > template[data-sb-target=""]');
      if (!template) return;
      indicator = document.createElement("span");
      indicator.setAttribute("data-sb-target-indicator", kind);
      indicator.setAttribute("aria-hidden", "true");
      indicator.inert = true;
      indicator.style.position = "absolute";
      indicator.style.pointerEvents = "none";
      indicator.append(template.content.cloneNode(true));
      target.append(indicator);
    },
    clear,
  };
}
