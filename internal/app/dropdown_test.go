package app_test

import "testing"

// TestDropdownFollowsNotch checks that sb-dropdown's pixel details follow --sb-notch: at 1 the
// stepped caret, turning in two steps, and the stepped submenu arrow; at 0 triangles and a smooth turn.
func TestDropdownFollowsNotch(t *testing.T) {
	_, body := probe(t, "/components/dropdown", pdPrelude+dropdownNotchJS)
	copyRows(t, body)
}

const dropdownNotchJS = `
try {
	await customElements.whenDefined('sb-dropdown')
	const el = document.createElement('sb-dropdown')
	el.setAttribute('items', '[{"label":"Sort","children":[{"value":"a","label":"A"}]}]')
	document.body.prepend(el)
	await settle()
	const root = el.shadowRoot
	root.querySelector('.trigger').click()
	await until(() => root.querySelector('.more'))
	const look = () => {
		const cs = (q) => getComputedStyle(root.querySelector(q))
		return [cs('.caret').clipPath.includes('6.667px'), cs('.caret').transitionTimingFunction, cs('.more').clipPath.includes('4.5px')]
	}
	document.documentElement.style.setProperty('--sb-notch', '1')
	check('notch 1: the stepped caret, turning in two steps, and the stepped arrow', look(), [false, 'steps(2)', false])
	document.documentElement.style.setProperty('--sb-notch', '0')
	check('notch 0: triangles, and a smooth turn', look(), [true, 'steps(1000)', true])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
