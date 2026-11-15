package annotate

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func sample() []Answer {
	return []Answer{
		{Doc: "b", Sentence: 1, Question: "q", Label: "yes", Score: 900},
		{Doc: "a", Sentence: 2, Question: "tone", Label: "", Score: 250},
		{Doc: "a", Sentence: 1, Question: "q", Label: "no", Score: 0},
		{Doc: "a\"x", Sentence: 1, Question: "q", Label: "é\n", Score: 1000},
	}
}

func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sample()); err != nil {
		t.Fatal(err)
	}
	got, err := Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sample()) {
		t.Fatalf("got %+v", got)
	}
}

func TestWriteBytes(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, []Answer{{Doc: "d1", Sentence: 3, Question: "q", Label: "yes", Score: 730}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"doc":"d1","label":"yes","question":"q","score":730,"sentence":3}` + "\n"
	if buf.String() != want {
		t.Fatalf("got %q want %q", buf.String(), want)
	}
}

func TestReadErrors(t *testing.T) {
	in := "\n" + `{"doc":"a","label":"","question":"q","score":1,"sentence":1}` + "\n\n" + "nope\n"
	_, err := Read(strings.NewReader(in))
	if err == nil || !strings.Contains(err.Error(), "line 4") {
		t.Fatalf("got %v", err)
	}
	_, err = Read(strings.NewReader(`{"doc":"a"}`))
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("got %v", err)
	}
}

func TestFileLookupOrder(t *testing.T) {
	f := NewFile([]Answer{
		{Doc: "a", Sentence: 1, Question: "z", Label: "x", Score: 1},
		{Doc: "a", Sentence: 1, Question: "b", Label: "y", Score: 2},
		{Doc: "a", Sentence: 1, Question: "b", Label: "x", Score: 3},
		{Doc: "a", Sentence: 2, Question: "q", Label: "x", Score: 4},
		{Doc: "c", Sentence: 1, Question: "q", Label: "x", Score: 5},
	})
	got, err := f.Answers([]Request{
		{Doc: "c", Sentence: 1},
		{Doc: "missing", Sentence: 9},
		{Doc: "a", Sentence: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	var order []int64
	for _, a := range got {
		order = append(order, a.Score)
	}
	if !reflect.DeepEqual(order, []int64{5, 3, 2, 1}) {
		t.Fatalf("got %v", order)
	}
}

type fixed struct {
	answers []Answer
	err     error
}

func (p fixed) Answers([]Request) ([]Answer, error) { return p.answers, p.err }

func TestRecorder(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(fixed{answers: sample()}, &buf)
	got, err := rec.Answers([]Request{{Doc: "a"}})
	if err != nil || !reflect.DeepEqual(got, sample()) {
		t.Fatalf("got %v %v", got, err)
	}
	read, err := Read(&buf)
	if err != nil || !reflect.DeepEqual(read, sample()) {
		t.Fatalf("read %v %v", read, err)
	}
}

func TestRecorderInnerFails(t *testing.T) {
	var buf bytes.Buffer
	boom := errors.New("boom")
	_, err := NewRecorder(fixed{answers: sample(), err: boom}, &buf).Answers(nil)
	if err != boom || buf.Len() != 0 {
		t.Fatalf("err %v len %d", err, buf.Len())
	}
}

func TestCompare(t *testing.T) {
	a := []Answer{
		{Doc: "d", Sentence: 1, Question: "same", Label: "yes", Score: 800},
		{Doc: "d", Sentence: 1, Question: "label", Label: "yes", Score: 800},
		{Doc: "d", Sentence: 1, Question: "score", Label: "yes", Score: 800},
		{Doc: "d", Sentence: 1, Question: "onlya", Label: "yes", Score: 1},
		{Doc: "d", Sentence: 1, Question: "multi", Label: "p", Score: 100},
		{Doc: "d", Sentence: 1, Question: "multi", Label: "q", Score: 600},
	}
	b := []Answer{
		{Doc: "d", Sentence: 1, Question: "same", Label: "yes", Score: 800},
		{Doc: "d", Sentence: 1, Question: "label", Label: "no", Score: 700},
		{Doc: "d", Sentence: 1, Question: "score", Label: "yes", Score: 650},
		{Doc: "d", Sentence: 1, Question: "onlyb", Label: "yes", Score: 1},
		{Doc: "d", Sentence: 1, Question: "multi", Label: "q", Score: 600},
	}
	ac, bc := append([]Answer(nil), a...), append([]Answer(nil), b...)
	got := Compare(a, b)
	want := Diff{Same: 2, LabelDiffers: 1, ScoreDiffers: 1, OnlyA: 1, OnlyB: 1, MaxScoreDelta: 150}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	if !reflect.DeepEqual(a, ac) || !reflect.DeepEqual(b, bc) {
		t.Fatal("inputs mutated")
	}
}
