import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_track: var(--sb-surface-inset, #0B1224);
	--_edge: var(--sb-border, #283552);
	--_ok: var(--sb-ok, #6EF59A);
	--_warn: var(--sb-warn, #F5C451);
	--_danger: var(--sb-danger, #F2777A);
	--_label: var(--sb-text-2, #AEBBDD);
	--_text: var(--sb-text-1, #F3F4FA);
	display: block;
	inline-size: 100%;
	max-inline-size: var(--sb-meter-width, 22rem);
}
.head { display: flex; justify-content: space-between; gap: 1rem; margin-block-end: 0.4rem; font-size: 0.8125rem; }
.label { color: var(--_label); font-weight: 600; }
.value { color: var(--_text); font-weight: 700; font-variant-numeric: tabular-nums; }
.bar {
	--_c: var(--_ok);
	display: grid;
	grid-auto-flow: column;
	grid-auto-columns: 1fr;
	gap: 3px;
	padding: 3px;
	background: var(--_track);
	box-shadow: 0 0 0 2px var(--_edge);
}
.bar.warn { --_c: var(--_warn); }
.bar.danger { --_c: var(--_danger); }
.seg {
	block-size: var(--sb-meter-height, 12px);
	background: color-mix(in oklch, var(--_edge) 60%, transparent);
	transition: 120ms steps(2, end);
	transition-delay: calc(var(--i) * 18ms);
}
.seg.on { background: var(--_c); box-shadow: inset 0 -3px 0 color-mix(in oklch, var(--_c), black 25%); }
@media (prefers-reduced-motion: reduce) { .seg { transition: none; } }
`

rocket('sb-meter', {
	props: ({ bool, number, string }) => ({
		value: number.docs({ description: 'Current value.' }),
		min: number.docs({ description: 'Minimum.' }),
		max: number.default(100).docs({ description: 'Maximum.' }),
		segments: number.clamp(2, 40).step(1).default(12).docs({ description: 'Number of blocks.' }),
		warn: number.default(70).docs({ description: 'Warning threshold, in value units. Unset, it sits at 70% of the range (70 on 0–100). Below danger means low values are bad (like fuel).' }),
		danger: number.default(90).docs({ description: 'Danger threshold, in value units. Unset, it sits at 90% of the range (90 on 0–100).' }),
		label: string.trim.docs({ description: 'Caption.' }),
		unit: string.docs({ description: 'Unit after the value.' }),
		showValue: bool.default(true).docs({ description: 'Show the value.' }),
		decimals: number.clamp(0, 4).docs({ description: 'Decimals shown.' }),
	}),
	setup: ({ adoptStyles, host }) => adoptStyles(host, styles),
	render: ({ html, host, props: { value, min, max, segments, warn, danger, label, unit, showValue, decimals } }) => {
		const f = Math.max(0, Math.min(1, (value - min) / (max - min || 1)))
		// The nearest block, but any value above min lights one, and only max lights them all.
		const lit = Math.min(segments - (f < 1), Math.max(f > 0, Math.round(f * segments)))
		// The cascade runs from the edge that moves: blocks light up outwards
		// from the last fill and go dark from the far end. Blocks below a
		// rising fill get a negative delay, so a new tone reaches them at once.
		const from = host._lit ?? lit
		host._lit = lit
		// Unset thresholds sit at 70% and 90% of the range (property writes
		// reflect, so el.warn = x counts as set).
		if (!host.hasAttribute('warn')) warn = min + (max - min) * 0.7
		if (!host.hasAttribute('danger')) danger = min + (max - min) * 0.9
		const s = danger < warn ? -1 : 1
		const tone = s * (value - danger) >= 0 ? 'danger' : s * (value - warn) >= 0 ? 'warn' : 'ok'
		const shown = Number(value).toFixed(decimals) + unit
		return html`
			${label || showValue ? html`
				<div class="head">
					<span class="label" part="label">${label}</span>
					${showValue ? html`<span class="value" part="value">${shown}</span>` : null}
				</div>` : null}
			<div class="bar ${tone}" part="bar" role="meter" aria-label="${label || 'Meter'}"
				aria-valuemin="${min}" aria-valuemax="${max}" aria-valuenow="${value}" aria-valuetext="${shown}">
				${Array.from({ length: segments }, (_, i) => html`<span class="seg ${i < lit ? 'on' : ''}" style="--i: ${lit > from ? i - from : from - 1 - i}"></span>`)}
			</div>
		`
	},
})
