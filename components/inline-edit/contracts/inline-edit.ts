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

export const inlineEditContract = {
  tag: "sb-inline-edit",
  selectors: {
    trigger: "[data-inline-edit-trigger]",
    value: "[data-inline-edit-value]",
    input: "[data-inline-edit-input]",
  },
  events: {
    request: "sb-inline-edit-request",
    commit: "sb-inline-edit-commit",
    cancel: "sb-inline-edit-cancel",
  },
} as const;

export type InlineEditRequestDetail = { contextId: string };
export type InlineEditCommitDetail = { contextId: string; value: string };
