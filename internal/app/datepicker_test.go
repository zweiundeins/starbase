package app_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestDatePickerTime checks sb-date-picker's time of day in the browser: values, zones, typed text,
// the draft that Apply or Enter commits, the time row's keys, min and max, pending and revert().
func TestDatePickerTime(t *testing.T) {
	_, body := probe(t, "/", datePickerTimeJS)
	var rows []struct {
		Step, Got, Want, Error string
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range rows {
		switch {
		case r.Error != "":
			t.Errorf("%s: %s", r.Step, r.Error)
		case r.Got != r.Want:
			t.Errorf("%s: got %s, want %s", r.Step, r.Got, r.Want)
		}
	}
}

const datePickerTimeJS = `
const settle = () => new Promise((r) => setTimeout(r, 60))
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const box = document.createElement('div')
document.body.append(box)
await customElements.whenDefined('sb-date-picker')
const make = async (attrs) => {
	const el = document.createElement('sb-date-picker')
	for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v)
	box.append(el)
	await settle()
	return el
}
const part = (el, sel) => el.shadowRoot.querySelector(sel)
const active = (el) => el.shadowRoot.activeElement
const key = (target, key, o) => target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, composed: true, cancelable: true, ...o }))
const events = (el) => {
	const log = []
	el.addEventListener('change', () => log.push('change'))
	el.addEventListener('sb-change', (e) => log.push(e.detail.value))
	return log
}
const cell = (el, d) => [...el.shadowRoot.querySelectorAll('td')].find((td) => td.textContent === String(d))
const picked = (el) => [...el.shadowRoot.querySelectorAll('td[aria-selected=true]')].map((td) => td.textContent)
// The time row of end i as it shows: its visible segments.
const time = (el, i = 0) => ['b', 'h', 'm', 's', 'a'].map((k) => part(el, '#' + k + i)).filter((s) => s.getClientRects().length).map((s) => s.value).join(' ')
const typeIn = async (el, s) => {
	const i = part(el, '#i')
	i.value = s
	i.dispatchEvent(new Event('input', { bubbles: true }))
	i.dispatchEvent(new Event('change'))
	await settle()
}
const typeSeg = async (seg, s) => {
	seg.value = s
	seg.dispatchEvent(new InputEvent('input', { bubbles: true, composed: true, data: s.slice(-1), inputType: 'insertText' }))
	await settle()
}

await group('native parity', async () => {
	const el = await make({ time: '' })
	const native = document.createElement('input')
	native.type = 'datetime-local'
	native.step = '1'
	for (const v of ['2026-09-29T08:00', '2026-09-29T08:00:00', '2026-09-29T08:00:30', '2026-09-29 08:00', '2026-9-29T8:00', '2026-02-30T08:00', '2026-09-29T24:00', '2026-09-29', '2026-09-29T08:00Z']) {
		el.value = v
		native.value = v
		check('value ' + v + ' normalises as datetime-local does', el.value, native.value)
	}
	el.remove()
})

await group('zones', async () => {
	const el = await make({ time: '' })
	for (const [tz, v, want] of [
		['Europe/Zurich', '2026-03-29T02:30', '2026-03-29T03:30+02:00'],
		['Europe/Zurich', '2026-10-25T02:30', '2026-10-25T02:30+02:00'],
		['Europe/Zurich', '2026-10-25T00:30:00Z', '2026-10-25T02:30+02:00'],
		['Europe/Zurich', '2026-10-25T01:30:00Z', '2026-10-25T02:30+01:00'],
		['Europe/Zurich', '2026-09-29T12:00:00Z', '2026-09-29T14:00+02:00'],
		['Asia/Kathmandu', '2026-01-01T00:00:00Z', '2026-01-01T05:45+05:45'],
		['America/New_York', '2026-10-14T09:30:15.5-04:00', '2026-10-14T09:30:15-04:00'],
		['UTC', '2026-01-01T00:00:00Z', '2026-01-01T00:00+00:00'],
		['Africa/Monrovia', '1970-01-01T12:00-00:44', '1970-01-01T12:00-00:44'],
	]) {
		el.setAttribute('time-zone', tz)
		await settle()
		el.value = v
		check(tz + ': ' + v, el.value, want)
	}
	const d = new Date('2026-09-29T12:00:00Z')
	const p2 = (n) => String(Math.floor(n)).padStart(2, '0')
	const o = -d.getTimezoneOffset()
	const mine = d.getFullYear() + '-' + p2(d.getMonth() + 1) + '-' + p2(d.getDate()) + 'T' + p2(d.getHours()) + ':' + p2(d.getMinutes()) + (o < 0 ? '-' : '+') + p2(Math.abs(o) / 60) + ':' + p2(Math.abs(o) % 60)
	for (const tz of ['local', 'Not/AZone']) {
		el.setAttribute('time-zone', tz)
		await settle()
		el.value = '2026-09-29T12:00:00Z'
		check('time-zone="' + tz + '" is the viewer\'s zone', el.value, mine)
	}
	el.remove()

	const k = await make({ time: '', inline: '', 'time-zone': 'Europe/Zurich', lang: 'en-GB', value: '2026-10-25T02:30+01:00' })
	const log = events(k)
	check('an overlap time from the server keeps its offset', [k.value, time(k), picked(k)], ['2026-10-25T02:30+01:00', '02 30', ['25']])
	cell(k, 25).click()
	key(part(k, '#m0'), 'ArrowUp')
	key(part(k, '#m0'), 'ArrowDown')
	key(part(k, '#m0'), 'Enter')
	await settle()
	check('applied with the same wall clock, it stays +01:00', [k.value, log], ['2026-10-25T02:30+01:00', []])
	key(part(k, '#h0'), 'ArrowUp')
	key(part(k, '#h0'), 'Enter')
	await settle()
	check('another wall clock is read in the zone', [k.value, log], ['2026-10-25T03:30+01:00', ['change', '2026-10-25T03:30+01:00']])
	k.month = '2026-03'
	await settle()
	cell(k, 29).click()
	key(part(k, '#h0'), 'Home')
	key(part(k, '#h0'), 'ArrowUp')
	key(part(k, '#h0'), 'ArrowUp')
	key(part(k, '#h0'), 'Enter')
	await settle()
	check('a wall clock in the spring gap is the time after it', k.value, '2026-03-29T03:30+02:00')
	k.remove()

	const [east, west] = ['Pacific/Kiritimati', 'Pacific/Niue']
	const days = []
	for (const tz of [east, west]) {
		const z = await make({ time: '', inline: '', 'time-zone': tz, lang: 'en-GB' })
		const want = new Intl.DateTimeFormat('en-GB', { timeZone: tz, day: 'numeric' }).format(Date.now())
		days.push(z.shadowRoot.querySelector('td[aria-current=date]')?.textContent, want)
		z.remove()
	}
	check('today is the zone\'s day', [days[0], days[2]], [days[1], days[3]])
	const tokyo = await make({ time: '', inline: '', 'time-zone': 'Asia/Tokyo', lang: 'en-GB', value: '2026-10-14T23:30:00Z', 'disabled-dates': '["2026-10-15"]' })
	check('the value\'s day and disabled-dates are the zone\'s days', [picked(tokyo), cell(tokyo, 15).getAttribute('aria-disabled'), time(tokyo)], [['15'], 'true', '08 30'])
	tokyo.remove()

	const form = document.createElement('form')
	box.append(form)
	const t = document.createElement('sb-date-picker')
	for (const [k, v] of Object.entries({ time: '', 'time-zone': 'America/New_York', lang: 'en-US', name: 'f', min: '2026-10-14T13:00:00Z' })) t.setAttribute(k, v)
	form.append(t)
	await settle()
	await typeIn(t, '10/14/2026 9:30 AM')
	check('typed text is the zone\'s wall clock', [t.value, part(t, '#i').value.replace(/\s/g, ' ')], ['2026-10-14T09:30-04:00', '10/14/2026, 09:30 AM'])
	await typeIn(t, '2026-10-14T15:30:00Z')
	check('a typed RFC 3339 instant is shown in the zone', t.value, '2026-10-14T11:30-04:00')
	await typeIn(t, '10/14/2026 8:30 AM')
	check('min with an offset bounds typed text', [t.value, part(t, '#i').getAttribute('aria-invalid')], ['2026-10-14T11:30-04:00', 'true'])
	check('a form gets the RFC 3339 string', [...new FormData(form)], [['f', '2026-10-14T11:30-04:00']])
	form.remove()

	const plain = await make({ value: '2026-10-14' })
	plain.value = '2026-10-20'
	plain.setAttribute('step', '900')
	plain.setAttribute('time-zone', 'Europe/Zurich')
	await settle()
	check('without time, a new step or time-zone leaves the local value alone', plain.value, '2026-10-20')
	plain.remove()

	const s = await make({ time: '', 'time-zone': 'Europe/Zurich', value: '2026-09-29T12:00:00Z' })
	s.value = '2026-10-01T08:00+02:00'
	s.setAttribute('time-zone', 'Asia/Kathmandu')
	await settle()
	check('a new time-zone: the server\'s value wins, in the new zone', s.value, '2026-09-29T17:45+05:45')
	s.remove()
})

await group('typing', async () => {
	for (const [lang, s, want] of [
		['de', '14.10.2026, 09:30', '2026-10-14T09:30'],
		['de', '14.10.2026 9:30', '2026-10-14T09:30'],
		['en-US', '10/14/2026, 9:30 PM', '2026-10-14T21:30'],
		['en-US', '10/14/2026 12:15 AM', '2026-10-14T00:15'],
		['en-US', '10/14/2026 21:30', '2026-10-14T21:30'],
		['en-US', '10/14/2026 9:30:15 pm', '2026-10-14T21:30:15'],
		['en-US', '2026-10-14 21:30', '2026-10-14T21:30'],
		['ja', '2026/10/14 9:30', '2026-10-14T09:30'],
		['ko', '2026. 10. 14. 오후 9:30', '2026-10-14T21:30'],
		['ar-EG', '١٤/١٠/٢٠٢٦ ٩:٣٠ م', '2026-10-14T21:30'],
		['fi', '14.10.2026 klo 9.30', '2026-10-14T09:30'],
		['vi', '21:45 14/10/2026', '2026-10-14T21:45'],
		['vi', '14/10/2026 21:45', '2026-10-14T21:45'],
	]) {
		const el = await make({ time: '', lang })
		const log = events(el)
		await typeIn(el, s)
		check(lang + ' "' + s + '"', [el.value, log], [want, ['change', want]])
		el.remove()
	}
	for (const [lang, text, ph] of [['de', '14.10.2026, 09:30', 'dd.mm.yyyy, hh:mm'], ['en-US', '10/14/2026, 09:30 AM', 'mm/dd/yyyy, hh:mm AM']]) {
		const el = await make({ time: '', lang, value: '2026-10-14T09:30' })
		const i = part(el, '#i')
		check(lang + ': the field shows the value in the language, and its pattern', [i.value, i.placeholder].map((s) => s.replace(/\s/g, ' ')), [text, ph])
		el.remove()
	}
	for (const lang of ['vi', 'de', 'en-US', 'ja', 'ko', 'zh-TW', 'ar-EG', 'fa', 'he', 'hi', 'bg', 'fi']) {
		for (const [step, v] of [['60', '2026-10-14T21:30'], ['1', '2026-10-14T09:05:07']]) {
			const el = await make({ time: '', lang, step, value: v })
			const shown = part(el, '#i').value
			el.value = ''
			await typeIn(el, shown)
			check(lang + ': the text the field shows, "' + shown + '", reads back', [el.value, part(el, '#i').getAttribute('aria-invalid')], [v, 'false'])
			el.remove()
		}
	}
	for (const lang of ['en-u-hc-h24', 'de-u-hc-h24']) {
		const el = await make({ time: '', lang, value: '2026-10-14T00:30' })
		const shown = part(el, '#i').value
		el.value = ''
		await typeIn(el, shown)
		check(lang + ': midnight shows as 00, and "' + shown + '" reads back', [/00:30/.test(shown), el.value, part(el, '#i').getAttribute('aria-invalid')], [true, '2026-10-14T00:30', 'false'])
		el.remove()
	}
	for (const [lang, s] of [['de', '14.10.2026 25:00'], ['de', 'Mi 14.10.2026 9:30'], ['de', '14.10.2026'], ['de', '14.10.2026 9:30 Uhr'], ['en-US', '10/14/2026 13:30 PM'], ['de', '2026-10-14T09:30+02:00']]) {
		const el = await make({ time: '', lang, value: '2026-10-14T09:30' })
		const log = events(el)
		await typeIn(el, s)
		check(lang + ' "' + s + '" is invalid, without an event', [el.value, part(el, '#i').getAttribute('aria-invalid'), log], ['2026-10-14T09:30', 'true', []])
		el.remove()
	}
	const r = await make({ time: '', mode: 'range', lang: 'de' })
	const log = events(r)
	await typeIn(r, '14.10.2026 09:30 - 13.10.2026 18:00')
	const want = { start: '2026-10-13T18:00', end: '2026-10-14T09:30' }
	check('a typed range, in order', [r.value, log], [want, ['change', want]])
	r.remove()
})

await group('draft', async () => {
	const el = await make({ time: '', lang: 'en-GB', name: 'd', value: '2026-10-14T09:30', month: '2026-10' })
	const log = events(el)
	el.open = true
	await settle()
	check('the time row shows the value', time(el), '09 30')
	cell(el, 20).click()
	await settle()
	check('a day pick sends no event and leaves the popover open', [log, el.open, picked(el)], [[], true, ['20']])
	check('a day pick moves the focus to the hour', active(el)?.id, 'h0')
	const full = new Intl.DateTimeFormat('en-GB', { dateStyle: 'full', timeZone: 'UTC' }).format(Date.UTC(2026, 9, 20))
	check('the time row is named by its date', part(el, '[part~=time]').getAttribute('aria-label'), full)
	key(active(el), 'ArrowUp')
	await settle()
	check('a segment change sends no event', [log, time(el)], [[], '10 30'])
	key(part(el, '#m0'), 'Enter')
	await settle()
	check('Enter commits the whole value once and closes', [log, el.value, el.open], [['change', '2026-10-20T10:30'], '2026-10-20T10:30', false])
	check('the focus goes back to the button', active(el)?.getAttribute('part'), 'button')
	el.open = true
	await settle()
	cell(el, 22).click()
	key(part(el, '#h0'), 'ArrowUp')
	part(el, '#cal').hidePopover()
	await settle()
	check('closing commits nothing', [log.length, el.value], [2, '2026-10-20T10:30'])
	el.open = true
	await settle()
	check('reopening shows the value, not the dropped draft', [picked(el), time(el)], [['20'], '10 30'])
	el.remove()

	const r = await make({ time: '', mode: 'range', lang: 'en-GB', label: 'Window' })
	const rlog = events(r)
	r.open = true
	await settle()
	r.month = '2026-10'
	await settle()
	const apply = part(r, '[part~=apply]')
	check('Apply waits for the dates, and a row without a date is named by the picker', [apply.getAttribute('aria-disabled'), apply.textContent, part(r, '[part~=time]').getAttribute('aria-label')], ['true', 'Apply', 'Window'])
	cell(r, 16).click()
	await settle()
	check('the first pick of a range keeps the focus in the grid, and Apply waits', [active(r)?.localName, apply.getAttribute('aria-disabled')], ['td', 'true'])
	cell(r, 12).click()
	await settle()
	check('the second pick moves the focus to the start hour', [active(r)?.id, apply.getAttribute('aria-disabled'), rlog], ['h0', null, []])
	check('a new start is 00:00, a new end the last step of the day', [time(r, 0), time(r, 1)], ['00 00', '23 59'])
	await typeSeg(part(r, '#h0'), '08')
	check('two digits move on to the minute', active(r)?.id, 'm0')
	await typeSeg(part(r, '#m0'), '30')
	check('two digits in the start line\'s last field move on to the end line', active(r)?.id, 'h1')
	apply.click()
	await settle()
	check('Apply commits the range once, in order, and closes', [rlog, r.open], [['change', { start: '2026-10-12T08:30', end: '2026-10-16T23:59' }], false])
	r.open = true
	await settle()
	cell(r, 12).click()
	cell(r, 12).click()
	key(part(r, '#h0'), 'End')
	key(part(r, '#h1'), 'Home')
	key(part(r, '#m1'), 'Enter')
	await settle()
	check('ends in the wrong order are put in time order', [rlog.length, rlog.at(-1)], [4, { start: '2026-10-12T00:59', end: '2026-10-12T23:30' }])
	r.remove()
})

await group('keys', async () => {
	const el = await make({ time: '', inline: '', step: '900', lang: 'en-GB', value: '2026-10-14T09:07' })
	const log = events(el)
	let inputs = 0
	el.addEventListener('input', () => inputs++)
	const [h, m] = ['#h0', '#m0'].map((s) => part(el, s))
	m.focus()
	key(m, 'ArrowUp')
	await settle()
	check('Up at step 900 from 09:07 gives 09:15', time(el), '09 15')
	el.value = '2026-10-14T09:45'
	await settle()
	key(m, 'ArrowUp')
	await settle()
	check('Up from 09:45 wraps to 09:00, the hour unchanged', time(el), '09 00')
	key(m, 'ArrowDown')
	await settle()
	check('Down wraps back to the last step', time(el), '09 45')
	key(m, 'Home')
	await settle()
	const home = time(el)
	key(m, 'End')
	key(h, 'Home')
	await settle()
	check('Home and End: the first and last value, whatever the step', [home, time(el)], ['09 00', '00 59'])
	key(h, 'End')
	await settle()
	check('End on the hour', time(el), '23 59')
	h.focus()
	await typeSeg(h, '1')
	check('one digit keeps the focus', [active(el)?.id, time(el)], ['h0', '1 59'])
	await typeSeg(h, '14')
	check('two digits move on to the next segment', [active(el)?.id, time(el)], ['m0', '14 59'])
	await typeSeg(m, '7')
	key(m, 'ArrowUp')
	await settle()
	check('a typed value off the step is kept, and Up goes on from it', time(el), '14 15')
	m.setSelectionRange(0, 0)
	key(m, 'ArrowLeft')
	check('Left at the start of the text moves to the previous segment', active(el)?.id, 'h0')
	h.setSelectionRange(1, 1)
	check('Right inside the text is the caret\'s', key(h, 'ArrowRight'), true)
	h.setSelectionRange(2, 2)
	key(h, 'ArrowRight')
	check('Right at the end of the text moves to the next segment', active(el)?.id, 'm0')
	check('segment changes send no event, and their input events stay inside', [log, inputs], [[], 0])
	key(m, 'Escape')
	await settle()
	check('Escape resets an inline draft to the value', time(el), '09 45')
	h.focus()
	await typeSeg(h, '1')
	key(h, 'Escape')
	await settle()
	check('Escape drops a half-typed digit too', time(el), '09 45')
	key(h, 'Enter')
	await settle()
	check('so Enter then applies the value unchanged', [el.value, log], ['2026-10-14T09:45', []])
	await typeSeg(h, '1')
	el.setAttribute('disabled-dates', '["2026-10-20"]')
	await settle()
	check('a prop the time row doesn\'t show leaves a half-typed digit alone', [active(el)?.id, h.value], ['h0', '1'])
	el.remove()

	for (const step of ['abc', '', '0', '-60']) {
		const z = await make({ time: '', inline: '', step, lang: 'en-GB', value: '2026-10-14T09:30' })
		key(part(z, '#m0'), 'ArrowUp')
		await settle()
		check('step="' + step + '" is 60, as on datetime-local', time(z), '09 31')
		z.remove()
	}

	const us = await make({ time: '', inline: '', lang: 'en-US', value: '2026-10-14T11:30' })
	const [uh, ua] = ['#h0', '#a0'].map((s) => part(us, s))
	check('en-US: hour, minute and the day period after them', time(us), '11 30 AM')
	check('segments are spinbuttons with numeric keyboards, and values', ['inputmode', 'role', 'aria-valuenow', 'aria-valuemin', 'aria-valuemax', 'aria-valuetext'].map((a) => uh.getAttribute(a)), ['numeric', 'spinbutton', '11', '1', '12', '11'])
	check('the day period has no numeric keyboard', [ua.getAttribute('inputmode'), ua.getAttribute('aria-valuetext')], [null, 'AM'])
	key(uh, 'ArrowUp')
	await settle()
	check('the 12-hour hour wraps without changing the day period', time(us), '12 30 AM')
	ua.focus()
	ua.value = 'AMp'
	ua.dispatchEvent(new InputEvent('input', { bubbles: true, composed: true, data: 'p', inputType: 'insertText' }))
	await settle()
	check('p sets PM', time(us), '12 30 PM')
	key(ua, 'ArrowDown')
	await settle()
	check('Up and Down toggle the day period', time(us), '12 30 AM')
	key(ua, 'Enter')
	await settle()
	check('Enter in the day period applies, 12 AM being 0', us.value, '2026-10-14T00:30')
	us.remove()

	const ko = await make({ time: '', inline: '', lang: 'ko', value: '2026-10-14T21:30' })
	check('ko: the day period before the hour', time(ko), '오후 09 30')
	ko.remove()
	const de = await make({ time: '', inline: '', lang: 'de', step: '1', value: '2026-10-14T09:30:05' })
	const names = ['#h0', '#m0', '#s0'].map((s) => part(de, s).getAttribute('aria-label'))
	const dn = new Intl.DisplayNames('de', { type: 'dateTimeField' })
	check('de: seconds with step 1, and the fields named in the language', [time(de), names], ['09 30 05', ['hour', 'minute', 'second'].map((f) => dn.of(f))])
	de.remove()

	const t = await make({ time: '', lang: 'en-US', value: '2026-10-14T09:30' })
	t.open = true
	await settle()
	const seen = []
	for (let n = 0; n < 12; n++) {
		const a = active(t)
		seen.push(a?.localName === 'td' ? 'day' : a?.id || a?.getAttribute('part'))
		key(a, 'Tab')
	}
	check('the Tab cycle includes the segments and Apply', seen, ['day', 'h0', 'm0', 'a0', 'apply', 'nav', 'nav', 'nav', 'nav', 'day', 'h0', 'm0'])
	t.remove()
})

await group('min and max', async () => {
	const el = await make({ time: '', inline: '', lang: 'en-GB', month: '2026-10', min: '2026-10-05T09:30', max: '2026-10-20' })
	check('a day before min cannot be picked', cell(el, 4).getAttribute('aria-disabled'), 'true')
	cell(el, 5).click()
	await settle()
	check('picking min\'s day shows min\'s time', time(el), '09 30')
	key(part(el, '#h0'), 'Home')
	await settle()
	check('a segment change is clamped to min', time(el), '09 30')
	cell(el, 20).click()
	key(part(el, '#h0'), 'End')
	await settle()
	check('max, a date alone, is the end of that day', time(el), '23 30')
	key(part(el, '#m0'), 'End')
	key(part(el, '#m0'), 'Enter')
	await settle()
	check('max\'s last step can be applied', el.value, '2026-10-20T23:59')
	el.remove()

	const q = await make({ time: '', step: '900', lang: 'en-GB', max: '2026-10-20' })
	await typeIn(q, '20/10/2026 23:50')
	check('max, a date alone, takes a time off the step up to the end of that day', [q.value, part(q, '#i').getAttribute('aria-invalid')], ['2026-10-20T23:50', 'false'])
	q.remove()

	const t = await make({ time: '', lang: 'en-GB', min: '2026-10-05T09:30' })
	const log = events(t)
	await typeIn(t, '05/10/2026 09:00')
	check('a typed value before min is invalid', [t.value, part(t, '#i').getAttribute('aria-invalid'), log], ['', 'true', []])
	await typeIn(t, '05/10/2026 09:30')
	check('min itself can be typed', t.value, '2026-10-05T09:30')
	t.remove()
})

await group('confirm and revert', async () => {
	const el = await make({ time: '', inline: '', confirm: '', lang: 'en-GB', month: '2026-10', value: '2026-10-14T09:30' })
	const apply = part(el, '[part~=apply]')
	cell(el, 15).click()
	await settle()
	check('a draft is not pending', el.matches(':state(pending)'), false)
	apply.click()
	await settle()
	check('Apply: pending while the server has the old value', [el.value, el.matches(':state(pending)')], ['2026-10-15T09:30', true])
	el.setAttribute('value', '2026-10-15T09:30')
	await settle()
	check('the server\'s matching value ends it', el.matches(':state(pending)'), false)
	cell(el, 16).click()
	apply.click()
	cell(el, 17).click()
	key(part(el, '#h0'), 'ArrowUp')
	await settle()
	el.revert()
	await settle()
	check('revert() restores the server\'s value and drops the draft', [el.value, el.matches(':state(pending)'), picked(el), time(el)], ['2026-10-15T09:30', false, ['15'], '09 30'])
	el.remove()

	const s = await make({ value: '2026-10-14T09:30:20' })
	check('without time, a date-time is no date', s.value, '')
	s.setAttribute('time', '')
	await settle()
	check('with time, the server\'s value is read again', s.value, '2026-10-14T09:30:20')
	s.value = '2026-10-15T10:00'
	s.setAttribute('step', '900')
	await settle()
	check('a new step: the server\'s value wins over the local one', s.value, '2026-10-14T09:30:20')
	s.remove()
})

await group('forced colours', async () => {
	const el = await make({ time: '', inline: '' })
	const sheets = [...el.shadowRoot.adoptedStyleSheets, ...[...el.shadowRoot.querySelectorAll('style')].map((s) => s.sheet)]
	const forced = sheets.flatMap((s) => [...s.cssRules]).filter((r) => r.media?.mediaText.includes('forced-colors')).flatMap((m) => [...m.cssRules])
	const ring = (sel) => forced.filter((r) => r.selectorText === sel).map((r) => [r.style.outlineWidth, r.style.outlineStyle, r.style.outlineColor]).at(-1)
	check('forced colours: a focused Apply has a ring of its own', ring('[part~="apply"]:focus-visible'), ['2px', 'solid', 'highlight'])
	el.remove()
})

await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`

// TestDatePickerOpenState checks sb-date-picker's :state(open) in Chrome: it follows the calendar
// popover through the button, a pick, the open property and attribute, a close by the browser
// (hidePopover, as Escape and a click outside do), disabled, inline, a server morph, removal and a
// re-attach. An inline calendar is never open.
func TestDatePickerOpenState(t *testing.T) {
	copyRows(t, probeBundle(t, "/", pdPrelude+morphJS+datePickerOpenJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler}))
}

const datePickerOpenJS = `
const box = document.createElement('div')
document.body.prepend(box)
await customElements.whenDefined('sb-date-picker')
let n = 0
const make = async (attrs = {}) => {
	const el = document.createElement('sb-date-picker')
	el.id = 'dp' + ++n
	for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v)
	box.append(el)
	await settle()
	return el
}
const $ = (el, s) => el.shadowRoot.querySelector(s)
const state = (el) => [el.matches(':state(open)'), $(el, '#cal').matches(':popover-open')]
const click = async (el, s) => ($(el, s).click(), await settle())
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const OPEN = [true, true], SHUT = [false, false]

await group('popover', async () => {
	const el = await make({ label: 'Launch', value: '2026-09-15' })
	check('closed at first', state(el), SHUT)
	await click(el, '[part=button]')
	check('the button opens', state(el), OPEN)
	await click(el, '[part=button]')
	check('the button closes', state(el), SHUT)
	await click(el, '[part=button]')
	await click(el, '[part~=day]:not([part~=selected])')
	check('a pick closes', state(el), SHUT)
	el.open = true
	await settle()
	check('open = true opens', state(el), OPEN)
	el.open = false
	await settle()
	check('open = false closes', state(el), SHUT)
	el.setAttribute('open', '')
	await settle()
	check('the server\'s open opens', state(el), OPEN)
	el.setAttribute('open', 'false')
	await settle()
	check('the server\'s open="false" closes', state(el), SHUT)
	await click(el, '[part=button]')
	$(el, '#cal').hidePopover()
	await settle()
	check('a close by the browser (Escape, a click outside)', state(el), SHUT)
	await click(el, '[part=button]')
	el.setAttribute('disabled', '')
	await settle()
	check('disabled closes', state(el), SHUT)
	el.removeAttribute('disabled')
	await click(el, '[part=button]')
	el.setAttribute('inline', '')
	await settle()
	check('inline closes', el.matches(':state(open)'), false)
	el.removeAttribute('inline')
	await settle()
	await click(el, '[part=button]')
	check('a popover again', state(el), OPEN)
	el.remove()
	await settle()
	check('removed while open: no longer open', el.matches(':state(open)'), false)
	box.append(el)
	await settle()
	check('re-attached: closed', state(el), SHUT)
	await click(el, '[part=button]')
	check('re-attached: the button opens', state(el), OPEN)
	el.remove()
})

await group('morphs', async () => {
	const el = await make({ label: 'Launch' })
	await click(el, '[part=button]')
	await morph('<sb-date-picker id="' + el.id + '" label="Launch" value="2026-09-15"></sb-date-picker>')
	check('a morph without open leaves it open', [state(el), el.value], [OPEN, '2026-09-15'])
	await morph('<sb-date-picker id="' + el.id + '" label="Launch" value="2026-09-15" open="false"></sb-date-picker>')
	check('a morph with open="false" closes it', state(el), SHUT)
	el.remove()
})

await group('inline', async () => {
	const el = await make({ inline: '' })
	check('inline: never open', el.matches(':state(open)'), false)
	el.remove()
})

check('no errors', errors, [])
await report()
`
