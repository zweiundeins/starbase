package catalog

// Category groups components in the sidebar. The set is fixed so the UI can
// give every category an icon and a stable order.
type Category struct {
	Slug  string
	Label string
	Icon  string // name of an icon in internal/ui/icons
}

var Categories = []Category{
	{"forms", "Forms", "forms"},
	{"navigation", "Navigation", "navigation"},
	{"feedback", "Feedback", "feedback"},
	{"layout", "Layout", "layout"},
	{"media", "Media", "media"},
	{"data", "Data", "data"},
	{"utilities", "Utilities", "utilities"},
	{"experimental", "Experimental", "experimental"},
}

func CategoryBySlug(slug string) (Category, bool) {
	for _, c := range Categories {
		if c.Slug == slug {
			return c, true
		}
	}
	return Category{}, false
}
