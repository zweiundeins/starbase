package web

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/sse"

	"starbase/internal/queries"
)

// A stream fills the hole in each frame of its tab's view with the
// region's newest frame it has, or with the region's shell before the
// first, so that the frame is the page as if the region were rendered in
// place.
func TestFillHoles(t *testing.T) {
	r := newRegion(live.View{Key: "board"}, `<div id="b"></div>`, func(context.Context, *queries.Reader) ([]byte, error) { return nil, nil })
	page := func(region string) []byte {
		return sse.PatchElements([]byte("<div id=\"app\"><h1>Board</h1>\n" + region + "\n<p>after</p></div>"))
	}
	board := func(cells string) []byte { return sse.PatchElements([]byte("<div id=\"b\">\n" + cells + "</div>")) }
	tab := page(holeMarker("board"))

	rec := httptest.NewRecorder()
	st, err := sse.Open(rec, httptest.NewRequest("POST", "/", nil), sse.Options{})
	if err != nil {
		t.Fatal(err)
	}
	write := fillHoles("tab:1", []*region{r})
	var want strings.Builder
	for _, step := range []struct {
		frames []live.Frame
		want   [][]byte
	}{
		{[]live.Frame{{Key: "tab:1", Value: tab}}, [][]byte{page(`<div id="b"></div>`)}},
		{[]live.Frame{{Key: "board", Value: board("01")}}, [][]byte{board("01")}},
		{[]live.Frame{{Key: "tab:1", Value: tab}}, [][]byte{page("<div id=\"b\">\n01</div>")}},
		{[]live.Frame{{Key: "tab:1", Value: tab}, {Key: "board", Value: board("02")}}, [][]byte{page("<div id=\"b\">\n02</div>"), board("02")}},
	} {
		if err := write(st, step.frames); err != nil {
			t.Fatal(err)
		}
		for _, e := range step.want {
			want.Write(e)
		}
	}
	st.Close()
	if got := rec.Body.String(); got != want.String() {
		t.Errorf("the stream sent\n%s\nwant\n%s", got, want.String())
	}
}
