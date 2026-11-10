// sbCommand(el, key, send) sends a command with send(), a function around the
// page's @post. A command that meets 409 (its tab's stream is not open) stays
// pending and goes out again once the stream connects: the newest per key,
// at most once per connect.
;(() => {
	const stream = document.currentScript.dataset.stream
	const last = new Map() // key → { send, epoch: the connect it went out under, -1 before one }
	const keyOf = new WeakMap() // element → key of its newest command
	const waiting = new Set() // keys whose newest command met a 409
	let epoch = 0
	let open = false
	const fire = (key) => {
		const c = last.get(key)
		c.epoch = open ? epoch : -1
		waiting.delete(key)
		c.send()?.catch?.(() => {})
	}
	window.sbCommand = (el, key, send) => {
		keyOf.set(el, key)
		last.set(key, { send })
		fire(key)
	}
	document.addEventListener('datastar-fetch', ({ detail: { el, type, argsRaw } }) => {
		if (el?.id === stream) {
			if (!type.startsWith('datastar-patch')) open = false
			else if (!open) {
				open = true
				epoch++
				for (const key of [...waiting]) fire(key)
			}
			return
		}
		const key = el && keyOf.get(el)
		if (key === undefined || type !== 'error' || argsRaw?.status !== '409') return
		// Sent before this connect: once more now. Otherwise at the next connect.
		if (open && last.get(key).epoch < epoch) fire(key)
		else waiting.add(key)
	})
	// Datastar closes a hidden tab's stream and opens it again without an event.
	document.addEventListener('visibilitychange', () => document.hidden && (open = false))
})()
