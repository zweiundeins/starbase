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

The issue texts as filed are in `docs/upstream/`.
