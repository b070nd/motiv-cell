package motive

import (
	"cmp"
	"slices"

	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/model"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrSignal    Error = "motive: unknown signal kind"
	ErrIntensity Error = "motive: negative intensity"
	ErrPosition  Error = "motive: negative position"
	ErrOverflow  Error = "motive: intensity sum overflows"
	ErrThread    Error = "motive: a thread cell takes no signals"
)

const Part = cell.Link(3)

type Rule struct{}

func (Rule) Step(_ cell.Context, self model.Motivation, neighbors []cell.Neighbor[model.Motivation], signals []model.Signal) (model.Motivation, error) {
	if self.Level == model.LevelThread {
		return thread(self, neighbors, signals)
	}
	if len(signals) == 0 {
		return self, nil
	}
	evidence := self.Evidence[:len(self.Evidence):len(self.Evidence)]
	peak := self.Peak
	for _, signal := range signals {
		switch signal.Kind {
		case model.SignalEvidence:
			if signal.Evidence.Intensity < 0 {
				return self, ErrIntensity
			}
			if signal.Evidence.Position < 0 {
				return self, ErrPosition
			}
			evidence = append(evidence, signal.Evidence)
		case model.SignalPeak:
			if signal.Peak < 0 {
				return self, ErrPosition
			}
			peak.HasSupplied = true
			peak.Supplied = signal.Peak
		default:
			return self, ErrSignal
		}
	}
	computed, err := Field(evidence)
	if err != nil {
		return self, err
	}
	computed.HasSupplied = peak.HasSupplied
	computed.Supplied = peak.Supplied
	self.Evidence = evidence
	self.Peak = computed
	return self, nil
}

func thread(self model.Motivation, neighbors []cell.Neighbor[model.Motivation], signals []model.Signal) (model.Motivation, error) {
	if len(signals) > 0 {
		return self, ErrThread
	}
	var evidence []model.Evidence
	for _, neighbor := range neighbors {
		if neighbor.Link != Part {
			continue
		}
		var total int64
		for _, item := range neighbor.State.Evidence {
			if total > maxSum-item.Intensity {
				return self, ErrOverflow
			}
			total += item.Intensity
		}
		if total > 0 {
			evidence = append(evidence, model.Evidence{Position: neighbor.State.Place, Intensity: total})
		}
	}
	slices.SortStableFunc(evidence, func(a, b model.Evidence) int { return cmp.Compare(a.Position, b.Position) })
	peak, err := Field(evidence)
	if err != nil {
		return self, err
	}
	self.Evidence = evidence
	self.Peak = peak
	return self, nil
}

func Field(evidence []model.Evidence) (model.Peak, error) {
	if len(evidence) == 0 {
		return model.Peak{}, nil
	}
	sorted := slices.Clone(evidence)
	slices.SortStableFunc(sorted, func(a, b model.Evidence) int { return cmp.Compare(a.Position, b.Position) })
	peak := model.Peak{Known: true}
	var weight, moment int64
	strongest, best := int64(-1), int64(-1)
	var last, previous, older int64
	front := sorted[len(sorted)-1].Position
	for i := 0; i < len(sorted); {
		position := sorted[i].Position
		var total int64
		for ; i < len(sorted) && sorted[i].Position == position; i++ {
			intensity := sorted[i].Intensity
			if intensity > strongest {
				strongest = intensity
				peak.Single = position
			}
			if total > maxSum-intensity {
				return model.Peak{}, ErrOverflow
			}
			total += intensity
		}
		if total > best {
			best = total
			peak.Total = position
		}
		if weight > maxSum-total {
			return model.Peak{}, ErrOverflow
		}
		weight += total
		if position != 0 && total > (maxSum-moment)/position {
			return model.Peak{}, ErrOverflow
		}
		moment += total * position
		switch position {
		case front:
			last = total
		case front - 1:
			previous = total
		case front - 2:
			older = total
		}
	}
	if weight > 0 {
		peak.Centroid = moment / weight
	} else {
		peak.Centroid = sorted[0].Position
	}
	peak.Slope = last - previous
	peak.Bend = last - 2*previous + older
	return peak, nil
}

const maxSum = 1<<62 - 1
