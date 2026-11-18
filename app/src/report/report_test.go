package report

import (
	"bytes"
	"testing"

	"github.com/b070nd/motiv-cell/src/lattice"
	"github.com/b070nd/motiv-cell/src/model"
	"github.com/b070nd/motiv-cell/src/narrative"
)

func TestWrite(t *testing.T) {
	result := narrative.Result{
		Docs: 1, Sentences: 2, Answers: 6, Ticks: 3,
		Timeline: []narrative.Moment{{Doc: "a", At: 5}},
		Cells: []narrative.Cell{
			{Doc: "a", Owner: "acme", State: model.Motivation{
				ID: "a#capital", Level: model.LevelDocument, VoidLabel: "capital",
				Evidence: []model.Evidence{{Position: 1, Intensity: 700}},
				Peak:     model.Peak{Known: true, Total: 1, Single: 1, Centroid: 1, Slope: 700, Bend: 700},
			}},
			{Owner: "acme", State: model.Motivation{
				ID: "#acme#capital", Level: model.LevelThread, VoidLabel: "capital",
				Evidence: []model.Evidence{{Position: 0, Intensity: 700}},
				Peak:     model.Peak{Known: true, Slope: 700, Bend: 700},
			}},
		},
		Links: []lattice.Edge{{From: 0, To: 1, Link: narrative.Part}},
	}
	var out bytes.Buffer
	if err := Write(&out, result); err != nil {
		t.Fatal(err)
	}
	want := `{"answers":6,"cells":2,"docs":1,"links":1,"sentences":2,"ticks":3,"type":"run"}
{"cell":0,"doc":"a","evidence":1,"id":"a#capital","level":1,"owner":"acme","peak_bend":700,"peak_centroid":1,"peak_single":1,"peak_slope":700,"peak_total":1,"type":"cell","void":"capital"}
{"cell":0,"intensity":700,"sentence":1,"type":"evidence"}
{"cell":1,"doc":"","evidence":1,"id":"#acme#capital","level":2,"owner":"acme","peak_bend":700,"peak_centroid":0,"peak_single":0,"peak_slope":700,"peak_total":0,"type":"cell","void":"capital"}
{"at":5,"cell":1,"doc":"a","intensity":700,"position":0,"type":"evidence"}
{"from":0,"kind":3,"to":1,"type":"link"}
`
	if out.String() != want {
		t.Fatalf("got:\n%s", out.String())
	}
}

func TestWriteRejectsThreadEvidenceOutsideTimeline(t *testing.T) {
	result := narrative.Result{Cells: []narrative.Cell{{Owner: "acme", State: model.Motivation{
		ID: "#acme#capital", Level: model.LevelThread, Evidence: []model.Evidence{{Position: 2, Intensity: 1}},
	}}}}
	var out bytes.Buffer
	if err := Write(&out, result); err == nil {
		t.Fatal("expected an error")
	}
}
