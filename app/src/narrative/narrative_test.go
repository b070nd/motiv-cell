package narrative

import (
	"strings"
	"testing"

	"github.com/b070nd/motiv-cell/src/annotate"
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
	result, err := Build(docs(), annotate.NewFile(answers(t)), spec)
	if err != nil {
		t.Fatal(err)
	}
	if result.Docs != 2 || result.Sentences != 5 || result.Answers != 15 || result.Ticks != 5 {
		t.Fatalf("counts %+v", result)
	}
	if len(result.Cells) != 2 || result.Cells[0].State.ID != "a#capital" || result.Cells[1].Doc != "b" {
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
	if _, err := Build(changed, annotate.NewFile(answers(t)), spec); err != annotate.ErrDrift {
		t.Fatalf("got %v", err)
	}
}
