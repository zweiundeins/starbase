# Rocket patches

Applied to the Datastar v1.0.5 tag by `go tool task vendor` (`scripts/vendor-rocket.sh`) to build `static/vendor/datastar-rocket.js`. Drop a patch once a release contains its fix, then rebuild.

| Patch | Upstream issue |
|---|---|
| 0002 observers hear an attribute write whose value decodes the same | [#1223](https://github.com/starfederation/datastar/issues/1223) |

Upstream closed the issue with a different answer, so the fix will not be released: it keeps notifying only when the decoded value changes. 0002 makes an attribute write whose text changed reach `observeProps`, so `value=""` written onto an element rendered without `value` is heard. The components don't depend on it: each watches its value attribute itself, so they behave the same on the official release.

0001 (a bool prop reads `"false"` and `"0"` as false and may declare a default, [#1219](https://github.com/starfederation/datastar/issues/1219)) is gone. v1.0.5 reads a bool prop by presence alone, as HTML does, and refuses `bool.default(…)`; the components now read `checked`, `open` and `busy` the same way, and flags that are on by default are `oneOf(true, false)`. Install snippets pinned to an older catalog snapshot keep the build it recorded (`/c/datastar@<hash>/`), 0001 included.

v1.0.5 has the other ten fixes this folder carried for v1.0.4: #1209, #1217, #1218, #1220, #1221, #1222, #1224, #1225, #1226 and #1227. Their repro pages in `docs/repro/` and the cases in the filed texts pass on the official v1.0.5 bundle. Rocket beta.3 makes form association its own way (`formAssociated`, `formValue` and the `onForm…` callbacks in the definition, `internals` in the setup context), not with the setup callbacks Starbase's patch had; no component used them.

The issue texts as filed are in `docs/upstream/`.
