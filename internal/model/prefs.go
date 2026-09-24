package model

// SessionPrefs are preferences that outlive a page: they carry across
// navigations and browser tabs of one session (session_prefs, keyed by sid,
// set by commands). TabState, by contrast, lives for one loaded page. A new
// preference is one field here plus its command (commands/prefs.go).
type SessionPrefs struct {
	// InstallTab is the component pages' installation tab (InstallTabs).
	InstallTab string `json:"installTab,omitempty"`
	// PreviewTheme is the token set the Themes page previews (PreviewThemes).
	PreviewTheme string `json:"previewTheme,omitempty"`
	// PreviewSmooth shows the Themes page previews without 8-bit details.
	PreviewSmooth bool `json:"smooth,omitempty"`
	// GallerySort is the gallery's default sort, used when the URL has none.
	GallerySort Sort `json:"sort,omitempty"`
}

// PreviewThemeOrDefault is the stored preview theme, or Deep Space.
func (p SessionPrefs) PreviewThemeOrDefault() string {
	if ValidPreviewTheme(p.PreviewTheme) {
		return p.PreviewTheme
	}
	return "deep-space"
}

// DefaultSort is the gallery's sort when the URL names none: the session's
// last choice, or the most popular first.
func (p SessionPrefs) DefaultSort() Sort {
	if p.GallerySort.Valid() {
		return p.GallerySort
	}
	return SortPopular
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
