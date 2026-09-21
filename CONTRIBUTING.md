# Contributing to Starbase

Thanks for launching something!

## Submitting a component

The easiest way needs no tools at all. Open the
[submission form](https://github.com/zweiundeins/starbase/issues/new?template=new-component.yml),
then paste your code, link the GitHub repository it lives in, or paste a
share link from the site's `/playground`. A bot
validates it and opens a pull request with you as the author.

The house rules (shadow DOM, `--sb-*` tokens with fallbacks, `$$` signals for interaction state, declarative wiring, accessibility) are on the
[Contribute page](content/contribute.md).

## Working on the site

```sh
go tool task live        # dev server with live reload on :7331
go tool task test        # go vet + tests, including validation of every component folder
go tool task manifests   # regenerate component manifests (needs Chrome/Chromium)
go tool task vulncheck   # govulncheck
```

Please read [CLAUDE.md](CLAUDE.md) for the architecture rules (CQRS: commands, the single writer, render streams) and the known Datastar/Rocket pitfalls.

Pull requests run CI: tests, generated-code and manifest checks, govulncheck, and builds for Linux, macOS and Windows.
