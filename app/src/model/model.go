package model

type Doc struct {
	ID       string
	Source   string
	Headline string
	Body     string
	At       int64
}

type Evidence struct {
	Sentence  int64
	Intensity int64
}

type Peak struct {
	Known       bool
	Total       int64
	Single      int64
	Centroid    int64
	Slope       int64
	Bend        int64
	HasSupplied bool
	Supplied    int64
}

type Motivation struct {
	ID        string
	Owners    []string
	VoidText  string
	VoidLabel string
	Evidence  []Evidence
	Peak      Peak
}

type SignalKind uint8

const (
	SignalEvidence SignalKind = 1
	SignalPeak     SignalKind = 2
)

type Signal struct {
	Kind     SignalKind
	Evidence Evidence
	Peak     int64
}
