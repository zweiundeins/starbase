package ui

import (
	"encoding/json"
	"sort"
	"strings"

	"starbase/internal/catalog"
	"starbase/internal/model"
)

func sortStrings(s []string) { sort.Strings(s) }

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func modelBrowseCategory(cat string) model.Browse { return model.Browse{Category: cat} }

func propType(p catalog.ManifestProp) string {
	if p.Type == "oneOf" && len(p.Values) > 0 {
		parts := make([]string, len(p.Values))
		for i, v := range p.Values {
			b, _ := json.Marshal(v)
			parts[i] = string(b)
		}
		return strings.Join(parts, " | ")
	}
	return p.Type
}

func propDefault(p catalog.ManifestProp) string {
	if len(p.Default) == 0 || string(p.Default) == "null" {
		return "—"
	}
	return string(p.Default)
}

func propDoc(p catalog.ManifestProp) string {
	if p.Docs != nil {
		return p.Docs.Description
	}
	return ""
}
