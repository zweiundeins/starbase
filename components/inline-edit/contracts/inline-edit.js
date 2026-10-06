// Generated from inline-edit.ts by `go tool task ts`: edit the TypeScript, not this file.
// From PD rockets by derekr (https://github.com/derekr/pd-rockets, v2026-09-28-2), under the Beer-Ware licence
// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd`, pd- names renamed to sb-.
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
};
