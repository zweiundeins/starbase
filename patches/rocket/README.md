# Rocket patches

Applied to the Datastar v1.0.5 tag by `go tool task vendor` (`scripts/vendor-rocket.sh`) to build `static/vendor/datastar-rocket.js`. Drop a patch once a release contains its fix, then rebuild.

| Patch | Upstream issue |
|---|---|
| 0001 a bool prop reads "false" and "0" as false and may declare a default | [#1219](https://github.com/starfederation/datastar/issues/1219) |
| 0002 observers hear an attribute write whose value decodes the same | [#1223](https://github.com/starfederation/datastar/issues/1223) |

Upstream closed both issues with a different answer, so neither will be released:

- #1219: v1.0.5 (Rocket beta.3) reads a bool prop by presence alone. `checked="false"` and `checked="0"` are true, and `bool.default(…)` throws when the component is defined. 0001 keeps presence for every other value, so `checked="checked"` is true, but reads `"false"` and `"0"` as false and lets an absent attribute take the declared default. Starbase's server says false in markup (`checked="false"`, `open="false"`), and twelve components declare `bool.default(true)`.
- #1223: upstream keeps notifying only when the decoded value changes. 0002 makes an attribute write whose text changed reach `observeProps`, so `value=""` written onto an element rendered without `value` is heard.

v1.0.5 has the other ten fixes this folder carried for v1.0.4: #1209, #1217, #1218, #1220, #1221, #1222, #1224, #1225, #1226 and #1227. Their repro pages in `docs/repro/` and the cases in the filed texts pass on the official v1.0.5 bundle. Rocket beta.3 makes form association its own way (`formAssociated`, `formValue` and the `onForm…` callbacks in the definition, `internals` in the setup context), not with the setup callbacks Starbase's patch had; no component used them.

The issue texts as filed are in `docs/upstream/`.
