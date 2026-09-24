import { rocket } from 'datastar'
// uqr (MIT), vendored unmodified from npm; see vendor.json.
import { encode } from './vendor/uqr.mjs'

// paths draws the dark modules as horizontal runs: [modules, corners]. With
// accent, the finder squares (uqr type 2, "Position") go to the second path.
const paths = ({ data, types }, accent) => {
	const d = ['', '']
	data.forEach((row, y) => {
		for (let x = 0, s, k; x < row.length; ) {
			if (!row[x]) { x++; continue }
			k = +(accent && types[y][x] == 2)
			for (s = x; row[x] && +(accent && types[y][x] == 2) == k; ) x++
			d[k] += `M${s} ${y}h${x - s}v1h-${x - s}z`
		}
	})
	return d
}

const styles = /* css */ `
:host {
	--_fg: var(--sb-qr-color, #0B1224);
	--_bg: var(--sb-qr-background, #FFFFFF);
	--_eye: var(--sb-qr-accent, var(--sb-brand, #8C6BFF));
	display: inline-block;
	inline-size: 10rem;
	aspect-ratio: 1;
	line-height: 0;
}
svg { inline-size: 100%; block-size: 100%; }
.bg { fill: var(--_bg); }
.mod { fill: var(--_fg); }
.eye { fill: var(--_eye); }
.error { display: grid; place-items: center; inline-size: 100%; block-size: 100%; padding: 0.5rem; box-sizing: border-box; background: var(--_bg); color: #A52A3A; font-size: 0.75rem; line-height: 1.3; text-align: center; }
`

rocket('sb-qr-code', {
	props: ({ bool, number, oneOf, string }) => ({
		value: string.docs({ description: 'Text or URL to encode.' }),
		ecc: oneOf('L', 'M', 'Q', 'H').default('M').docs({ description: 'Error correction: L 7%, M 15%, Q 25%, H 30% of the code can be damaged or covered.' }),
		border: number.round.clamp(0, 8).default(2).docs({ description: 'Quiet zone around the code, in modules (the standard asks for 4; 2 scans fine on a plain background).' }),
		accent: bool.docs({ description: 'Colour the three corner squares with --sb-qr-accent (default: the brand colour).' }),
		label: string.trim.docs({ description: 'Accessible name (default: "QR code: " and the value).' }),
	}),
	renderOnPropChange: false,
	setup: ({ $$, adoptStyles, host, observeProps, props }) => {
		adoptStyles(host, styles)
		const build = () => {
			$$.error = ''
			$$.name = props.label || (props.value ? 'QR code: ' + props.value : 'QR code')
			if (!props.value) {
				$$.n = 0
				$$.mods = $$.eyes = ''
				return
			}
			try {
				const qr = encode(props.value, { ecc: props.ecc, border: props.border })
				$$.n = qr.size
				;[$$.mods, $$.eyes] = paths(qr, props.accent)
			} catch {
				$$.n = 0
				$$.mods = $$.eyes = ''
				$$.error = 'Too much data for a QR code'
			}
		}
		build()
		observeProps(build)
	},
	render: ({ html }) => html`
		<svg part="svg" role="img" shape-rendering="crispEdges"
			data-show="$$n > 0"
			data-attr:aria-label="$$name"
			data-effect="el.setAttribute('viewBox', '0 0 ' + $$n + ' ' + $$n)">
			<rect class="bg" part="background" width="100%" height="100%"></rect>
			<path class="mod" part="modules" data-attr:d="$$mods"></path>
			<path class="eye" part="corners" data-attr:d="$$eyes || null"></path>
		</svg>
		<div class="error" role="alert" data-show="$$error" data-text="$$error"></div>
	`,
})
