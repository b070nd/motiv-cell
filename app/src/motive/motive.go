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
	ErrSentence  Error = "motive: negative sentence index"
	ErrOverflow  Error = "motive: intensity sum overflows"
)

type Rule struct{}

func (Rule) Step(_ cell.Context, self model.Motivation, _ []cell.Neighbor[model.Motivation], signals []model.Signal) (model.Motivation, error) {
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
			if signal.Evidence.Sentence < 0 {
				return self, ErrSentence
			}
			evidence = append(evidence, signal.Evidence)
		case model.SignalPeak:
			if signal.Peak < 0 {
				return self, ErrSentence
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

func Field(evidence []model.Evidence) (model.Peak, error) {
	if len(evidence) == 0 {
		return model.Peak{}, nil
	}
	sorted := slices.Clone(evidence)
	slices.SortStableFunc(sorted, func(a, b model.Evidence) int { return cmp.Compare(a.Sentence, b.Sentence) })
	peak := model.Peak{Known: true}
	var weight, moment int64
	strongest, best := int64(-1), int64(-1)
	var last, previous, older int64
	front := sorted[len(sorted)-1].Sentence
	for i := 0; i < len(sorted); {
		sentence := sorted[i].Sentence
		var total int64
		for ; i < len(sorted) && sorted[i].Sentence == sentence; i++ {
			intensity := sorted[i].Intensity
			if intensity > strongest {
				strongest = intensity
				peak.Single = sentence
			}
			if total > maxSum-intensity {
				return model.Peak{}, ErrOverflow
			}
			total += intensity
		}
		if total > best {
			best = total
			peak.Total = sentence
		}
		if weight > maxSum-total {
			return model.Peak{}, ErrOverflow
		}
		weight += total
		if sentence != 0 && total > (maxSum-moment)/sentence {
			return model.Peak{}, ErrOverflow
		}
		moment += total * sentence
		switch sentence {
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
		peak.Centroid = sorted[0].Sentence
	}
	peak.Slope = last - previous
	peak.Bend = last - 2*previous + older
	return peak, nil
}

const maxSum = 1<<62 - 1
