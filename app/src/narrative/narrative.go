package narrative

import (
	"cmp"
	"slices"

	"github.com/b070nd/motiv-cell/src/annotate"
	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/engine"
	"github.com/b070nd/motiv-cell/src/evidence"
	"github.com/b070nd/motiv-cell/src/lattice"
	"github.com/b070nd/motiv-cell/src/model"
	"github.com/b070nd/motiv-cell/src/motive"
	"github.com/b070nd/motiv-cell/src/reading"
)

const SameVoid = cell.Link(1)

type Cell struct {
	Doc   string
	State model.Motivation
}

type Result struct {
	Docs      int64
	Sentences int64
	Answers   int64
	Ticks     int64
	Cells     []Cell
	Links     []lattice.Edge
}

type place struct {
	doc      string
	sentence int64
}

func Build(docs []model.Doc, provider annotate.Provider, spec evidence.Spec) (Result, error) {
	if err := spec.Validate(); err != nil {
		return Result{}, err
	}
	docs = ordered(docs)
	var requests []annotate.Request
	for _, doc := range docs {
		for _, sentence := range reading.Sentences(doc) {
			requests = append(requests, annotate.Request{Doc: doc.ID, Sentence: sentence.Index, Mark: sentence.Mark, Text: sentence.Text})
		}
	}
	answers, err := provider.Answers(requests)
	if err != nil {
		return Result{}, err
	}
	grouped := map[place][]annotate.Answer{}
	for _, answer := range answers {
		key := place{doc: answer.Doc, sentence: answer.Sentence}
		grouped[key] = append(grouped[key], answer)
	}
	result := Result{Docs: int64(len(docs)), Sentences: int64(len(requests)), Answers: int64(len(answers))}
	var items []engine.Item[model.Signal]
	var changes []engine.Change[model.Motivation]
	var cellDocs []string
	ids := map[string]cell.ID{}
	sameVoid := map[string][]cell.ID{}
	for at, request := range requests {
		finding, ok := spec.Read(grouped[place{doc: request.Doc, sentence: request.Sentence}])
		if !ok {
			continue
		}
		name := request.Doc + "#" + finding.Void
		id, known := ids[name]
		if !known {
			id = cell.ID(len(cellDocs))
			ids[name] = id
			cellDocs = append(cellDocs, request.Doc)
			change := engine.Change[model.Motivation]{At: int64(at), Add: []model.Motivation{{ID: name, VoidLabel: finding.Void}}}
			for _, other := range sameVoid[finding.Void] {
				change.Link = append(change.Link,
					lattice.Edge{From: other, To: id, Link: SameVoid},
					lattice.Edge{From: id, To: other, Link: SameVoid})
			}
			sameVoid[finding.Void] = append(sameVoid[finding.Void], id)
			changes = append(changes, change)
			result.Links = append(result.Links, change.Link...)
		}
		items = append(items, engine.Item[model.Signal]{At: int64(at), To: id, Event: model.Signal{
			Kind:     model.SignalEvidence,
			Evidence: model.Evidence{Sentence: request.Sentence, Intensity: finding.Intensity},
		}})
	}
	machine, err := engine.New(engine.Config[model.Motivation, model.Signal]{Rule: motive.Rule{}, Schedule: engine.Synchronous})
	if err != nil {
		return Result{}, err
	}
	feed, err := engine.NewDiscourse(items, changes, result.Sentences)
	if err != nil {
		return Result{}, err
	}
	if err := machine.Run(feed); err != nil {
		return Result{}, err
	}
	result.Ticks = int64(machine.Ticks())
	for i, state := range machine.States(nil) {
		result.Cells = append(result.Cells, Cell{Doc: cellDocs[i], State: state})
	}
	return result, nil
}

func ordered(docs []model.Doc) []model.Doc {
	sorted := slices.Clone(docs)
	slices.SortStableFunc(sorted, func(a, b model.Doc) int {
		return cmp.Or(cmp.Compare(a.At, b.At), cmp.Compare(a.ID, b.ID))
	})
	return sorted
}
