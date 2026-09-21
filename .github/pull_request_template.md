<!--
Adding a component? The easiest way is the "Submit a component" issue form:
no tools needed, and a bot opens this pull request for you.

Opening a PR by hand works too. Please check:
-->

- [ ] One folder `components/<slug>/` with `README.md`, `<slug>.js`, `manifest.json`
- [ ] `manifest.json` regenerated: `go tool task manifests`
- [ ] `go tool task test` passes
- [ ] Styles use `--sb-*` tokens with fallbacks; interaction state lives in `$$` signals
- [ ] I license my contribution under the MIT license
