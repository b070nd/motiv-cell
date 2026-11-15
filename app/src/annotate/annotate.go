package annotate

import (
	"bufio"
	"cmp"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/b070nd/motiv-cell/src/canon"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrNilProvider Error = "annotate: no inner provider"
	ErrNilWriter   Error = "annotate: no writer"
	ErrDrift       Error = "annotate: a sentence has no recorded answers although its document was recorded"
)

const maxScore = 1000

const (
	Yes = "yes"
	No  = "no"
)

type Request struct {
	Doc      string
	Sentence int64
	Mark     string
	Text     string
}

type Answer struct {
	Doc      string
	Sentence int64
	Mark     string
	Question string
	Label    string
	Score    int64
}

type Provider interface {
	Answers(requests []Request) ([]Answer, error)
}

func Write(w io.Writer, answers []Answer) error {
	cw := canon.NewWriter(w)
	for _, a := range answers {
		err := cw.Write(
			canon.Str("doc", a.Doc),
			canon.Int("sentence", a.Sentence),
			canon.Str("mark", a.Mark),
			canon.Str("question", a.Question),
			canon.Str("label", a.Label),
			canon.Int("score", a.Score),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

type record struct {
	Doc      *string `json:"doc"`
	Sentence *int64  `json:"sentence"`
	Mark     *string `json:"mark"`
	Question *string `json:"question"`
	Label    *string `json:"label"`
	Score    *int64  `json:"score"`
}

func lineError(line int, what string) error {
	return Error("annotate: line " + strconv.Itoa(line) + ": " + what)
}

func Read(r io.Reader) ([]Answer, error) {
	br := bufio.NewReader(r)
	var out []Answer
	for line := 1; ; line++ {
		text, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			var rec record
			if jerr := json.Unmarshal([]byte(trimmed), &rec); jerr != nil {
				return nil, lineError(line, "invalid record")
			}
			if rec.Doc == nil || rec.Sentence == nil || rec.Mark == nil || rec.Question == nil || rec.Label == nil || rec.Score == nil {
				return nil, lineError(line, "missing field")
			}
			if *rec.Score < 0 || *rec.Score > maxScore {
				return nil, lineError(line, "score out of range")
			}
			out = append(out, Answer{
				Doc:      *rec.Doc,
				Sentence: *rec.Sentence,
				Mark:     *rec.Mark,
				Question: *rec.Question,
				Label:    *rec.Label,
				Score:    *rec.Score,
			})
		}
		if err == io.EOF {
			return out, nil
		}
	}
}

func compareAnswers(a, b Answer) int {
	return cmp.Or(
		cmp.Compare(a.Doc, b.Doc),
		cmp.Compare(a.Sentence, b.Sentence),
		cmp.Compare(a.Question, b.Question),
		cmp.Compare(a.Label, b.Label),
		cmp.Compare(a.Score, b.Score),
	)
}

func Sort(answers []Answer) {
	slices.SortFunc(answers, compareAnswers)
}

type passage struct {
	doc  string
	mark string
}

type File struct {
	byPassage map[passage][][]Answer
	docs      map[string]bool
}

func NewFile(answers []Answer) *File {
	sorted := slices.Clone(answers)
	Sort(sorted)
	index := make(map[passage][][]Answer)
	docs := make(map[string]bool)
	for i, a := range sorted {
		p := passage{doc: a.Doc, mark: a.Mark}
		docs[a.Doc] = true
		if i > 0 && sorted[i-1].Doc == a.Doc && sorted[i-1].Sentence == a.Sentence && sorted[i-1].Mark == a.Mark {
			last := len(index[p]) - 1
			index[p][last] = append(index[p][last], a)
			continue
		}
		index[p] = append(index[p], []Answer{a})
	}
	return &File{byPassage: index, docs: docs}
}

func (f *File) Answers(requests []Request) ([]Answer, error) {
	var out []Answer
	seen := make(map[passage]int)
	for _, req := range requests {
		p := passage{doc: req.Doc, mark: req.Mark}
		occurrence := seen[p]
		seen[p]++
		stored := f.byPassage[p]
		if occurrence >= len(stored) {
			if f.docs[req.Doc] {
				return nil, ErrDrift
			}
			continue
		}
		for _, a := range stored[occurrence] {
			a.Sentence = req.Sentence
			out = append(out, a)
		}
	}
	return out, nil
}

type Recorder struct {
	inner Provider
	w     io.Writer
}

func NewRecorder(inner Provider, w io.Writer) *Recorder {
	return &Recorder{inner: inner, w: w}
}

func (r *Recorder) Answers(requests []Request) ([]Answer, error) {
	if r.inner == nil {
		return nil, ErrNilProvider
	}
	if r.w == nil {
		return nil, ErrNilWriter
	}
	answers, err := r.inner.Answers(requests)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	if err := Write(&sb, answers); err != nil {
		return nil, err
	}
	if _, err := io.WriteString(r.w, sb.String()); err != nil {
		return nil, err
	}
	return answers, nil
}

type Diff struct {
	Same          int64
	LabelDiffers  int64
	ScoreDiffers  int64
	OnlyA         int64
	OnlyB         int64
	MaxScoreDelta int64
}

type key struct {
	doc      string
	sentence int64
	question string
}

func better(candidate, current Answer) bool {
	if candidate.Score != current.Score {
		return candidate.Score > current.Score
	}
	return candidate.Label < current.Label
}

func best(answers []Answer) map[key]Answer {
	out := make(map[key]Answer, len(answers))
	for _, a := range answers {
		k := key{doc: a.Doc, sentence: a.Sentence, question: a.Question}
		if cur, ok := out[k]; !ok || better(a, cur) {
			out[k] = a
		}
	}
	return out
}

func Compare(a, b []Answer) Diff {
	ma, mb := best(a), best(b)
	var d Diff
	for k, x := range ma {
		y, ok := mb[k]
		if !ok {
			d.OnlyA++
			continue
		}
		delta := x.Score - y.Score
		if delta < 0 {
			delta = -delta
		}
		d.MaxScoreDelta = max(d.MaxScoreDelta, delta)
		switch {
		case x.Label != y.Label:
			d.LabelDiffers++
		case x.Score != y.Score:
			d.ScoreDiffers++
		default:
			d.Same++
		}
	}
	for k := range mb {
		if _, ok := ma[k]; !ok {
			d.OnlyB++
		}
	}
	return d
}
