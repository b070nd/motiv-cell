package lattice

import (
	"errors"
	"slices"
	"testing"

	"github.com/b070nd/motiv-cell/src/cell"
)

func TestGraphOrderAndValidation(t *testing.T) {
	g, err := NewGraph(3, []Edge{{From: 2, To: 1, Link: 7}, {From: 0, To: 1, Link: 9}, {From: 0, To: 1, Link: 3}, {From: 1, To: 0}})
	if err != nil {
		t.Fatal(err)
	}
	want := []Edge{{From: 0, To: 1, Link: 3}, {From: 0, To: 1, Link: 9}, {From: 2, To: 1, Link: 7}}
	if got := g.Inbound(1, nil); !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := g.Inbound(2, nil); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if got := g.Inbound(3, nil); len(got) != 0 {
		t.Fatalf("out of range: got %v", got)
	}
	if _, err := NewGraph(2, []Edge{{From: 0, To: 2}}); !errors.Is(err, ErrEndpoint) {
		t.Fatalf("got %v", err)
	}
	if _, err := NewGraph(2, []Edge{{From: 0, To: 1}, {From: 0, To: 1}}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("got %v", err)
	}
}

func TestLine(t *testing.T) {
	l, err := NewLine(5, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	froms := func(to cell.ID) []cell.ID {
		var out []cell.ID
		for _, e := range l.Inbound(to, nil) {
			if e.To != to || e.Link != 4 {
				t.Fatalf("bad edge %v", e)
			}
			out = append(out, e.From)
		}
		return out
	}
	if got := froms(0); !slices.Equal(got, []cell.ID{1, 2}) {
		t.Fatalf("got %v", got)
	}
	if got := froms(2); !slices.Equal(got, []cell.ID{0, 1, 3, 4}) {
		t.Fatalf("got %v", got)
	}
	if got := froms(4); !slices.Equal(got, []cell.ID{2, 3}) {
		t.Fatalf("got %v", got)
	}
}

func TestLayers(t *testing.T) {
	line, err := NewLine(3, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := NewGraph(2, []Edge{{From: 0, To: 1, Link: 2}})
	if err != nil {
		t.Fatal(err)
	}
	layers, err := NewLayers([]Topology{line, graph}, []Edge{{From: 1, To: 4, Link: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if layers.Size() != 5 || layers.Count() != 2 {
		t.Fatalf("size %d count %d", layers.Size(), layers.Count())
	}
	if base, err := layers.Base(1); err != nil || base != 3 {
		t.Fatalf("base %d %v", base, err)
	}
	want := []Edge{{From: 3, To: 4, Link: 2}, {From: 1, To: 4, Link: 3}}
	if got := layers.Inbound(4, nil); !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	want = []Edge{{From: 0, To: 1, Link: 1}, {From: 2, To: 1, Link: 1}}
	if got := layers.Inbound(1, nil); !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
