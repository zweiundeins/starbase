# Patches to PD rockets

`go run ./cmd/vendorpd` applies these to [PD rockets](https://github.com/derekr/pd-rockets) (by derekr) at the
pinned release, in file name order, before it vendors the surfaces. Each one is a commit against that repository,
ready to offer upstream; its message says what it changes. Drop it once a release carries the change, and when one
is offered, add the pull request's link to its message. They are proposed in upstream issues, one per surface and
one for `core/`: [#2](https://github.com/derekr/pd-rockets/issues/2) sortable-list, [#3](https://github.com/derekr/pd-rockets/issues/3) drag-group,
[#4](https://github.com/derekr/pd-rockets/issues/4) kanban, [#5](https://github.com/derekr/pd-rockets/issues/5) sortable-tree, [#6](https://github.com/derekr/pd-rockets/issues/6) context-menu,
[#7](https://github.com/derekr/pd-rockets/issues/7) bento, [#8](https://github.com/derekr/pd-rockets/issues/8) inline-edit, [#9](https://github.com/derekr/pd-rockets/issues/9) core.

Numbers: 0001 to 0009 for `core/` and changes across surfaces (0001, the context menu's host type, predates the
blocks), then ten per surface, so patches made on different branches never collide: 0010 sortable-list,
0020 drag-group, 0030 kanban, 0040 sortable-tree, 0050 context-menu, 0060 bento, 0070 inline-edit. A surface whose
block is full gets a further one 100 higher: 0130 kanban.

To make one: clone the repository at the pinned tag, `git am` the patches before yours, commit the change with a
neutral author (their AGENTS.md asks for one), and `git format-patch --start-number <n>` it into this folder. Keep a
surface's patch to its own files (`rocket/<surface>/`, `contracts/<surface>.ts`): a change to `core/` reaches every
surface's copy.
