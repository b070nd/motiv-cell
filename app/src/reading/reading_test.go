package reading

import (
	"testing"

	"github.com/b070nd/motiv-cell/src/model"
)

func TestSentences(t *testing.T) {
	got := Sentences(model.Doc{Headline: "Acme wants cash", Body: "It asks banks. Rain fell."})
	if len(got) != 3 || got[0].Text != "Acme wants cash" || got[2].Index != 2 || got[2].Text != "Rain fell." {
		t.Fatalf("got %+v", got)
	}
	if len(got[1].Mark) != 16 {
		t.Fatalf("mark %q", got[1].Mark)
	}
	if got[1].Mark == got[2].Mark || got[1].Mark != Mark("It asks banks.") {
		t.Fatalf("marks %q %q", got[1].Mark, got[2].Mark)
	}
	if n := len(Sentences(model.Doc{Body: "One."})); n != 1 {
		t.Fatalf("no headline: %d", n)
	}
}
