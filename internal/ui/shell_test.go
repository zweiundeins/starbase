package ui

import (
	"context"
	"strings"
	"testing"
)

type artAssets struct {
	Assets
	svg string
}

func (a artAssets) ArtSVG(string) string { return a.svg }

// Art keeps its markup per illustration and classes, and makes it again
// when the illustration it was made from changes.
func TestArtFollowsItsSource(t *testing.T) {
	render := func(svg string) string {
		var b strings.Builder
		if err := Art(artAssets{svg: svg}, "test-art", "a b").Render(context.Background(), &b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	one := `<svg xmlns="http://www.w3.org/2000/svg" class="px-art" width="1"></svg>`
	want := `<svg xmlns="http://www.w3.org/2000/svg" class="px-art a b" aria-hidden="true" focusable="false" preserveAspectRatio="xMidYMax meet" width="1"></svg>`
	if got := render(one); got != want {
		t.Fatalf("Art = %s", got)
	}
	if got := render(one); got != want {
		t.Fatalf("Art from its cache = %s", got)
	}
	if got := render(strings.Replace(one, `"1"`, `"2"`, 1)); got != strings.Replace(want, `"1"`, `"2"`, 1) {
		t.Errorf("Art should follow a changed source: %s", got)
	}
}
