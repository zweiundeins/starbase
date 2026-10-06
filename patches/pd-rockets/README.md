# Patches to PD rockets

`go run ./cmd/vendorpd` applies these to [PD rockets](https://github.com/derekr/pd-rockets) (by derekr) at the
pinned release before it vendors the surfaces. Each one is a commit against that repository, ready to offer
upstream; drop it once a release carries the change.

| Patch | What it changes | Upstream |
|---|---|---|
| 0001 | `pd-context-menu`'s setup types its host as the plain element Rocket hands it and casts it where the menu methods are installed | not offered yet |
| 0002 | `core/ownership.ts` keeps its host registry on `globalThis` (`Symbol.for`), so surfaces that each carry their own copy of core, as Starbase's folders do, still nest | not offered yet |
| 0003 | `core/flip.ts` skips its move animation under `prefers-reduced-motion: reduce` | not offered yet |

To make one: clone the repository at the pinned tag, commit the change with a neutral author (their AGENTS.md asks
for one), and `git format-patch` it into this folder with the next number. Keep a patch to one surface's own files
(`rocket/<surface>/`, `contracts/<surface>.ts`) where you can: a change to `core/` reaches every surface's copy.
