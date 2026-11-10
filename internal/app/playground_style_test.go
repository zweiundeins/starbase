package app_test

import "testing"

// TestPlaygroundSmoothStyle checks the auto Playground's style switch in Chrome: it sets
// data-sb-style="smooth" on the stage only (the live element gets --sb-notch: 0, the page keeps
// 1), turning it off brings the 8-bit look back, and the copied markup stays the same.
func TestPlaygroundSmoothStyle(t *testing.T) {
	for _, slug := range []string{"select", "button", "dropdown", "modal"} {
		t.Run(slug, func(t *testing.T) {
			t.Parallel()
			_, body := probe(t, "/components/"+slug, pdPrelude+playgroundStyleJS)
			copyRows(t, body)
		})
	}
}

const playgroundStyleJS = `
try {
	const stage = document.querySelector('.playground__stage')
	const live = stage.firstElementChild
	await customElements.whenDefined(live.localName)
	await customElements.whenDefined('sb-toggle')
	const sw = document.querySelector('.playground__style sb-toggle')
	await until(() => sw.shadowRoot?.querySelector('[part=switch]'))
	const code = () => document.querySelector('.playground__code code').textContent
	const before = code()
	const notch = (el) => getComputedStyle(el).getPropertyValue('--sb-notch').trim()
	// sb-select's arrow: a triangle at notch 0 (see TestSelectFollowsNotch).
	const arrow = () => live.shadowRoot?.querySelector('[part=arrow]') ? getComputedStyle(live.shadowRoot.querySelector('[part=arrow]')).clipPath.includes('6.667px') : null
	const look = () => [stage.getAttribute('data-sb-style'), notch(live), notch(document.body), arrow()]
	const flip = async () => (sw.shadowRoot.querySelector('[part=switch]').click(), await settle())
	const tri = live.localName === 'sb-select'
	check('8-bit at first', look(), [null, '1', '1', tri ? false : null])
	await flip()
	check('smooth: the stage only', look(), ['smooth', '0', '1', tri ? true : null])
	check('the copied markup is the same', code(), before)
	await flip()
	check('off: 8-bit again', look(), [null, '1', '1', tri ? false : null])
	if (tri) check('loading can be ticked', !!document.querySelector('.playground__controls sb-toggle[label=loading]'), true)
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
