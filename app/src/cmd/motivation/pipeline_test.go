package main

import (
	"bytes"
	"testing"

	"github.com/b070nd/motiv-cell/src/annotate"
	"github.com/b070nd/motiv-cell/src/model"
)

func TestSimulate(t *testing.T) {
	docs := []model.Doc{
		{ID: "b", At: 2, Headline: "Beta wants cash", Body: "It asks banks. Rain fell."},
		{ID: "a", At: 1, Headline: "Alpha wants cash", Body: "It sells assets."},
	}
	answer := func(doc string, sentence int64, void string, yes bool, intensity int64) []annotate.Answer {
		label := "no"
		if yes {
			label = "yes"
		}
		return []annotate.Answer{
			{Doc: doc, Sentence: sentence, Question: "void", Label: void, Score: 900},
			{Doc: doc, Sentence: sentence, Question: "pursuit", Label: label, Score: 800},
			{Doc: doc, Sentence: sentence, Question: "intensity", Score: intensity},
		}
	}
	var answers []annotate.Answer
	answers = append(answers, answer("a", 0, "capital", true, 300)...)
	answers = append(answers, answer("a", 1, "capital", true, 700)...)
	answers = append(answers, answer("b", 0, "capital", true, 500)...)
	answers = append(answers, answer("b", 1, "capital", false, 900)...)
	answers = append(answers, answer("b", 2, "none", true, 900)...)
	var out bytes.Buffer
	total, err := simulate(docs, answers, "none", 500, &out)
	if err != nil {
		t.Fatal(err)
	}
	if total.cells != 2 || total.links != 2 || total.sentences != 5 {
		t.Fatalf("summary %+v", total)
	}
	want := `{"answers":15,"cells":2,"docs":2,"links":2,"sentences":5,"ticks":5,"type":"run"}
{"cell":0,"doc":"a","evidence":2,"id":"a#capital","peak_bend":100,"peak_centroid":0,"peak_single":1,"peak_slope":400,"peak_total":1,"type":"cell","void":"capital"}
{"cell":0,"intensity":300,"sentence":0,"type":"evidence"}
{"cell":0,"intensity":700,"sentence":1,"type":"evidence"}
{"cell":1,"doc":"b","evidence":1,"id":"b#capital","peak_bend":500,"peak_centroid":0,"peak_single":0,"peak_slope":500,"peak_total":0,"type":"cell","void":"capital"}
{"cell":1,"intensity":500,"sentence":0,"type":"evidence"}
{"from":0,"kind":1,"to":1,"type":"link"}
{"from":1,"kind":1,"to":0,"type":"link"}
`
	if out.String() != want {
		t.Fatalf("got:\n%s", out.String())
	}
}
