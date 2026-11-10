// sbCommand(el, key, send) sends a command as an op: send(op) is the page's
// @post with the op id in its Op-Id header, and the tab's row acks the newest
// op of each key, applied or refused, in #ops (TabState.Ops). The op ends on
// a signal, never on a timer:
//   - a frame acks it: el.revert() when el is still :state(pending), since the
//     server didn't follow it (a refusal, or a value normalized back);
//   - its request fails (an error status other than 409, or retries-failed),
//     or the stream connects again without its ack and with no request of its
//     key under way: el.revert(), and sb-op-failed {key, reason} on el.
// A 409 (the tab's stream is not open) keeps it pending: it goes out again
// once the stream connects, at most once per connect.
;(() => {
	const { stream, ops: opsID } = document.currentScript.dataset
	const ops = new Map() // key → { op, el, send, epoch: the connect it went out under, waiting: met 409 }
	const inflight = new Map() // key → its requests under way
	const keyOf = new WeakMap() // element → key of its newest op
	let epoch = 0
	let open = false
	let connected = false // the stream connected, and its first frame of the tab is still to come
	const fire = (key) => {
		const o = ops.get(key)
		o.epoch = open ? epoch : -1
		o.waiting = false
		o.send(o.op)?.catch?.(() => {})
	}
	const pending = (el) => {
		try {
			return el.matches(':state(pending)')
		} catch {
			return false
		}
	}
	const end = (key, reason) => {
		const { el } = ops.get(key)
		ops.delete(key)
		if (reason === 'acked') return void (pending(el) && el.revert?.())
		el.revert?.()
		el.dispatchEvent(new CustomEvent('sb-op-failed', { detail: { key, reason } }))
	}
	// After the morph of a frame of the tab, which carries #ops.
	const frame = () => {
		let acks = {}
		try {
			acks = JSON.parse(document.getElementById(opsID)?.dataset.acks || '{}')
		} catch {}
		for (const [key, o] of ops) if (acks[key] === o.op) end(key, 'acked')
		if (!connected) return
		connected = false
		for (const [key, o] of ops) {
			if (o.waiting) fire(key)
			else if (!inflight.get(key)) end(key, 'lost')
		}
	}
	window.sbCommand = (el, key, send) => {
		keyOf.set(el, key)
		ops.set(key, { op: Date.now().toString(36) + Math.random().toString(36).slice(2, 7), el, send })
		fire(key)
	}
	document.addEventListener('datastar-fetch', ({ detail: { el, type, argsRaw } }) => {
		if (el?.id === stream) {
			if (!type.startsWith('datastar-patch')) return void (open = false)
			if (!open) (open = true), (connected = true), epoch++
			if (type === 'datastar-patch-elements' && argsRaw?.elements?.includes(`id="${opsID}"`)) setTimeout(frame)
			return
		}
		const key = el && keyOf.get(el)
		if (key === undefined) return
		if (type === 'started') inflight.set(key, (inflight.get(key) ?? 0) + 1)
		if (type === 'finished') inflight.set(key, Math.max(0, (inflight.get(key) ?? 1) - 1))
		const o = ops.get(key)
		if (!o || o.waiting) return
		if (type === 'error' && argsRaw?.status === '409') {
			// Sent before this connect: once more now. Otherwise at the next one.
			if (open && o.epoch < epoch) fire(key)
			else o.waiting = true
		} else if (type === 'error' || type === 'retries-failed') end(key, type)
	})
	// Datastar closes a hidden tab's stream and opens it again without an event.
	document.addEventListener('visibilitychange', () => document.hidden && (open = false))
})()
