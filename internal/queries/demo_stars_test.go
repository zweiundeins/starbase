package queries_test

import (
	"context"
	"testing"

	"starbase/internal/commands"
	"starbase/internal/demo"
	"starbase/internal/queries"
)

func TestDemoStars(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for range 2 { // idempotent, and next to the bodies
		if err := e.bus.Exec(ctx, commands.SeedStars{}); err != nil {
			t.Fatal(err)
		}
		if err := e.bus.Exec(ctx, commands.SeedDemo{}); err != nil {
			t.Fatal(err)
		}
	}
	window := func(key string, desc bool, offset, count int) ([]demo.Star, int) {
		var stars []demo.Star
		var total int
		if err := e.q.View(ctx, func(r *queries.Reader) (err error) {
			stars, total, err = r.DemoStars(ctx, key, desc, offset, count)
			return
		}); err != nil {
			t.Fatal(err)
		}
		return stars, total
	}
	first, total := window("", false, 0, 3)
	if total != demo.StarCount || len(first) != 3 || first[0].ID != 1 || first[2].ID != 3 {
		t.Fatalf("catalog order: %d stars, %+v", total, first)
	}
	if last, _ := window("", false, demo.StarCount-2, 10); len(last) != 2 || last[1].ID != demo.StarCount {
		t.Errorf("the last window = %+v", last)
	}
	// Two windows of one order fit together, both ways.
	for _, desc := range []bool{false, true} {
		a, _ := window("distance", desc, 1000, 50)
		b, _ := window("distance", desc, 1049, 2)
		if b[0] != a[49] {
			t.Errorf("desc=%v: windows don't overlap: %+v vs %+v", desc, a[49], b[0])
		}
		for i := 1; i < len(a); i++ {
			if (a[i].Distance < a[i-1].Distance) != desc && a[i].Distance != a[i-1].Distance {
				t.Fatalf("desc=%v: out of order at %d: %v then %v", desc, i, a[i-1].Distance, a[i].Distance)
			}
		}
	}
	// The class sorts by temperature: descending, the hottest first.
	if hot, _ := window("class", true, 0, 1); hot[0].Class[0] != 'O' {
		t.Errorf("the hottest star is %s", hot[0].Class)
	}
	if !queries.DemoStarSortable("name") || queries.DemoStarSortable("planets") || queries.DemoStarSortable("id; DROP TABLE demo_stars") {
		t.Error("DemoStarSortable")
	}
}
