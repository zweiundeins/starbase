// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
/** Bind text and inert metadata; never interpolate into executable directives or URLs. */
export function bindTemplate(template: HTMLTemplateElement, context: Record<string, string>): DocumentFragment {
  const fragment = template.content.cloneNode(true) as DocumentFragment;
  const replace = (value: string) =>
    value.replace(/\{([a-zA-Z][\w]*)\}/g, (match, key: string) => context[key] ?? match);
  const walker = document.createTreeWalker(fragment, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    if (node.nodeType === Node.TEXT_NODE) node.textContent = replace(node.textContent ?? "");
    else if (node instanceof Element) {
      for (const attribute of [...node.attributes]) {
        if (
          attribute.name.startsWith("data-menu-param-") ||
          attribute.name === "aria-label" ||
          attribute.name === "title"
        )
          node.setAttribute(attribute.name, replace(attribute.value));
      }
    }
  }
  return fragment;
}
