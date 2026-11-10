package web

import (
	"bytes"
	"context"
	"database/sql"

	"github.com/a-h/templ"
	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/sse"

	"starbase/internal/queries"
	"starbase/internal/ui"
)

// A region is a part of a page that a view of its own renders once for
// all the page's viewers, such as the Showcase's board: a commit that
// touches only the region's topics renders the region alone. A document
// renders the region in place. The tab's view leaves a hole there, which
// the stream fills with the region's newest frame, so that the morph of
// #app finds the region as the page shows it and leaves it alone.
type region struct {
	view  live.View
	html  func(ctx context.Context, rd *queries.Reader) ([]byte, error)
	hole  []byte // the data line that stands for the region in the tab's frames
	shell []byte // data lines of the region's elements without content, for a hole before the region's first frame
}

// newRegion makes a region from the key, topics and presence of v and
// html. Its frame is a patch of the elements html renders, by their ids;
// shell is the same elements, empty.
func newRegion(v live.View, shell string, html func(context.Context, *queries.Reader) ([]byte, error)) *region {
	v.Render = func(ctx context.Context, tx *sql.Tx) (any, error) {
		b, err := html(ctx, queries.Read(tx))
		if err != nil {
			return nil, err
		}
		return sse.PatchElements(b), nil
	}
	return &region{
		view:  v,
		html:  html,
		hole:  []byte("data: elements " + holeMarker(v.Key) + "\n"),
		shell: dataLines(sse.PatchElements([]byte(shell))),
	}
}

func holeMarker(key string) string { return "<!--region:" + key + "-->" }

// region is r's place in the page: the region itself in a document, and
// in the tab's stream a hole on a line of its own, which the stream
// replaces with the region's data lines. The region gets the same newlines
// around it in a document, so both give the page the same markup.
func (rc *renderCtx) region(r *region) (templ.Component, error) {
	if rc.stream {
		return templ.Raw("\n" + holeMarker(r.view.Key) + "\n"), nil
	}
	html, err := r.html(rc.ctx, rc.r)
	if err != nil {
		return nil, err
	}
	return templ.Raw("\n" + string(html) + "\n"), nil
}

// fillHoles is the WriteTick of a tab's stream (without regions, it writes
// the frames as they are). On a page with regions, each frame of the page (the view with key page) goes out with its holes
// filled by the newest frame of each region on this stream, or the
// region's shell before its first. A region's frame goes out as well, also
// after a page frame that carries it, where its morph changes nothing.
func fillHoles(page string, regions []*region) func(*sse.Stream, []live.Frame) error {
	latest := make([][]byte, len(regions))
	var out [][]byte
	return func(st *sse.Stream, frames []live.Frame) error {
		for _, f := range frames {
			for i, r := range regions {
				if f.Key == r.view.Key {
					latest[i] = dataLines(f.Value.([]byte))
				}
			}
		}
		for _, f := range frames {
			ev := f.Value.([]byte)
			if f.Key != page {
				out = append(out, ev)
				continue
			}
			for i, r := range regions {
				head, tail, ok := bytes.Cut(ev, r.hole)
				if !ok {
					continue
				}
				fill := latest[i]
				if fill == nil {
					fill = r.shell
				}
				out = append(out, head, fill)
				ev = tail
			}
			out = append(out, ev)
		}
		err := st.Write(out...)
		clear(out)
		out = out[:0]
		return err
	}
}

// watchdogReset sets the page's watchdog counter (ui.QuietSignal) to zero.
var watchdogReset = sse.PatchSignals([]byte(`{"`+ui.QuietSignal+`":0}`), false)

// resetWatchdog sends watchdogReset after the stream's first frames: after an
// outage the counter may be one tick short of firing, and lith's first ping
// comes a heartbeat after the stream opens.
func resetWatchdog(write func(*sse.Stream, []live.Frame) error) func(*sse.Stream, []live.Frame) error {
	first := true
	return func(st *sse.Stream, frames []live.Frame) error {
		if first {
			first = false
			frames = append(frames[:len(frames):len(frames)], live.Frame{Value: watchdogReset})
		}
		return write(st, frames)
	}
}

const elementsEvent = "event: datastar-patch-elements\n"

// dataLines are the data lines of ev, a patch of elements by their ids
// (sse.PatchElements without a Patch): ev without its event line and the
// blank line that ends it.
func dataLines(ev []byte) []byte {
	lines, _ := bytes.CutPrefix(ev, []byte(elementsEvent))
	lines, _ = bytes.CutSuffix(lines, []byte("\n"))
	return lines
}
