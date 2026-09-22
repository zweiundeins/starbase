package submission

import (
	"fmt"
	"slices"
	"strings"

	"starbase/internal/catalog"
)

// Similar lists catalog components that look like c, most similar first,
// so reviewers can spot a component the gallery already has. It scores
// shared tags (2 each), the same category (1) and shared summary words (1
// each), and keeps the top 3 scoring at least 3.
func Similar(cat *catalog.Catalog, c *catalog.Component) []*catalog.Component {
	type scored struct {
		c     *catalog.Component
		score int
	}
	var out []scored
	tags, words := set(c.Tags), set(summaryWords(c.Summary))
	for _, o := range cat.Components {
		if o.Slug == c.Slug {
			continue
		}
		score := 0
		for _, t := range o.Tags {
			if tags[strings.ToLower(t)] {
				score += 2
			}
		}
		if o.Category == c.Category {
			score++
		}
		for _, w := range summaryWords(o.Summary) {
			if words[w] {
				score++
			}
		}
		if score >= 3 {
			out = append(out, scored{o, score})
		}
	}
	slices.SortStableFunc(out, func(a, b scored) int { return b.score - a.score })
	var top []*catalog.Component
	for _, s := range out[:min(3, len(out))] {
		top = append(top, s.c)
	}
	return top
}

// SimilarMarkdown renders Similar for a pull request body; site is the
// public Starbase URL (links are omitted when empty).
func SimilarMarkdown(list []*catalog.Component, site string) string {
	if len(list) == 0 {
		return "No similar components in the catalog."
	}
	var b strings.Builder
	for _, c := range list {
		name := "`<" + c.Tag + ">` " + c.Name
		if site != "" {
			name = fmt.Sprintf("[%s](%s/components/%s)", name, strings.TrimSuffix(site, "/"), c.Slug)
		}
		fmt.Fprintf(&b, "- %s (%s; %s): %s\n", name, c.Category, strings.Join(c.Tags, ", "), c.Summary)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

var stopWords = set([]string{"with", "that", "this", "from", "your", "into", "attribute", "attributes", "component", "components"})

func summaryWords(s string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') }) {
		if len(w) > 3 && !stopWords[w] {
			out = append(out, w)
		}
	}
	return out
}

func set(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[strings.ToLower(x)] = true
	}
	return m
}
