package app_test

import "testing"

// TestRadioGroupSegmented drives sb-radio-group's segmented variant with real
// pointer and key input: the pills sit in a row sized like sb-button and
// wrap in a narrow container, a click and the arrow keys pick (skipping a
// disabled pill), the form submits the pick, confirm marks it pending, the
// chosen pill stays visible in forced colours, and --sb-notch: 0 rounds the
// ends. STARBASE_SHOTS=<dir> saves screenshots in both themes and smooth.
// With STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build.
func TestRadioGroupSegmented(t *testing.T) {
	copyRows(t, probeCDP(t, "/", segmentedJS))
}

const segmentedJS = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const cdp = async (...cmds) => {
	const r = await fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) })
	if (!r.ok) throw new Error('cdp: ' + (await r.text()))
	return r.json()
}
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const ints = new WeakMap()
const attach = HTMLElement.prototype.attachInternals
HTMLElement.prototype.attachInternals = function () {
	const i = attach.call(this)
	ints.set(this, i)
	return i
}
const rect = (e) => e.getBoundingClientRect()
const shot = (name, el, pad = 12) => { const r = rect(el); return cdp({ method: 'shot', params: { name, clip: { x: r.left + scrollX - pad, y: r.top + scrollY - pad, width: r.width + 2 * pad, height: r.height + 2 * pad } } }) }
const click = async (el) => {
	const r = rect(el), p = { x: r.left + r.width / 2, y: r.top + r.height / 2, button: 'left', clickCount: 1 }
	await cdp({ method: 'Input.dispatchMouseEvent', params: { type: 'mouseMoved', ...p } }, { method: 'Input.dispatchMouseEvent', params: { type: 'mousePressed', ...p } }, { method: 'Input.dispatchMouseEvent', params: { type: 'mouseReleased', ...p } })
	await settle()
}
const codes = { ArrowRight: 39, ArrowLeft: 37, Home: 36, End: 35 }
const press = async (key) => {
	const p = { key, code: key, windowsVirtualKeyCode: codes[key] }
	await cdp({ method: 'Input.dispatchKeyEvent', params: { type: 'rawKeyDown', ...p } }, { method: 'Input.dispatchKeyEvent', params: { type: 'keyUp', ...p } })
	await settle()
}
await customElements.whenDefined('sb-radio-group')
const stage = document.createElement('div')
stage.style.cssText = 'position: relative; z-index: 10; padding: 24px; background: var(--sb-bg); color: var(--sb-text-1); display: grid; gap: 24px; inline-size: 32rem'
document.body.prepend(stage)
scrollTo(0, 0)
stage.innerHTML = '<form id="mail-form"><sb-radio-group id="mail" name="mail" label="E-Mail bei neuen Beiträgen" variant="segmented" value="daily">' +
	'<sb-radio value="now">Sofort</sb-radio><sb-radio value="daily">Täglich</sb-radio><sb-radio value="weekly" disabled>Wöchentlich</sb-radio><sb-radio value="never">Nie</sb-radio>' +
	'</sb-radio-group></form>' +
	'<sb-radio-group id="yn" label="Anonym teilnehmen?" variant="segmented" size="sm" value="no" confirm><sb-radio value="yes">Ja</sb-radio><sb-radio value="no">Nein</sb-radio></sb-radio-group>' +
	'<div id="narrow" style="inline-size: 9rem"><sb-radio-group id="wrap" label="Schmal" variant="segmented" value="b" options=\'["Sofort","Täglich","Nie"]\'></sb-radio-group></div>' +
	'<div style="display: flex; gap: 8px; align-items: center"><sb-button size="sm">sm</sb-button><sb-button>md</sb-button><sb-button size="lg">lg</sb-button></div>'
await settle(300)
const mail = document.getElementById('mail'), yn = document.getElementById('yn'), wrap = document.getElementById('wrap')
const pills = (el) => [...el.shadowRoot.querySelectorAll('[role="radio"]')]
const pill = (el, v) => el.shadowRoot.querySelector('[data-value="' + v + '"]')
const bg = (el) => getComputedStyle(el).backgroundColor

await group('layout', async () => {
	const p = pills(mail)
	check('four pills in one row', new Set(p.map((x) => Math.round(rect(x).top))).size, 1)
	check('md pills are as tall as an md sb-button', Math.round(rect(p[0]).height), 40)
	check('sm pills are as tall as an sm sb-button', Math.round(rect(pills(yn)[0]).height), 32)
	check('no dot in a pill', p.every((x) => x.querySelector('.dot').getClientRects().length === 0), true)
	check('the radio group is horizontal', ints.get(mail)?.ariaOrientation, 'horizontal')
	check('the chosen pill is filled differently', bg(pill(mail, 'daily')) !== bg(pill(mail, 'now')), true)
	check('parts: item, and checked on the chosen one', [pill(mail, 'daily').part.value, pill(mail, 'now').part.value], ['item checked', 'item'])
	const w = pills(wrap), box = rect(document.getElementById('narrow'))
	check('narrow: the pills wrap onto more rows', new Set(w.map((x) => Math.round(rect(x).top))).size > 1, true)
	check('narrow: every pill stays inside the container', w.every((x) => rect(x).right <= box.right + 0.5 && rect(x).left >= box.left - 0.5), true)
})

await group('pointer and keys', async () => {
	let said = []
	mail.addEventListener('sb-change', (e) => said.push(e.detail.value))
	await click(pill(mail, 'now'))
	check('a click picks', [mail.value, said], ['now', ['now']])
	check('the picked pill has the focus', mail.shadowRoot.activeElement?.dataset.value, 'now')
	await press('ArrowRight')
	check('ArrowRight picks the next pill', mail.value, 'daily')
	await press('ArrowRight')
	check('ArrowRight skips the disabled pill', mail.value, 'never')
	await press('ArrowRight')
	check('and wraps around', mail.value, 'now')
	await press('End')
	check('End picks the last enabled pill', mail.value, 'never')
	await press('Home')
	check('Home picks the first', mail.value, 'now')
	await click(pill(mail, 'weekly'))
	check('a click on a disabled pill picks nothing', mail.value, 'now')
	check('tab order: only the chosen pill', pills(mail).map((x) => x.getAttribute('tabindex')), ['0', '-1', null, '-1'])
	check('sb-change for each pick', said, ['now', 'daily', 'never', 'now', 'never', 'now'])
	check('the form submits the pick', new FormData(document.getElementById('mail-form')).get('mail'), 'now')
	document.getElementById('mail-form').reset()
	await settle()
	check('a form reset brings back the server\'s value', mail.value, 'daily')
})

await group('confirm', async () => {
	await click(pill(yn, 'yes'))
	check('a pick waits for the server', [yn.value, yn.matches(':state(pending)')], ['yes', true])
	yn.revert()
	await settle()
	check('revert() goes back', [yn.value, yn.matches(':state(pending)')], ['no', false])
	yn.setAttribute('pending', '')
	await settle()
	check('the page\'s pending', [yn.matches(':state(pending)'), ints.get(yn)?.ariaBusy], [true, 'true'])
	yn.removeAttribute('pending')
	await settle()
})

await group('notch', async () => {
	check('pixel corners by default: square ends under the notch', getComputedStyle(pill(mail, 'now')).borderTopLeftRadius, '0px')
	mail.style.setProperty('--sb-notch', '0')
	await settle()
	check('--sb-notch: 0 rounds the ends', getComputedStyle(pill(mail, 'now')).borderTopLeftRadius, '20px')
	check('--sb-notch: 0 clips nothing, so a page\'s shadow on the chosen pill shows', getComputedStyle(pill(mail, 'daily')).clipPath, 'none')
	mail.style.removeProperty('--sb-notch')
})

// Hover keeps the text readable: in both themes, and in a theme whose hover border is its ink.
await group('hover', async () => {
	const ctx = Object.assign(document.createElement('canvas'), { width: 1, height: 1 }).getContext('2d', { willReadFrequently: true })
	const rgb = (c) => (ctx.clearRect(0, 0, 1, 1), (ctx.fillStyle = c), ctx.fillRect(0, 0, 1, 1), [...ctx.getImageData(0, 0, 1, 1).data].slice(0, 3))
	const lum = (c) => rgb(c).map((v) => ((v /= 255) <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4)).reduce((a, v, i) => a + v * [0.2126, 0.7152, 0.0722][i], 0)
	const contrast = (a, b) => { const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p); return (x + 0.05) / (y + 0.05) }
	const now = pill(mail, 'now')
	const hover = async () => { const r = rect(now); await cdp({ method: 'Input.dispatchMouseEvent', params: { type: 'mouseMoved', x: r.left + r.width / 2, y: r.top + r.height / 2 } }); await settle(250) }
	const away = () => cdp({ method: 'Input.dispatchMouseEvent', params: { type: 'mouseMoved', x: 1400, y: 900 } })
	for (const theme of ['deep-space', 'daylight', 'ink']) {
		document.documentElement.dataset.sbTheme = theme === 'ink' ? 'daylight' : theme
		if (theme === 'ink') mail.style.setProperty('--sb-control-border-hover', 'var(--sb-control-text)')
		await away()
		await settle(200)
		const rest = bg(now)
		await hover()
		const fill = bg(now), ink = getComputedStyle(now).color
		check(theme + ': hover changes the fill', fill !== rest, true)
		check(theme + ': the text on the hovered pill stays readable (4.5:1)', contrast(fill, ink) >= 4.5 ? 'ok' : contrast(fill, ink).toFixed(2), 'ok')
		await shot('segmented-hover-' + theme, stage)
	}
	mail.style.setProperty('--sb-segment-hover', 'rgb(1, 2, 3)')
	await settle(250)
	check('--sb-segment-hover sets the hover fill', bg(now), 'rgb(1, 2, 3)')
	mail.style.removeProperty('--sb-segment-hover')
	mail.style.removeProperty('--sb-control-border-hover')
	delete document.documentElement.dataset.sbTheme
	await away()
	await settle(200)
})

await group('screenshots', async () => {
	document.activeElement?.blur()
	await cdp({ method: 'Input.dispatchMouseEvent', params: { type: 'mouseMoved', x: 1400, y: 900 } })
	document.documentElement.dataset.sbTheme = 'deep-space'
	await settle(200)
	await shot('segmented-deep-space', stage)
	document.documentElement.dataset.sbTheme = 'daylight'
	await settle(200)
	await shot('segmented-daylight', stage)
	document.documentElement.dataset.sbStyle = 'smooth'
	await settle(200)
	await shot('segmented-daylight-smooth', stage)
	document.documentElement.dataset.sbTheme = 'deep-space'
	await settle(200)
	await shot('segmented-deep-space-smooth', stage)
	delete document.documentElement.dataset.sbTheme
	delete document.documentElement.dataset.sbStyle
	await settle(200)
})

await group('forced colours', async () => {
	await cdp({ method: 'Emulation.setEmulatedMedia', params: { features: [{ name: 'forced-colors', value: 'active' }] } })
	await settle(200)
	const on = pill(mail, 'daily'), off = pill(mail, 'now')
	check('forced colours: the chosen pill is filled apart from the others', bg(on) !== bg(off), true)
	check('forced colours: every pill has a border', parseFloat(getComputedStyle(off).borderTopWidth) >= 1, true)
	await shot('segmented-forced-colors', stage)
	await cdp({ method: 'Emulation.setEmulatedMedia', params: { features: [] } })
})
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
