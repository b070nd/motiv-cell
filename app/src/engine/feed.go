package engine

import (
	"cmp"
	"slices"

	"github.com/b070nd/motiv-cell/src/cell"
)

type Item[E any] struct {
	At    int64
	To    cell.ID
	Event E
}

type Feed[E any] struct {
	items  []Item[E]
	next   int
	dense  bool
	at     int64
	length int64
	buf    []Delivery[E]
}

func NewTimestamps[E any](items []Item[E]) *Feed[E] {
	return &Feed[E]{items: ordered(items)}
}

func NewDiscourse[E any](items []Item[E], length int64) (*Feed[E], error) {
	if length < 0 {
		return nil, ErrLength
	}
	for _, item := range items {
		if item.At < 0 || item.At >= length {
			return nil, ErrPosition
		}
	}
	return &Feed[E]{items: ordered(items), dense: true, length: length}, nil
}

func (f *Feed[E]) Next() (Batch[E], bool, error) {
	if f.dense {
		if f.at >= f.length {
			return Batch[E]{}, false, nil
		}
		at := f.at
		f.at++
		return Batch[E]{At: at, Deliveries: f.take(at)}, true, nil
	}
	if f.next >= len(f.items) {
		return Batch[E]{}, false, nil
	}
	at := f.items[f.next].At
	return Batch[E]{At: at, Deliveries: f.take(at)}, true, nil
}

func (f *Feed[E]) take(at int64) []Delivery[E] {
	clear(f.buf)
	f.buf = f.buf[:0]
	for f.next < len(f.items) && f.items[f.next].At == at {
		f.buf = append(f.buf, Delivery[E]{To: f.items[f.next].To, Event: f.items[f.next].Event})
		f.next++
	}
	return f.buf
}

func ordered[E any](items []Item[E]) []Item[E] {
	sorted := slices.Clone(items)
	slices.SortStableFunc(sorted, func(a, b Item[E]) int { return cmp.Compare(a.At, b.At) })
	return sorted
}
