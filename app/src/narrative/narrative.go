package narrative

import (
	"cmp"
	"slices"

	"github.com/b070nd/motiv-cell/src/actors"
	"github.com/b070nd/motiv-cell/src/annotate"
	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/engine"
	"github.com/b070nd/motiv-cell/src/evidence"
	"github.com/b070nd/motiv-cell/src/lattice"
	"github.com/b070nd/motiv-cell/src/model"
	"github.com/b070nd/motiv-cell/src/motive"
	"github.com/b070nd/motiv-cell/src/reading"
)

const (
	SameVoid  = cell.Link(1)
	SameOwner = cell.Link(2)
	Part      = motive.Part
)

type Cell struct {
	Doc   string
	Owner string
	State model.Motivation
}

type Moment struct {
	Doc string
	At  int64
}

type Result struct {
	Docs      int64
	Sentences int64
	Answers   int64
	Ticks     int64
	Timeline  []Moment
	Cells     []Cell
	Links     []lattice.Edge
}

type place struct {
	doc      string
	sentence int64
}

func Build(docs []model.Doc, provider annotate.Provider, spec evidence.Spec, gazetteer *actors.Gazetteer) (Result, error) {
	if err := spec.Validate(); err != nil {
		return Result{}, err
	}
	docs = ordered(docs)
	rank := map[string]int64{}
	timeline := make([]Moment, len(docs))
	for i, doc := range docs {
		rank[doc.ID] = int64(i)
		timeline[i] = Moment{Doc: doc.ID, At: doc.At}
	}
	var requests []annotate.Request
	var owners []string
	for _, doc := range docs {
		owner := ""
		for _, sentence := range reading.Sentences(doc) {
			if found := gazetteer.Find(sentence.Text); len(found) > 0 {
				owner = found[0]
			}
			owners = append(owners, owner)
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
	result := Result{Docs: int64(len(docs)), Sentences: int64(len(requests)), Answers: int64(len(answers)), Timeline: timeline}
	var items []engine.Item[model.Signal]
	var changes []engine.Change[model.Motivation]
	var cells []Cell
	ids := map[string]cell.ID{}
	threads := map[string]cell.ID{}
	sameVoid := map[string][]cell.ID{}
	sameOwner := map[string][]cell.ID{}
	for at, request := range requests {
		finding, ok := spec.Read(grouped[place{doc: request.Doc, sentence: request.Sentence}])
		if !ok {
			continue
		}
		owner := owners[at]
		name := request.Doc + "#" + owner + "#" + finding.Void
		id, known := ids[name]
		if !known {
			id = cell.ID(len(cells))
			ids[name] = id
			cells = append(cells, Cell{Doc: request.Doc, Owner: owner})
			state := model.Motivation{ID: name, Level: model.LevelDocument, Place: rank[request.Doc], VoidLabel: finding.Void}
			if owner != "" {
				state.Owners = []string{owner}
			}
			change := engine.Change[model.Motivation]{At: int64(at), Add: []model.Motivation{state}}
			if owner != "" {
				key := "#" + owner + "#" + finding.Void
				parent, exists := threads[key]
				if !exists {
					parent = cell.ID(len(cells))
					threads[key] = parent
					cells = append(cells, Cell{Owner: owner})
					change.Add = append(change.Add, model.Motivation{ID: key, Level: model.LevelThread, Owners: []string{owner}, VoidLabel: finding.Void})
				}
				change.Link = append(change.Link, lattice.Edge{From: id, To: parent, Link: motive.Part})
			}
			change.Link = both(change.Link, sameVoid[finding.Void], id, SameVoid)
			sameVoid[finding.Void] = append(sameVoid[finding.Void], id)
			if owner != "" {
				change.Link = both(change.Link, sameOwner[owner], id, SameOwner)
				sameOwner[owner] = append(sameOwner[owner], id)
			}
			changes = append(changes, change)
			result.Links = append(result.Links, change.Link...)
		}
		items = append(items, engine.Item[model.Signal]{At: int64(at), To: id, Event: model.Signal{
			Kind:     model.SignalEvidence,
			Evidence: model.Evidence{Position: request.Sentence, Intensity: finding.Intensity},
		}})
	}
	machine, err := engine.New(engine.Config[model.Motivation, model.Signal]{Rule: motive.Rule{}, Schedule: engine.Synchronous})
	if err != nil {
		return Result{}, err
	}
	feed, err := engine.NewDiscourse(items, changes, result.Sentences+1)
	if err != nil {
		return Result{}, err
	}
	if err := machine.Run(feed); err != nil {
		return Result{}, err
	}
	result.Ticks = int64(machine.Ticks())
	for i, state := range machine.States(nil) {
		cells[i].State = state
	}
	result.Cells = cells
	return result, nil
}

func ordered(docs []model.Doc) []model.Doc {
	sorted := slices.Clone(docs)
	slices.SortStableFunc(sorted, func(a, b model.Doc) int {
		return cmp.Or(cmp.Compare(a.At, b.At), cmp.Compare(a.ID, b.ID))
	})
	return sorted
}

func both(links []lattice.Edge, others []cell.ID, id cell.ID, kind cell.Link) []lattice.Edge {
	for _, other := range others {
		links = append(links,
			lattice.Edge{From: other, To: id, Link: kind},
			lattice.Edge{From: id, To: other, Link: kind})
	}
	return links
}
