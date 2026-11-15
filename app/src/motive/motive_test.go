package motive

import (
	"errors"
	"testing"

	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/model"
)

func ev(sentence, intensity int64) model.Signal {
	return model.Signal{Kind: model.SignalEvidence, Evidence: model.Evidence{Sentence: sentence, Intensity: intensity}}
}

func TestField(t *testing.T) {
	var m model.Motivation
	var err error
	m, err = Rule{}.Step(cell.Context{}, m, nil, []model.Signal{ev(2, 3), ev(5, 4), ev(2, 2)})
	if err != nil {
		t.Fatal(err)
	}
	want := model.Peak{Known: true, Total: 2, Single: 5, Centroid: 3, Slope: 4, Bend: 4}
	if m.Peak != want {
		t.Fatalf("got %+v want %+v", m.Peak, want)
	}
	before := m
	m, err = Rule{}.Step(cell.Context{}, m, nil, []model.Signal{ev(6, 1), {Kind: model.SignalPeak, Peak: 5}})
	if err != nil {
		t.Fatal(err)
	}
	want = model.Peak{Known: true, Total: 2, Single: 5, Centroid: 3, Slope: -3, Bend: -7, HasSupplied: true, Supplied: 5}
	if m.Peak != want {
		t.Fatalf("got %+v want %+v", m.Peak, want)
	}
	if len(before.Evidence) != 3 || len(m.Evidence) != 4 {
		t.Fatalf("evidence lengths %d %d", len(before.Evidence), len(m.Evidence))
	}
	same, err := Rule{}.Step(cell.Context{}, m, nil, nil)
	if err != nil || same.Peak != m.Peak {
		t.Fatalf("no-signal step changed state: %+v %v", same.Peak, err)
	}
	if _, err := (Rule{}).Step(cell.Context{}, m, nil, []model.Signal{ev(1, -1)}); !errors.Is(err, ErrIntensity) {
		t.Fatalf("got %v", err)
	}
	if _, err := (Rule{}).Step(cell.Context{}, m, nil, []model.Signal{{Kind: 9}}); !errors.Is(err, ErrSignal) {
		t.Fatalf("got %v", err)
	}
}
