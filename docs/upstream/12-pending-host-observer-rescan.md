**Title:** Rocket: reordering n children with `replaceChildren()` costs O(n²) in the pending-host observer

**Filed:** not yet

### Bug Report

The bundle starts a document-wide `MutationObserver` for hosts whose definition has not run yet (`observePendingRocketHosts`, runtime.ts:402 to :437). For every `childList` record it calls `markPendingRocketHosts(record.target)` (runtime.ts:419 to :421), which walks the target's whole subtree with `querySelectorAll('*')`. `replaceChildren(...nodes)` with nodes that are already children of the parent queues one removal record per node before the insertion, so the parent's subtree is walked once per node. Reordering the rows of a table (8 cells each) takes 206 ms for 250 rows, 760 ms for 500 and 2.9 s for 1000. Filling it with new rows, a single record, takes 6 to 45 ms. Every page that loads the Rocket bundle pays this, with or without a component on it.

We found it in nfsen-ng's result table, where going from 100 to 250 rows per page took 633 ms of script; it now empties the body first and appends the new page.

### Reproduce

**CodePen:** (to create with `codepen.html`, #12; open the browser devtools console, not CodePen's).

```html
<!doctype html>
<table><tbody id="rows"></tbody></table>
<script type="module">
  import 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js'
  const row = (i) => {
    const tr = document.createElement('tr')
    for (let c = 0; c < 8; c++) tr.insertCell().textContent = i + '.' + c
    return tr
  }
  for (const n of [250, 500, 1000]) {
    rows.replaceChildren(...Array.from({ length: n }, (_, i) => row(i)))
    await new Promise((r) => setTimeout(r))
    const t0 = performance.now()
    rows.replaceChildren(...[...rows.rows].reverse()) // the same rows, reordered
    await new Promise((r) => setTimeout(r)) // after the mutation observers ran
    console.log(n, 'rows:', Math.round(performance.now() - t0), 'ms') // about 200, 750, 2900
  }
</script>
```

### Suggested fix

A removal cannot bring a pending host, and one walk of a parent per batch covers every record of it:

```diff
diff --git a/library/src/rocket/runtime.ts b/library/src/rocket/runtime.ts
index f432cd2..5444a08 100644
--- a/library/src/rocket/runtime.ts
+++ b/library/src/rocket/runtime.ts
@@ -402,6 +402,9 @@ const markPendingRocketHosts = (root: ParentNode) => {
 const observePendingRocketHosts = () => {
   markPendingRocketHosts(document)
   new MutationObserver((records) => {
+    // replaceChildren() over nodes already in the parent queues a record per node: scan each
+    // parent once per batch, and not for a removal, which cannot bring a pending host.
+    const parents = new Set<Element>()
     for (const record of records) {
       if (record.type === 'attributes' && record.target instanceof Element) {
         const name = record.attributeName
@@ -416,10 +419,11 @@ const observePendingRocketHosts = () => {
       for (const node of record.addedNodes) {
         if (node instanceof Element) markPendingRocketHosts(node)
       }
-      if (record.target instanceof Element) {
-        markPendingRocketHosts(record.target)
+      if (record.addedNodes.length && record.target instanceof Element) {
+        parents.add(record.target)
       }
     }
+    for (const parent of parents) markPendingRocketHosts(parent)
   }).observe(document.documentElement, {
     subtree: true,
     childList: true,
```

With it, the same reorders take 24 to 27 ms, 43 to 44 ms and 22 to 85 ms (three runs each), and a child added to a pending shadow host before its definition still gets the marker and binds after the upgrade (checked in Chrome).

We run this patch on https://starbase.zweiundeins.gmbh (a gallery of Rocket components) until a fix is released.

I'm happy to open a PR against `develop` if you'd prefer one.

*Claude was used to draft this issue, find the fix and check it in Chrome.*
