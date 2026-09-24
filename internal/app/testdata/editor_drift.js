// sbEditorDrift measures whether sb-code-editor's three text layers line up:
// the highlighted <pre> (what you see), the transparent textarea (where the
// caret is) and the line-number gutter. For every visible editor with enough
// lines, at each font size, it compares the first and the last non-empty line
// of the highlighting with where the textarea puts those lines (its top,
// padding and line height) and with the gutter's number for that line.
// A drift means the highlighting slides away from the caret line by line.
//
// Used by TestCodeEditorLinesAlign (editor_drift_test.go) and by hand from a
// CDP session: sbEditorDrift(['', '1.25rem']) resolves to one row per editor
// and size.
globalThis.sbEditorDrift = async (sizes = ['', '1.25rem'], minLines = 8) => {
	const frame = () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
	const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

	// Editors in the page and one shadow root deep (sb-code-playground's).
	const editors = () => {
		const out = [...document.querySelectorAll('sb-code-editor')]
		for (const host of document.querySelectorAll('*')) if (host.shadowRoot) out.push(...host.shadowRoot.querySelectorAll('sb-code-editor'))
		return [...new Set(out)]
	}

	// The client rect of the first non-blank character of line n in root's text.
	const lineTop = (root, n) => {
		const text = root.textContent
		let start = 0
		for (let i = 0; i < n; i++) start = text.indexOf('\n', start) + 1
		while (text[start] === ' ' || text[start] === '\t') start++
		const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
		for (let node, seen = 0; (node = walk.nextNode()); seen += node.length) {
			if (start < seen + node.length) {
				const range = document.createRange()
				range.setStart(node, start - seen)
				range.setEnd(node, start - seen + 1)
				return range.getClientRects()[0]?.top ?? range.getBoundingClientRect().top
			}
		}
		return NaN
	}

	// Wait for the components and the highlighting (Prism loads on demand:
	// until then the code shows as plain text, without tokens).
	for (let i = 0; i < 100; i++) {
		const ready = editors().filter((e) => e.shadowRoot?.querySelector('pre code .token'))
		if (ready.length) break
		await sleep(100)
	}
	await sleep(300)

	const rows = []
	editors().forEach((ed, index) => {
		ed.dataset.driftIndex = index
	})
	for (const size of sizes) {
		for (const ed of editors()) {
			if (size) ed.style.setProperty('--sb-code-editor-font-size', size)
			else ed.style.removeProperty('--sb-code-editor-font-size')
		}
		await frame()
		await sleep(100)
		for (const ed of editors()) {
			const sr = ed.shadowRoot
			const ta = sr?.querySelector('textarea'), code = sr?.querySelector('pre code'), gutter = sr?.querySelector('.gutter')
			if (!ta || !code || ta.getBoundingClientRect().height === 0) continue
			const lines = code.textContent.split('\n')
			const filled = lines.map((l, i) => (l.trim() ? i : -1)).filter((i) => i >= 0)
			if (filled.length < 2 || filled.at(-1) + 1 < minLines) continue
			const first = filled[0], last = filled.at(-1)
			const cs = getComputedStyle(ta)
			// Chrome lays lines out in 1/64 px units (20.8 px is 1331/64), the textarea's too.
			const lh = Math.round(parseFloat(cs.lineHeight) * 64) / 64
			const origin = ta.getBoundingClientRect().top + parseFloat(cs.paddingTop) + parseFloat(cs.borderTopWidth)
			// Where the glyph sits inside its line is the same on every line, so
			// only the difference between the first and the last line counts.
			const inLine = (top, n) => top - (origin + n * lh)
			const pre = [lineTop(code, first), lineTop(code, last)]
			const row = {
				editor: ed.getAttribute('label') || ed.closest('[data-file]')?.dataset.file || 'editor ' + ed.dataset.driftIndex,
				size: size || 'default',
				lines: last + 1,
				lineHeight: lh,
				codeLineHeight: parseFloat(getComputedStyle(code).lineHeight) || getComputedStyle(code).lineHeight,
				fonts: { textarea: cs.fontFamily + ' ' + cs.fontSize, code: getComputedStyle(code).fontFamily + ' ' + getComputedStyle(code).fontSize },
				caretDrift: +(inLine(pre[1], last) - inLine(pre[0], first)).toFixed(2),
			}
			if (gutter && getComputedStyle(gutter).display !== 'none') {
				row.gutterFirst = +(lineTop(gutter, first) - pre[0]).toFixed(2)
				row.gutterLast = +(lineTop(gutter, last) - pre[1]).toFixed(2)
			}
			row.ok = Math.abs(row.caretDrift) <= 1 && (row.gutterFirst === undefined || (Math.abs(row.gutterFirst) <= 1 && Math.abs(row.gutterLast) <= 1))
			rows.push(row)
		}
	}
	for (const ed of editors()) ed.style.removeProperty('--sb-code-editor-font-size')
	return rows
}
