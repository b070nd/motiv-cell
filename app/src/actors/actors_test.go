package actors

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

const file = `{"id":"acme","name":"Acme Corp","aliases":["Acme"],"tickers":["ACME"]}

{"id":"cobalt","name":"Cobalt Bank","aliases":["Cobalt"]}
`

func TestFind(t *testing.T) {
	g, err := Parse(strings.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	for text, want := range map[string][]string{
		"Cobalt Bank sued Acme Corp. Acme denied it.": {"cobalt", "acme"},
		"Shares of ACME rose.":                        {"acme"},
		"Acmeology and Cobalts are not names.":        nil,
		"The acme of success.":                        nil,
		"Pre-Acme era, said Cobalt.":                  {"acme", "cobalt"},
	} {
		if got := g.Find(text); !slices.Equal(got, want) {
			t.Fatalf("%q: got %v want %v", text, got, want)
		}
	}
	var none *Gazetteer
	if none.Find("Acme") != nil {
		t.Fatal("nil gazetteer")
	}
}

func TestParseErrors(t *testing.T) {
	for in, want := range map[string]error{
		`{"name":"A"}`: ErrID,
		`{"id":"a"}`:   ErrName,
		`{"id":"a","name":"A"}` + "\n" + `{"id":"a","name":"B"}`: ErrDuplicate,
		`{"id":"a","name":"A"}` + "\n" + `{"id":"b","name":"A"}`: ErrAmbiguous,
		`{"id":"a","name":"A","nmae":"x"}`:                       ErrRecord,
	} {
		_, err := Parse(strings.NewReader(in))
		var lineErr *LineError
		if !errors.Is(err, want) || !errors.As(err, &lineErr) {
			t.Fatalf("%s: got %v", in, err)
		}
	}
}
