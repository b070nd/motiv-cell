package clock

type Error string

func (e Error) Error() string {
	return string(e)
}

const ErrExhausted Error = "clock: replay exhausted"

type Instant int64

type Clock interface {
	Now() (Instant, error)
}

type Fixed struct {
	at Instant
}

func NewFixed(at Instant) Fixed {
	return Fixed{at: at}
}

func (f Fixed) Now() (Instant, error) {
	return f.at, nil
}

type Replay struct {
	instants []Instant
	next     int
}

func NewReplay(instants []Instant) *Replay {
	copied := make([]Instant, len(instants))
	copy(copied, instants)
	return &Replay{instants: copied}
}

func (r *Replay) Now() (Instant, error) {
	if r.next >= len(r.instants) {
		return 0, ErrExhausted
	}
	at := r.instants[r.next]
	r.next++
	return at, nil
}
