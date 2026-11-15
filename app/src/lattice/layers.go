package lattice

import (
	"slices"
	"sort"

	"github.com/b070nd/motiv-cell/src/cell"
)

type Layers struct {
	layers []Topology
	bases  []int
	links  *Graph
}

func NewLayers(layers []Topology, links []Edge) (*Layers, error) {
	if len(layers) == 0 {
		return nil, ErrLayers
	}
	bases := make([]int, len(layers)+1)
	for i, layer := range layers {
		if layer == nil {
			return nil, ErrLayers
		}
		size := layer.Size()
		if size < 0 || size > MaxSize-bases[i] {
			return nil, ErrSize
		}
		bases[i+1] = bases[i] + size
	}
	graph, err := NewGraph(bases[len(layers)], links)
	if err != nil {
		return nil, err
	}
	return &Layers{layers: slices.Clone(layers), bases: bases, links: graph}, nil
}

func (l *Layers) Size() int {
	if l == nil || len(l.bases) == 0 {
		return 0
	}
	return l.bases[len(l.layers)]
}

func (l *Layers) Count() int {
	if l == nil {
		return 0
	}
	return len(l.layers)
}

func (l *Layers) Base(layer int) (cell.ID, error) {
	if layer < 0 || layer >= l.Count() {
		return 0, ErrLayer
	}
	return cell.ID(l.bases[layer]), nil
}

func (l *Layers) Inbound(to cell.ID, buf []Edge) []Edge {
	if !within(to, l.Size()) {
		return buf
	}
	at := int(to)
	layer := sort.Search(len(l.layers), func(i int) bool { return l.bases[i+1] > at })
	base := l.bases[layer]
	start := len(buf)
	buf = l.layers[layer].Inbound(cell.ID(at-base), buf)
	for i := start; i < len(buf); i++ {
		buf[i].From += cell.ID(base)
		buf[i].To = to
	}
	return l.links.Inbound(to, buf)
}
