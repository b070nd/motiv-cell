package cell

type ID uint32

type Link uint16

type Tick struct {
	Index uint64
	At    int64
}

type Context struct {
	Tick Tick
	Self ID
}

type Neighbor[S any] struct {
	ID    ID
	Link  Link
	State S
}

type Rule[S, E any] interface {
	Step(ctx Context, self S, neighbors []Neighbor[S], events []E) (S, error)
}

type RuleFunc[S, E any] func(ctx Context, self S, neighbors []Neighbor[S], events []E) (S, error)

func (f RuleFunc[S, E]) Step(ctx Context, self S, neighbors []Neighbor[S], events []E) (S, error) {
	return f(ctx, self, neighbors, events)
}
