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
