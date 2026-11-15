package lattice

import (
	"slices"

	"github.com/b070nd/motiv-cell/src/cell"
)

type Graph struct {
	offsets []int
	edges   []Edge
}

func NewGraph(size int, edges []Edge) (*Graph, error) {
	if size < 0 || size > MaxSize {
		return nil, ErrSize
	}
	sorted := slices.Clone(edges)
	for _, edge := range sorted {
		if !within(edge.From, size) || !within(edge.To, size) {
			return nil, ErrEndpoint
		}
	}
	slices.SortFunc(sorted, compare)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			return nil, ErrDuplicate
		}
	}
	offsets := make([]int, size+1)
	for _, edge := range sorted {
		offsets[int(edge.To)+1]++
	}
	for i := 0; i < size; i++ {
		offsets[i+1] += offsets[i]
	}
	return &Graph{offsets: offsets, edges: sorted}, nil
}

func (g *Graph) Size() int {
	if g == nil || len(g.offsets) == 0 {
		return 0
	}
	return len(g.offsets) - 1
}

func (g *Graph) Inbound(to cell.ID, buf []Edge) []Edge {
	if !within(to, g.Size()) {
		return buf
	}
	at := int(to)
	return append(buf, g.edges[g.offsets[at]:g.offsets[at+1]]...)
}
