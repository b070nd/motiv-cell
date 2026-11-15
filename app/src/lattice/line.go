package lattice

import "github.com/b070nd/motiv-cell/src/cell"

type Line struct {
	size   int
	radius int
	link   cell.Link
}

func NewLine(size, radius int, link cell.Link) (Line, error) {
	if size < 0 || size > MaxSize {
		return Line{}, ErrSize
	}
	if radius < 0 || radius > MaxSize {
		return Line{}, ErrRadius
	}
	return Line{size: size, radius: radius, link: link}, nil
}

func (l Line) Size() int { return l.size }

func (l Line) Inbound(to cell.ID, buf []Edge) []Edge {
	if !within(to, l.size) {
		return buf
	}
	at := int(to)
	low := 0
	if l.radius < at {
		low = at - l.radius
	}
	high := l.size - 1
	if l.radius < high-at {
		high = at + l.radius
	}
	for from := low; from <= high; from++ {
		if from == at {
			continue
		}
		buf = append(buf, Edge{From: cell.ID(from), To: to, Link: l.link})
	}
	return buf
}
