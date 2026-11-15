package engine

import (
	"slices"
	"strconv"

	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/lattice"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrTopology Error = "engine: topology is nil"
	ErrRule     Error = "engine: rule is nil"
	ErrSchedule Error = "engine: unknown schedule"
	ErrSize     Error = "engine: topology size out of range"
	ErrInitial  Error = "engine: initial states do not match topology size"
	ErrEdge     Error = "engine: topology returned an invalid edge"
	ErrDelivery Error = "engine: delivery target out of range"
	ErrCell     Error = "engine: cell out of range"
	ErrSource   Error = "engine: source is nil"
	ErrLength   Error = "engine: discourse length out of range"
	ErrPosition Error = "engine: item position outside discourse"
	ErrLink     Error = "engine: link endpoint out of range"
	ErrRepeat   Error = "engine: link already exists"
)

type Schedule uint8

const (
	Synchronous Schedule = 1
	Sequential  Schedule = 2
)

type Delivery[E any] struct {
	To    cell.ID
	Event E
}

type Batch[S, E any] struct {
	At         int64
	Add        []S
	Link       []lattice.Edge
	Deliveries []Delivery[E]
}

type Source[S, E any] interface {
	Next() (Batch[S, E], bool, error)
}

type Config[S, E any] struct {
	Topology lattice.Topology
	Rule     cell.Rule[S, E]
	Schedule Schedule
	Initial  []S
}

type StepError struct {
	Tick uint64
	Cell cell.ID
	Err  error
}

func (e *StepError) Error() string {
	return "engine: tick " + strconv.FormatUint(e.Tick, 10) +
		" cell " + strconv.FormatUint(uint64(e.Cell), 10) + ": " + e.Err.Error()
}

func (e *StepError) Unwrap() error { return e.Err }

type inbound struct {
	from cell.ID
	link cell.Link
}

type mark struct {
	at     int
	length int
}

type Engine[S, E any] struct {
	rule      cell.Rule[S, E]
	schedule  Schedule
	inbound   [][]inbound
	marks     []mark
	current   []S
	pending   []S
	neighbors []cell.Neighbor[S]
	bounds    []int
	events    []E
	ticks     uint64
}

func New[S, E any](config Config[S, E]) (*Engine[S, E], error) {
	if config.Topology == nil {
		return nil, ErrTopology
	}
	if config.Rule == nil {
		return nil, ErrRule
	}
	if config.Schedule != Synchronous && config.Schedule != Sequential {
		return nil, ErrSchedule
	}
	size := config.Topology.Size()
	if size < 0 || size > lattice.MaxSize {
		return nil, ErrSize
	}
	if len(config.Initial) != size {
		return nil, ErrInitial
	}
	edges := make([][]inbound, size)
	var buf []lattice.Edge
	for at := 0; at < size; at++ {
		to := cell.ID(at)
		buf = config.Topology.Inbound(to, buf[:0])
		for _, edge := range buf {
			if edge.To != to || uint64(edge.From) >= uint64(size) {
				return nil, ErrEdge
			}
			edges[at] = append(edges[at], inbound{from: edge.From, link: edge.Link})
		}
	}
	current := make([]S, size)
	copy(current, config.Initial)
	return &Engine[S, E]{
		rule:     config.Rule,
		schedule: config.Schedule,
		inbound:  edges,
		current:  current,
		pending:  make([]S, size),
		bounds:   make([]int, size),
	}, nil
}

func (e *Engine[S, E]) Size() int { return len(e.current) }

func (e *Engine[S, E]) Ticks() uint64 { return e.ticks }

func (e *Engine[S, E]) State(id cell.ID) (S, error) {
	if uint64(id) >= uint64(len(e.current)) {
		var zero S
		return zero, ErrCell
	}
	return e.current[id], nil
}

func (e *Engine[S, E]) States(buf []S) []S {
	return append(buf, e.current...)
}

func (e *Engine[S, E]) Step(batch Batch[S, E]) error {
	before := len(e.current)
	if len(batch.Add) > lattice.MaxSize-before {
		return ErrSize
	}
	size := before + len(batch.Add)
	for _, delivery := range batch.Deliveries {
		if uint64(delivery.To) >= uint64(size) {
			return ErrDelivery
		}
	}
	for _, edge := range batch.Link {
		if uint64(edge.From) >= uint64(size) || uint64(edge.To) >= uint64(size) {
			return ErrLink
		}
	}
	if err := e.grow(batch.Add, batch.Link); err != nil {
		e.shrink(before)
		return err
	}
	events := e.distribute(batch.Deliveries)
	tick := cell.Tick{Index: e.ticks, At: batch.At}
	source := e.current
	if e.schedule == Sequential {
		copy(e.pending, e.current)
		source = e.pending
	}
	var failure error
	begin := 0
	for at := 0; at < size; at++ {
		neighbors := e.neighbors[:0]
		for _, in := range e.inbound[at] {
			neighbors = append(neighbors, cell.Neighbor[S]{ID: in.from, Link: in.link, State: source[in.from]})
		}
		end := e.bounds[at]
		self := cell.ID(at)
		next, err := e.rule.Step(cell.Context{Tick: tick, Self: self}, source[at], neighbors, events[begin:end:end])
		if err != nil {
			failure = &StepError{Tick: e.ticks, Cell: self, Err: err}
			break
		}
		e.pending[at] = next
		begin = end
		e.neighbors = neighbors
	}
	clear(events)
	clear(e.neighbors[:cap(e.neighbors)])
	if failure != nil {
		e.shrink(before)
		return failure
	}
	e.marks = e.marks[:0]
	e.current, e.pending = e.pending, e.current
	e.ticks++
	return nil
}

func (e *Engine[S, E]) Run(source Source[S, E]) error {
	if source == nil {
		return ErrSource
	}
	for {
		batch, ok, err := source.Next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if err := e.Step(batch); err != nil {
			return err
		}
	}
}

func (e *Engine[S, E]) distribute(deliveries []Delivery[E]) []E {
	clear(e.bounds)
	for _, delivery := range deliveries {
		e.bounds[delivery.To]++
	}
	total := 0
	for at, count := range e.bounds {
		e.bounds[at] = total
		total += count
	}
	e.events = slices.Grow(e.events[:0], total)[:total]
	for _, delivery := range deliveries {
		e.events[e.bounds[delivery.To]] = delivery.Event
		e.bounds[delivery.To]++
	}
	return e.events
}

func (e *Engine[S, E]) grow(add []S, link []lattice.Edge) error {
	e.marks = e.marks[:0]
	size := len(e.current) + len(add)
	e.current = append(e.current, add...)
	e.pending = slices.Grow(e.pending, len(add))[:size]
	e.bounds = slices.Grow(e.bounds, len(add))[:size]
	e.inbound = slices.Grow(e.inbound, len(add))[:size]
	for _, edge := range link {
		entry := inbound{from: edge.From, link: edge.Link}
		if slices.Contains(e.inbound[edge.To], entry) {
			return ErrRepeat
		}
		e.marks = append(e.marks, mark{at: int(edge.To), length: len(e.inbound[edge.To])})
		e.inbound[edge.To] = append(e.inbound[edge.To], entry)
	}
	return nil
}

func (e *Engine[S, E]) shrink(size int) {
	for i := len(e.marks) - 1; i >= 0; i-- {
		m := e.marks[i]
		e.inbound[m.at] = e.inbound[m.at][:m.length]
	}
	e.marks = e.marks[:0]
	clear(e.current[size:])
	clear(e.pending[size:])
	clear(e.inbound[size:])
	e.current = e.current[:size]
	e.pending = e.pending[:size]
	e.bounds = e.bounds[:size]
	e.inbound = e.inbound[:size]
}
