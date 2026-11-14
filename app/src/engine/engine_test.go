package engine

import (
	"errors"
	"slices"
	"testing"

	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/lattice"
)

const errBoom Error = "boom"

func rule110(ctx cell.Context, self uint8, neighbors []cell.Neighbor[uint8], _ []struct{}) (uint8, error) {
	var left, right uint8
	for _, n := range neighbors {
		if n.ID < ctx.Self {
			left = n.State
		} else {
			right = n.State
		}
	}
	return (110 >> (left<<2 | self<<1 | right)) & 1, nil
}

func sum(_ cell.Context, self int64, neighbors []cell.Neighbor[int64], events []int64) (int64, error) {
	for _, n := range neighbors {
		self += n.State
	}
	for _, e := range events {
		self += e
	}
	return self, nil
}

func row(t *testing.T, e *Engine[uint8, struct{}]) string {
	t.Helper()
	out := make([]byte, 0, e.Size())
	for _, s := range e.States(nil) {
		out = append(out, '0'+s)
	}
	return string(out)
}

func TestRule110(t *testing.T) {
	line, err := lattice.NewLine(8, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(Config[uint8, struct{}]{
		Topology: line,
		Rule:     cell.RuleFunc[uint8, struct{}](rule110),
		Schedule: Synchronous,
		Initial:  []uint8{0, 0, 0, 0, 0, 0, 0, 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00000011", "00000111", "00001101", "00011111", "00110001", "01110011", "11010111"}
	for i, w := range want {
		if err := e.Step(Batch[struct{}]{}); err != nil {
			t.Fatal(err)
		}
		if got := row(t, e); got != w {
			t.Fatalf("tick %d: got %s want %s", i+1, got, w)
		}
	}
	if e.Ticks() != uint64(len(want)) {
		t.Fatalf("ticks: got %d", e.Ticks())
	}
}

func TestSchedules(t *testing.T) {
	graph, err := lattice.NewGraph(3, []lattice.Edge{{From: 0, To: 1}, {From: 1, To: 2}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		schedule Schedule
		want     []int64
	}{
		{Synchronous, []int64{1, 1, 0}},
		{Sequential, []int64{1, 1, 1}},
	} {
		e, err := New(Config[int64, int64]{
			Topology: graph,
			Rule:     cell.RuleFunc[int64, int64](sum),
			Schedule: tc.schedule,
			Initial:  []int64{1, 0, 0},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Step(Batch[int64]{}); err != nil {
			t.Fatal(err)
		}
		if got := e.States(nil); !slices.Equal(got, tc.want) {
			t.Fatalf("schedule %d: got %v want %v", tc.schedule, got, tc.want)
		}
	}
}

func TestFeeds(t *testing.T) {
	items := []Item[int64]{
		{At: 30, To: 1, Event: 5},
		{At: 10, To: 0, Event: 1},
		{At: 30, To: 0, Event: 7},
		{At: 10, To: 0, Event: 2},
	}
	record := func(source Source[int64]) ([]int64, [][]int64) {
		graph, err := lattice.NewGraph(2, nil)
		if err != nil {
			t.Fatal(err)
		}
		var ats []int64
		var seen [][]int64
		rule := func(ctx cell.Context, self int64, _ []cell.Neighbor[int64], events []int64) (int64, error) {
			if ctx.Self == 0 {
				ats = append(ats, ctx.Tick.At)
				seen = append(seen, slices.Clone(events))
			}
			return self, nil
		}
		e, err := New(Config[int64, int64]{
			Topology: graph,
			Rule:     cell.RuleFunc[int64, int64](rule),
			Schedule: Synchronous,
			Initial:  []int64{0, 0},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Run(source); err != nil {
			t.Fatal(err)
		}
		return ats, seen
	}

	ats, seen := record(NewTimestamps(items))
	if !slices.Equal(ats, []int64{10, 30}) {
		t.Fatalf("timestamps: ticks at %v", ats)
	}
	if !slices.Equal(seen[0], []int64{1, 2}) || !slices.Equal(seen[1], []int64{7}) {
		t.Fatalf("timestamps: events %v", seen)
	}

	positions := []Item[int64]{{At: 2, To: 0, Event: 9}, {At: 0, To: 0, Event: 4}}
	feed, err := NewDiscourse(positions, 4)
	if err != nil {
		t.Fatal(err)
	}
	ats, seen = record(feed)
	if !slices.Equal(ats, []int64{0, 1, 2, 3}) {
		t.Fatalf("discourse: ticks at %v", ats)
	}
	if !slices.Equal(seen[0], []int64{4}) || len(seen[1]) != 0 || !slices.Equal(seen[2], []int64{9}) || len(seen[3]) != 0 {
		t.Fatalf("discourse: events %v", seen)
	}
	if _, err := NewDiscourse(positions, 2); !errors.Is(err, ErrPosition) {
		t.Fatalf("discourse bounds: got %v", err)
	}
}

func TestStepIsAtomic(t *testing.T) {
	graph, err := lattice.NewGraph(3, nil)
	if err != nil {
		t.Fatal(err)
	}
	fail := true
	rule := func(ctx cell.Context, self int64, _ []cell.Neighbor[int64], _ []int64) (int64, error) {
		if fail && ctx.Self == 2 {
			return 0, errBoom
		}
		return self + 1, nil
	}
	for _, schedule := range []Schedule{Synchronous, Sequential} {
		fail = true
		e, err := New(Config[int64, int64]{
			Topology: graph,
			Rule:     cell.RuleFunc[int64, int64](rule),
			Schedule: schedule,
			Initial:  []int64{0, 0, 0},
		})
		if err != nil {
			t.Fatal(err)
		}
		err = e.Step(Batch[int64]{})
		var stepErr *StepError
		if !errors.As(err, &stepErr) || stepErr.Cell != 2 || !errors.Is(err, errBoom) {
			t.Fatalf("schedule %d: got %v", schedule, err)
		}
		if got := e.States(nil); !slices.Equal(got, []int64{0, 0, 0}) || e.Ticks() != 0 {
			t.Fatalf("schedule %d: state changed after failure: %v ticks %d", schedule, got, e.Ticks())
		}
		fail = false
		if err := e.Step(Batch[int64]{}); err != nil {
			t.Fatal(err)
		}
		if got := e.States(nil); !slices.Equal(got, []int64{1, 1, 1}) {
			t.Fatalf("schedule %d: got %v", schedule, got)
		}
		if err := e.Step(Batch[int64]{Deliveries: []Delivery[int64]{{To: 3}}}); !errors.Is(err, ErrDelivery) {
			t.Fatalf("schedule %d: delivery bounds: got %v", schedule, err)
		}
	}
}
