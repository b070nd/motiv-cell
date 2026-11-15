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
		Docs: 1, Sentences: 2, Answers: 6, Ticks: 2,
		Cells: []narrative.Cell{{Doc: "a", State: model.Motivation{
			ID: "a#capital", VoidLabel: "capital",
			Evidence: []model.Evidence{{Sentence: 1, Intensity: 700}},
			Peak:     model.Peak{Known: true, Total: 1, Single: 1, Centroid: 1, Slope: 700, Bend: 700},
		}}},
		Links: []lattice.Edge{{From: 0, To: 0, Link: 1}},
	}
	var out bytes.Buffer
	if err := Write(&out, result); err != nil {
		t.Fatal(err)
	}
	want := `{"answers":6,"cells":1,"docs":1,"links":1,"sentences":2,"ticks":2,"type":"run"}
{"cell":0,"doc":"a","evidence":1,"id":"a#capital","peak_bend":700,"peak_centroid":1,"peak_single":1,"peak_slope":700,"peak_total":1,"type":"cell","void":"capital"}
{"cell":0,"intensity":700,"sentence":1,"type":"evidence"}
{"from":0,"kind":1,"to":0,"type":"link"}
`
	if out.String() != want {
		t.Fatalf("got:\n%s", out.String())
	}
}
