# Rocket patches

Applied to the Datastar v1.0.4 tag by `go tool task vendor` (`scripts/vendor-rocket.sh`) to build `static/vendor/datastar-rocket.js`. Drop a patch once a release contains its fix, then rebuild.

| Patch | Upstream issue |
|---|---|
| 0001 finish the teardown when deleting the signals throws | [#1217](https://github.com/starfederation/datastar/issues/1217) |
| 0002 keep the element as it is on an atomic move | [#1218](https://github.com/starfederation/datastar/issues/1218) |
| 0003 form-associated components and focus delegation | [#1220](https://github.com/starfederation/datastar/issues/1220) |
| 0004 bool props follow HTML boolean attributes | [#1219](https://github.com/starfederation/datastar/issues/1219) |
| 0005 clean up the shadow tree's attributes on disconnect | [#1221](https://github.com/starfederation/datastar/issues/1221) |
| 0006 instances share one constructed stylesheet per CSS text | [#1222](https://github.com/starfederation/datastar/issues/1222) |
| 0007 observers hear an attribute write whose value decodes the same | [#1223](https://github.com/starfederation/datastar/issues/1223) |
| 0008 a morph that starts inside another keeps the outer one's pantry and id maps | [#1209](https://github.com/starfederation/datastar/issues/1209) |
| 0009 a light component renders inside a data-ignore-morph ancestor | not filed yet: [`docs/upstream/09`](../../docs/upstream/09-render-inside-ignore-morph.md) |
| 0010 a removed element's mount root leaves the observed roots | not filed yet: [`docs/upstream/10`](../../docs/upstream/10-removed-elements-stay-observed.md) |
| 0011 a queued definition applies the shadow host's children that Datastar's first pass skipped | not filed yet: [`docs/upstream/11`](../../docs/upstream/11-queued-definition-children.md) |
| 0012 the pending-host observer scans a parent once per batch, not once per moved child | not filed yet: [`docs/upstream/12`](../../docs/upstream/12-pending-host-observer-rescan.md) |

Upstream fixed #1209 for its next release without a public commit, so 0008 is Starbase's own fix (the one proposed in `docs/repro/rocket-morph-reentrancy/`, plus restoring the outer morph's id maps). Drop it with the release that has upstream's. `docs/upstream/08` is a comment for #1209 with a case that needs the id maps restored and throws nothing (`docs/repro/rocket-morph-ids/`).

The issue texts as filed are in `docs/upstream/`.
