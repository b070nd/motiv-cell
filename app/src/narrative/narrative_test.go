package narrative

import (
	"strings"
	"testing"

	"github.com/b070nd/motiv-cell/src/actors"
	"github.com/b070nd/motiv-cell/src/annotate"
	"github.com/b070nd/motiv-cell/src/cell"
	"github.com/b070nd/motiv-cell/src/evidence"
	"github.com/b070nd/motiv-cell/src/lattice"
	"github.com/b070nd/motiv-cell/src/model"
	"github.com/b070nd/motiv-cell/src/reading"
)

const questions = `{"none":"none","pursuit_min":500,"questions":[
{"name":"void","kind":"choice","text":"?","options":["capital","none"]},
{"name":"pursuit","kind":"yesno","text":"?"},
{"name":"intensity","kind":"score","text":"?","options":["low","high"]}]}`

func docs() []model.Doc {
	return []model.Doc{
		{ID: "b", At: 2, Headline: "Beta wants cash", Body: "It asks banks. Rain fell."},
		{ID: "a", At: 1, Headline: "Alpha wants cash", Body: "It sells assets."},
	}
}

func answers(t *testing.T) []annotate.Answer {
	t.Helper()
	marks := map[string][]reading.Sentence{}
	for _, doc := range docs() {
		marks[doc.ID] = reading.Sentences(doc)
	}
	one := func(doc string, sentence int64, void, pursuit string, intensity int64) []annotate.Answer {
		mark := marks[doc][sentence].Mark
		return []annotate.Answer{
			{Doc: doc, Sentence: sentence, Mark: mark, Question: "void", Label: void, Score: 900},
			{Doc: doc, Sentence: sentence, Mark: mark, Question: "pursuit", Label: pursuit, Score: 800},
			{Doc: doc, Sentence: sentence, Mark: mark, Question: "intensity", Score: intensity},
		}
	}
	var out []annotate.Answer
	out = append(out, one("a", 0, "capital", "yes", 300)...)
	out = append(out, one("a", 1, "capital", "yes", 700)...)
	out = append(out, one("b", 0, "capital", "yes", 500)...)
	out = append(out, one("b", 1, "capital", "no", 900)...)
	out = append(out, one("b", 2, "none", "yes", 900)...)
	return out
}

func TestBuild(t *testing.T) {
	spec, err := evidence.Parse(strings.NewReader(questions))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Build(docs(), annotate.NewFile(answers(t)), spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Docs != 2 || result.Sentences != 5 || result.Answers != 15 || result.Ticks != 6 {
		t.Fatalf("counts %+v", result)
	}
	if len(result.Timeline) != 2 || result.Timeline[0] != (Moment{Doc: "a", At: 1}) {
		t.Fatalf("timeline %+v", result.Timeline)
	}
	if len(result.Cells) != 2 || result.Cells[0].State.ID != "a##capital" || result.Cells[1].Doc != "b" {
		t.Fatalf("cells %+v", result.Cells)
	}
	first := result.Cells[0].State
	if len(first.Evidence) != 2 || first.Peak.Total != 1 || first.Peak.Slope != 400 {
		t.Fatalf("first cell %+v", first)
	}
	if len(result.Cells[1].State.Evidence) != 1 {
		t.Fatalf("second cell %+v", result.Cells[1].State)
	}
	want := []lattice.Edge{{From: 0, To: 1, Link: SameVoid}, {From: 1, To: 0, Link: SameVoid}}
	if len(result.Links) != 2 || result.Links[0] != want[0] || result.Links[1] != want[1] {
		t.Fatalf("links %+v", result.Links)
	}
}

func TestBuildRejectsDriftedAnswers(t *testing.T) {
	spec, err := evidence.Parse(strings.NewReader(questions))
	if err != nil {
		t.Fatal(err)
	}
	changed := docs()
	changed[1].Body = "It sells its assets."
	if _, err := Build(changed, annotate.NewFile(answers(t)), spec, nil); err != annotate.ErrDrift {
		t.Fatalf("got %v", err)
	}
}

func TestBuildOwners(t *testing.T) {
	spec, err := evidence.Parse(strings.NewReader(questions))
	if err != nil {
		t.Fatal(err)
	}
	gazetteer, err := actors.New([]actors.Actor{{ID: "alpha", Name: "Alpha"}, {ID: "beta", Name: "Beta"}})
	if err != nil {
		t.Fatal(err)
	}
	extra := append(docs(), model.Doc{ID: "c", At: 3, Headline: "Alpha returns", Body: "It wants cash."})
	stored := answers(t)
	for i, sentence := range reading.Sentences(extra[2]) {
		stored = append(stored,
			annotate.Answer{Doc: "c", Sentence: int64(i), Mark: sentence.Mark, Question: "void", Label: "capital", Score: 900},
			annotate.Answer{Doc: "c", Sentence: int64(i), Mark: sentence.Mark, Question: "pursuit", Label: "yes", Score: 800},
			annotate.Answer{Doc: "c", Sentence: int64(i), Mark: sentence.Mark, Question: "intensity", Score: 400})
	}
	result, err := Build(extra, annotate.NewFile(stored), spec, gazetteer)
	if err != nil {
		t.Fatal(err)
	}
	var documents, threads []Cell
	var docIDs []cell.ID
	for i, c := range result.Cells {
		switch c.State.Level {
		case model.LevelDocument:
			documents = append(documents, c)
			docIDs = append(docIDs, cell.ID(i))
		case model.LevelThread:
			threads = append(threads, c)
		}
	}
	if len(documents) != 3 || len(threads) != 2 {
		t.Fatalf("cells %+v", result.Cells)
	}
	first, third := documents[0], documents[2]
	if first.Owner != "alpha" || first.State.ID != "a#alpha#capital" || len(first.State.Owners) != 1 || len(first.State.Evidence) != 2 {
		t.Fatalf("carry-forward failed: %+v", first)
	}
	if documents[1].Owner != "beta" || third.Owner != "alpha" || len(third.State.Evidence) != 2 || third.State.Place != 2 {
		t.Fatalf("owners %+v", documents)
	}
	alpha := threads[0].State
	if alpha.ID != "#alpha#capital" || len(alpha.Evidence) != 2 || alpha.Evidence[0] != (model.Evidence{Position: 0, Intensity: 1000}) || alpha.Evidence[1] != (model.Evidence{Position: 2, Intensity: 800}) {
		t.Fatalf("alpha thread %+v", alpha)
	}
	if alpha.Peak.Total != 0 || alpha.Peak.Slope != 800 || alpha.Peak.Bend != 1800 {
		t.Fatalf("alpha peak %+v", alpha.Peak)
	}
	if beta := threads[1].State; beta.ID != "#beta#capital" || len(beta.Evidence) != 1 || beta.Evidence[0].Position != 1 {
		t.Fatalf("beta thread %+v", beta)
	}
	var sameVoid, sameOwner, part int
	for _, link := range result.Links {
		switch link.Link {
		case SameVoid:
			sameVoid++
		case SameOwner:
			sameOwner++
			if (link.From != docIDs[0] || link.To != docIDs[2]) && (link.From != docIDs[2] || link.To != docIDs[0]) {
				t.Fatalf("same-owner link %+v", link)
			}
		case Part:
			part++
		}
	}
	if sameVoid != 6 || sameOwner != 2 || part != 3 {
		t.Fatalf("links void %d owner %d part %d", sameVoid, sameOwner, part)
	}
}
