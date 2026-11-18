package motive

import (
	"errors"
	"testing"

	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/model"
)

func ev(sentence, intensity int64) model.Signal {
	return model.Signal{Kind: model.SignalEvidence, Evidence: model.Evidence{Position: sentence, Intensity: intensity}}
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

func TestThreadReadsItsParts(t *testing.T) {
	doc := func(place int64, intensities ...int64) model.Motivation {
		state := model.Motivation{Level: model.LevelDocument, Place: place}
		for i, intensity := range intensities {
			state.Evidence = append(state.Evidence, model.Evidence{Position: int64(i), Intensity: intensity})
		}
		return state
	}
	neighbors := []cell.Neighbor[model.Motivation]{
		{ID: 3, Link: Part, State: doc(2, 400, 400)},
		{ID: 0, Link: Part, State: doc(0, 300, 700)},
		{ID: 1, Link: cell.Link(1), State: doc(1, 999)},
		{ID: 5, Link: Part, State: doc(4)},
	}
	got, err := Rule{}.Step(cell.Context{}, model.Motivation{Level: model.LevelThread}, neighbors, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []model.Evidence{{Position: 0, Intensity: 1000}, {Position: 2, Intensity: 800}}
	if len(got.Evidence) != 2 || got.Evidence[0] != want[0] || got.Evidence[1] != want[1] {
		t.Fatalf("evidence %+v", got.Evidence)
	}
	if !got.Peak.Known || got.Peak.Total != 0 || got.Peak.Slope != 800 || got.Peak.Bend != 1800 {
		t.Fatalf("peak %+v", got.Peak)
	}
	if _, err := (Rule{}).Step(cell.Context{}, model.Motivation{Level: model.LevelThread}, nil, []model.Signal{ev(0, 1)}); err != ErrThread {
		t.Fatalf("signal to thread: %v", err)
	}
}
