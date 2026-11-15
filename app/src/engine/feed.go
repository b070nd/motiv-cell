package engine

import (
	"cmp"
	"slices"

	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/lattice"
)

type Item[E any] struct {
	At    int64
	To    cell.ID
	Event E
}

type Change[S any] struct {
	At   int64
	Add  []S
	Link []lattice.Edge
}

type Feed[S, E any] struct {
	items      []Item[E]
	changes    []Change[S]
	nextItem   int
	nextChange int
	dense      bool
	at         int64
	length     int64
	deliveries []Delivery[E]
	add        []S
	link       []lattice.Edge
}

func NewTimestamps[S, E any](items []Item[E], changes []Change[S]) *Feed[S, E] {
	return &Feed[S, E]{items: orderedItems(items), changes: orderedChanges(changes)}
}

func NewDiscourse[S, E any](items []Item[E], changes []Change[S], length int64) (*Feed[S, E], error) {
	if length < 0 {
		return nil, ErrLength
	}
	for _, item := range items {
		if item.At < 0 || item.At >= length {
			return nil, ErrPosition
		}
	}
	for _, change := range changes {
		if change.At < 0 || change.At >= length {
			return nil, ErrPosition
		}
	}
	return &Feed[S, E]{items: orderedItems(items), changes: orderedChanges(changes), dense: true, length: length}, nil
}

func (f *Feed[S, E]) Next() (Batch[S, E], bool, error) {
	var at int64
	switch {
	case f.dense:
		if f.at >= f.length {
			return Batch[S, E]{}, false, nil
		}
		at = f.at
		f.at++
	case f.nextItem < len(f.items) && f.nextChange < len(f.changes):
		at = min(f.items[f.nextItem].At, f.changes[f.nextChange].At)
	case f.nextItem < len(f.items):
		at = f.items[f.nextItem].At
	case f.nextChange < len(f.changes):
		at = f.changes[f.nextChange].At
	default:
		return Batch[S, E]{}, false, nil
	}
	clear(f.deliveries)
	clear(f.add)
	f.deliveries, f.add, f.link = f.deliveries[:0], f.add[:0], f.link[:0]
	for f.nextChange < len(f.changes) && f.changes[f.nextChange].At == at {
		f.add = append(f.add, f.changes[f.nextChange].Add...)
		f.link = append(f.link, f.changes[f.nextChange].Link...)
		f.nextChange++
	}
	for f.nextItem < len(f.items) && f.items[f.nextItem].At == at {
		f.deliveries = append(f.deliveries, Delivery[E]{To: f.items[f.nextItem].To, Event: f.items[f.nextItem].Event})
		f.nextItem++
	}
	return Batch[S, E]{At: at, Add: f.add, Link: f.link, Deliveries: f.deliveries}, true, nil
}

func orderedItems[E any](items []Item[E]) []Item[E] {
	sorted := slices.Clone(items)
	slices.SortStableFunc(sorted, func(a, b Item[E]) int { return cmp.Compare(a.At, b.At) })
	return sorted
}

func orderedChanges[S any](changes []Change[S]) []Change[S] {
	sorted := slices.Clone(changes)
	slices.SortStableFunc(sorted, func(a, b Change[S]) int { return cmp.Compare(a.At, b.At) })
	return sorted
}
