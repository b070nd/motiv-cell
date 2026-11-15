package main

import (
	"cmp"
	"io"
	"slices"

	"github.com/b070nd/motiv-cell/src/annotate"
	"github.com/b070nd/motiv-cell/src/canon"
	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/engine"
	"github.com/b070nd/motiv-cell/src/lattice"
	"github.com/b070nd/motiv-cell/src/model"
	"github.com/b070nd/motiv-cell/src/motive"
	"github.com/b070nd/motiv-cell/src/segment"
)

const (
	questionVoid      = "void"
	questionPursuit   = "pursuit"
	questionIntensity = "intensity"
	labelYes          = "yes"
	linkSameVoid      = cell.Link(1)
)

type sentenceKey struct {
	doc      string
	sentence int64
}

type reading struct {
	void      string
	hasVoid   bool
	pursuit   int64
	gated     bool
	intensity int64
	hasScore  bool
}

type summary struct {
	docs      int64
	sentences int64
	answers   int64
	cells     int64
	links     int64
	ticks     int64
}

func requests(docs []model.Doc) []annotate.Request {
	var out []annotate.Request
	for _, doc := range docs {
		for i, text := range sentences(doc) {
			out = append(out, annotate.Request{Doc: doc.ID, Sentence: int64(i), Text: text})
		}
	}
	return out
}

func sentences(doc model.Doc) []string {
	var out []string
	if doc.Headline != "" {
		out = append(out, doc.Headline)
	}
	return append(out, segment.Sentences(doc.Body)...)
}

func ordered(docs []model.Doc) []model.Doc {
	sorted := slices.Clone(docs)
	slices.SortStableFunc(sorted, func(a, b model.Doc) int {
		if c := cmp.Compare(a.At, b.At); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return sorted
}

func readings(answers []annotate.Answer) map[sentenceKey]reading {
	out := map[sentenceKey]reading{}
	for _, answer := range answers {
		key := sentenceKey{doc: answer.Doc, sentence: answer.Sentence}
		r := out[key]
		switch answer.Question {
		case questionVoid:
			r.void, r.hasVoid = answer.Label, true
		case questionPursuit:
			r.gated = true
			r.pursuit = answer.Score
			if answer.Label != labelYes {
				r.pursuit = 1000 - answer.Score
			}
		case questionIntensity:
			r.intensity, r.hasScore = answer.Score, true
		}
		out[key] = r
	}
	return out
}

func simulate(docs []model.Doc, answers []annotate.Answer, none string, pursuitMin int64, out io.Writer) (summary, error) {
	docs = ordered(docs)
	read := readings(answers)
	var items []engine.Item[model.Signal]
	var changes []engine.Change[model.Motivation]
	var cells []model.Motivation
	var cellDocs []string
	ids := map[string]cell.ID{}
	byVoid := map[string][]cell.ID{}
	var links int64
	var position int64
	total := summary{docs: int64(len(docs)), answers: int64(len(answers))}
	for _, doc := range docs {
		count := int64(len(sentences(doc)))
		for sentence := int64(0); sentence < count; sentence++ {
			at := position
			position++
			r, ok := read[sentenceKey{doc: doc.ID, sentence: sentence}]
			if !ok || !r.hasVoid || !r.hasScore || r.void == none {
				continue
			}
			if r.gated && r.pursuit < pursuitMin {
				continue
			}
			name := doc.ID + "#" + r.void
			id, known := ids[name]
			if !known {
				id = cell.ID(len(cells))
				ids[name] = id
				state := model.Motivation{ID: name, VoidLabel: r.void}
				cells = append(cells, state)
				cellDocs = append(cellDocs, doc.ID)
				change := engine.Change[model.Motivation]{At: at, Add: []model.Motivation{state}}
				for _, other := range byVoid[r.void] {
					change.Link = append(change.Link,
						lattice.Edge{From: other, To: id, Link: linkSameVoid},
						lattice.Edge{From: id, To: other, Link: linkSameVoid})
					links += 2
				}
				byVoid[r.void] = append(byVoid[r.void], id)
				changes = append(changes, change)
			}
			items = append(items, engine.Item[model.Signal]{At: at, To: id, Event: model.Signal{
				Kind:     model.SignalEvidence,
				Evidence: model.Evidence{Sentence: sentence, Intensity: r.intensity},
			}})
		}
	}
	total.sentences = position
	empty, err := lattice.NewGraph(0, nil)
	if err != nil {
		return total, err
	}
	machine, err := engine.New(engine.Config[model.Motivation, model.Signal]{
		Topology: empty,
		Rule:     motive.Rule{},
		Schedule: engine.Synchronous,
	})
	if err != nil {
		return total, err
	}
	feed, err := engine.NewDiscourse(items, changes, position)
	if err != nil {
		return total, err
	}
	if err := machine.Run(feed); err != nil {
		return total, err
	}
	total.cells = int64(machine.Size())
	total.links = links
	total.ticks = int64(machine.Ticks())
	writer := canon.NewWriter(out)
	if err := writer.Write(
		canon.Str("type", "run"),
		canon.Int("docs", total.docs),
		canon.Int("sentences", total.sentences),
		canon.Int("answers", total.answers),
		canon.Int("cells", total.cells),
		canon.Int("links", total.links),
		canon.Int("ticks", total.ticks),
	); err != nil {
		return total, err
	}
	for i, state := range machine.States(nil) {
		fields := []canon.Field{
			canon.Str("type", "cell"),
			canon.Int("cell", int64(i)),
			canon.Str("id", state.ID),
			canon.Str("doc", cellDocs[i]),
			canon.Str("void", state.VoidLabel),
			canon.Int("evidence", int64(len(state.Evidence))),
		}
		if state.Peak.Known {
			fields = append(fields,
				canon.Int("peak_total", state.Peak.Total),
				canon.Int("peak_single", state.Peak.Single),
				canon.Int("peak_centroid", state.Peak.Centroid),
				canon.Int("peak_slope", state.Peak.Slope),
				canon.Int("peak_bend", state.Peak.Bend),
			)
		}
		if err := writer.Write(fields...); err != nil {
			return total, err
		}
		for _, item := range state.Evidence {
			if err := writer.Write(
				canon.Str("type", "evidence"),
				canon.Int("cell", int64(i)),
				canon.Int("sentence", item.Sentence),
				canon.Int("intensity", item.Intensity),
			); err != nil {
				return total, err
			}
		}
	}
	for _, change := range changes {
		for _, edge := range change.Link {
			if err := writer.Write(
				canon.Str("type", "link"),
				canon.Int("from", int64(edge.From)),
				canon.Int("to", int64(edge.To)),
				canon.Int("kind", int64(edge.Link)),
			); err != nil {
				return total, err
			}
		}
	}
	return total, nil
}
