package lattice

import (
	"cmp"

	"github.com/b070nd/motiv-cell/src/cell"
)

const MaxSize = 1<<31 - 1

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrSize      Error = "lattice: size out of range"
	ErrRadius    Error = "lattice: radius out of range"
	ErrEndpoint  Error = "lattice: edge endpoint out of range"
	ErrDuplicate Error = "lattice: duplicate edge"
	ErrLayers    Error = "lattice: no layers"
	ErrLayer     Error = "lattice: layer out of range"
)

type Edge struct {
	From cell.ID
	To   cell.ID
	Link cell.Link
}

type Topology interface {
	Size() int
	Inbound(to cell.ID, buf []Edge) []Edge
}

func compare(a, b Edge) int {
	if c := cmp.Compare(a.To, b.To); c != 0 {
		return c
	}
	if c := cmp.Compare(a.From, b.From); c != 0 {
		return c
	}
	return cmp.Compare(a.Link, b.Link)
}

func within(id cell.ID, size int) bool {
	return size > 0 && uint64(id) < uint64(size)
}
