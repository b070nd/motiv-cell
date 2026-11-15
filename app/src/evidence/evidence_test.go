package evidence

import (
	"strings"
	"testing"

	"github.com/b070nd/motiv-cell/src/annotate"
)

const good = `{"model":"english","none":"none","pursuit_min":500,"questions":[
{"name":"void","kind":"choice","text":"?","options":["capital","none"]},
{"name":"pursuit","kind":"yesno","text":"?"},
{"name":"intensity","kind":"score","text":"?","options":["low","high"]}]}`

func TestParse(t *testing.T) {
	if _, err := Parse(strings.NewReader(good)); err != nil {
		t.Fatal(err)
	}
	for bad, want := range map[string]error{
		strings.Replace(good, `"name":"void"`, `"name":"gap"`, 1):           ErrVoid,
		strings.Replace(good, `"none":"none"`, `"none":"nothing"`, 1):       ErrNone,
		strings.Replace(good, `"kind":"score"`, `"kind":"choice"`, 1):       ErrIntensity,
		strings.Replace(good, `"kind":"yesno"`, `"kind":"score"`, 1):        ErrPursuit,
		strings.Replace(good, `"pursuit_min":500`, `"pursuit_min":1001`, 1): ErrThreshold,
		strings.Replace(good, `"model":"english"`, `"modle":"english"`, 1):  ErrFile,
		strings.Replace(good, `"name":"pursuit"`, `"name":"void"`, 1):       ErrName,
		strings.Replace(good, `"kind":"yesno"`, `"kind":"maybe"`, 1):        ErrKind,
	} {
		if _, err := Parse(strings.NewReader(bad)); err != want {
			t.Fatalf("got %v want %v", err, want)
		}
	}
}

func TestRead(t *testing.T) {
	spec, err := Parse(strings.NewReader(good))
	if err != nil {
		t.Fatal(err)
	}
	answers := func(void, pursuit string, p, intensity int64) []annotate.Answer {
		return []annotate.Answer{
			{Question: "void", Label: void, Score: 900},
			{Question: "pursuit", Label: pursuit, Score: p},
			{Question: "intensity", Score: intensity},
		}
	}
	if f, ok := spec.Read(answers("capital", "yes", 700, 640)); !ok || f != (Finding{Void: "capital", Intensity: 640}) {
		t.Fatalf("got %+v %v", f, ok)
	}
	if f, ok := spec.Read(answers("capital", "no", 400, 640)); !ok || f.Intensity != 640 {
		t.Fatalf("no at 400 means yes at 600: %+v %v", f, ok)
	}
	if _, ok := spec.Read(answers("capital", "no", 700, 640)); ok {
		t.Fatal("pursuit below threshold")
	}
	if _, ok := spec.Read(answers("none", "yes", 900, 640)); ok {
		t.Fatal("none label")
	}
	if _, ok := spec.Read(answers("capital", "yes", 900, 640)[:2]); ok {
		t.Fatal("missing intensity")
	}
}
