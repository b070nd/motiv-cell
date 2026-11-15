package segment

import (
	"slices"
	"testing"
)

func TestSentences(t *testing.T) {
	got := Sentences(`Acme Corp. wants U.S. approval. "We need it," Mr. Lee said. Shares fell 3.5 percent! Why?  It lacks cash`)
	want := []string{`Acme Corp. wants U.S. approval.`, `"We need it," Mr. Lee said.`, `Shares fell 3.5 percent!`, `Why?`, `It lacks cash`}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q", got)
	}
	if len(Sentences("  ")) != 0 {
		t.Fatal("blank text")
	}
}
