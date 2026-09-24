package model

// SessionPrefs are preferences that outlive a page: they carry across
// navigations and browser tabs of one session (session_prefs, keyed by sid,
// set by commands). TabState, by contrast, lives for one loaded page. A new
// preference is one field here plus its command (commands/prefs.go).
type SessionPrefs struct {
	// InstallTab is the component pages' installation tab (InstallTabs).
	InstallTab string `json:"installTab,omitempty"`
}

// InstallTabs are the installation tabs, in display order, by stable key:
// stored choices stay valid when the tabs are reordered or relabelled.
var InstallTabs = []struct{ Key, Label string }{
	{"autoloader", "Autoloader"},
	{"component", "This component"},
	{"pinned", "Pinned"},
	{"self-host", "Self-host"},
}

// ValidInstallTab reports whether key names an installation tab.
func ValidInstallTab(key string) bool {
	return InstallTabIndex(key) >= 0
}

// InstallTabIndex is the position of key in InstallTabs, or -1.
func InstallTabIndex(key string) int {
	for i, t := range InstallTabs {
		if t.Key == key {
			return i
		}
	}
	return -1
}

// InstallTabOrDefault is the stored tab, or the first (the autoloader).
func (p SessionPrefs) InstallTabOrDefault() string {
	if ValidInstallTab(p.InstallTab) {
		return p.InstallTab
	}
	return InstallTabs[0].Key
}
