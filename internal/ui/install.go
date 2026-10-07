package ui

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"starbase/internal/catalog"
	"starbase/internal/model"
)

// Snippet is a copyable code block: Raw is what the copy button copies,
// HTML its highlighted form.
type Snippet struct {
	Raw, HTML string
}

// InstallView is a component page's Installation section: one sb-tabs
// panel per way of loading it (model.InstallTabs, in that order).
type InstallView struct {
	Tab              string   // the selected tab's key (a session preference)
	Autoloader       Snippet  // import map + the latest autoloader
	Component        Snippet  // import map + this version's one-file bundle and its dependencies'
	ComponentModules Snippet  // the same with the minified module files
	ModuleFiles      int      // how many module files ComponentModules loads
	Deps             []string // tags of the components it renders (transitively)
	Pinned           Snippet  // today's catalog snapshot, with integrity; Raw is empty before it is stored
	ImportMap        string   // URL of the snapshot's full import map (integrity for every file)
	SelfHost         Snippet  // import map at your own Datastar, and your copies of the bundles
	SelfHostModules  Snippet  // the same with copies of the module files
	Files            []SelfHostGroup
	Datastar         string // where to get datastar-rocket.js
	Patches          string // the fixes in that build (patches/rocket on GitHub)
}

// SelfHostGroup is one component's files to copy (the component, then each
// component it renders).
type SelfHostGroup struct {
	Tag    string
	Bundle []SelfHostFile // the one-file bundle, then what it loads lazily (no readable URL)
	Files  []SelfHostFile // the module files
}

// SelfHostFile is one file: its versioned minified and readable URLs.
type SelfHostFile struct {
	Name, Min, Readable string
	Size                catalog.Size
}

var nonSlug = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// tabSlot is the slot sb-tabs gives a label's panel (its slug() in tabs.js):
// lowercased, every run of anything but letters and digits a hyphen. (sb-tabs
// appends the index to a slot an earlier label already gave; the install
// labels give distinct slots.)
func tabSlot(label string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(label), "-"), "-")
}

func installLabels() string {
	labels := make([]string, len(model.InstallTabs))
	for i, t := range model.InstallTabs {
		labels[i] = t.Label
	}
	b, _ := json.Marshal(labels)
	return string(b)
}

func installSelected(key string) string {
	return strconv.Itoa(max(0, model.InstallTabIndex(key)))
}

// installChange posts the chosen tab by its key, not its index.
func installChange() string {
	keys := make([]string, len(model.InstallTabs))
	for i, t := range model.InstallTabs {
		keys[i] = t.Key
	}
	b, _ := json.Marshal(keys)
	return "evt.target === el && @post('/cmd/install-tab', {payload: {tab: " + string(b) + "[evt.detail.value]}})"
}

func installSlot(key string) string {
	return tabSlot(model.InstallTabs[model.InstallTabIndex(key)].Label)
}

// depSep follows the i-th of n dependency tags in "It renders …".
func depSep(i, n int) string {
	switch {
	case i < n-1:
		return ", "
	case n == 1:
		return ", so that is pinned here too."
	default:
		return ", so those are pinned here too."
	}
}

// moduleFiles counts module files: "one file", "12 files".
func moduleFiles(n int) string {
	if n == 1 {
		return "one file"
	}
	return strconv.Itoa(n) + " files"
}

// bundleLink and bundleNote describe the i-th file of a one-file bundle:
// the bundle, then what it loads lazily.
func bundleLink(i int) string {
	if i == 0 {
		return "one file"
	}
	return "minified"
}

func bundleNote(i int, s catalog.Size) string {
	if i == 0 {
		return FmtBytes(s.Min) + " brotli"
	}
	return FmtBytes(s.Min) + " brotli, loaded on first use"
}
