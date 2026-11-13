package app_test

import "testing"

// TestImageCompareExpandedWord: sb-image-compare's expanded is a word by
// presence, as open is on sb-modal: added opens, removed closes, any value is
// present, and identical markup keeps the reader's state, on any Datastar
// build. With STARBASE_DATASTAR_BUNDLE=<file>, on that build.
func TestImageCompareExpandedWord(t *testing.T) {
	copyRows(t, probeBundle(t, "/", pdPrelude+imageCompareJS, nil))
}

const imageCompareJS = `
await customElements.whenDefined('sb-image-compare')
const el = document.createElement('sb-image-compare')
el.innerHTML = '<img slot="before" alt="a" src="/og.png"><img slot="after" alt="b" src="/og.png">'
document.body.prepend(el)
await settle()
const full = () => el.matches(':state(expanded)')
const say = async (v) => (v == null ? el.removeAttribute('expanded') : el.setAttribute('expanded', v), await settle())
const frame = el.shadowRoot.querySelector('.frame')
check('no attribute: closed', full(), false)
await say('')
check('expanded: open', full(), true)
await say('false')
check('expanded="false" is present: still open', full(), true)
await say(null)
check('a removal closes', full(), false)
await say('true')
check('expanded="true": open', full(), true)
frame.hidePopover()
await settle()
check('the reader closes it', full(), false)
await say('true')
check('identical markup keeps the reader\'s state', full(), false)
await say(null)
await say('')
check('a new word opens again', full(), true)
el.expanded = false
await settle()
check('the property closes it', full(), false)
el.remove()
await report()
`
