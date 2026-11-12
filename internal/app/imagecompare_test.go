package app_test

import "testing"

// TestImageCompareExpandedWord: sb-image-compare's expanded keeps "false" as
// the server's word for closed and ignores a removal, on any Datastar build.
// With STARBASE_DATASTAR_BUNDLE=<file>, on that build.
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
check('no attribute: closed', full(), false)
await say('')
check('expanded: open', full(), true)
await say('false')
check('expanded="false": closed', full(), false)
await say(null)
check('a removal is no word: still closed', full(), false)
await say('true')
check('expanded="true": open', full(), true)
await say(null)
check('a removal is no word: still open', full(), true)
el.expanded = false
await settle()
check('the property closes it', full(), false)
el.remove()
await report()
`
