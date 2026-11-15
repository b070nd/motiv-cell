package laya

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/b070nd/motiv-cell/src/annotate"
)

var testQuestions = []Question{
	{Name: "kind", Kind: Choice, Text: "which kind", Options: []string{"alpha", "beta"}},
	{Name: "urgent", Kind: YesNo, Text: "is it urgent"},
	{Name: "tone", Kind: Score, Text: "how good", Options: []string{"a", "b", "c", "d", "e"}},
}

type fake struct {
	*httptest.Server
	mu      sync.Mutex
	calls   int
	headers []http.Header
	bodies  []wireRequest
}

func newFake(t *testing.T, status int, answer func(text string) map[string]any) *fake {
	t.Helper()
	f := &fake{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req wireRequest
		_ = json.Unmarshal(raw, &req)
		f.mu.Lock()
		f.calls++
		f.headers = append(f.headers, r.Header.Clone())
		f.bodies = append(f.bodies, req)
		f.mu.Unlock()
		if r.URL.Path != batchPath || r.Method != http.MethodPost {
			http.Error(w, "bad", http.StatusNotFound)
			return
		}
		if status != http.StatusOK {
			http.Error(w, "fail", status)
			return
		}
		results := make([]map[string]any, len(req.States))
		for i, s := range req.States {
			results[i] = map[string]any{"answers": answer(s), "usage": map[string]any{"truncated": false}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}))
	t.Cleanup(f.Close)
	return f
}

func canned(text string) map[string]any {
	switch text {
	case "one":
		return map[string]any{
			"kind":   map[string]any{"type": "choice", "choice": "beta", "answer_confidence": 0.8765, "probabilities": map[string]float64{"beta": 0.8765}},
			"urgent": map[string]any{"type": "noul", "noul": 0.2004, "answer_confidence": 0.9},
			"tone":   map[string]any{"type": "score", "score": 3.0, "answer_confidence": 0.7},
		}
	case "two":
		return map[string]any{
			"kind":   map[string]any{"type": "choice", "choice": "gamma", "answer_confidence": 0.9},
			"urgent": map[string]any{"type": "noul", "noul": 0.9996},
			"tone":   map[string]any{"type": "score", "score": "high"},
		}
	}
	return map[string]any{
		"kind":   map[string]any{"type": "choice", "choice": "alpha", "answer_confidence": 1.7},
		"urgent": map[string]any{"type": "noul", "noul": 0.5},
	}
}

func newClient(t *testing.T, url string, mutate func(*Config)) *Client {
	t.Helper()
	cfg := Config{URL: url, Questions: testQuestions}
	if mutate != nil {
		mutate(&cfg)
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func requests() []annotate.Request {
	return []annotate.Request{
		{Doc: "d", Sentence: 0, Text: "one"},
		{Doc: "d", Sentence: 1, Text: "two"},
		{Doc: "e", Sentence: 0, Text: "three"},
	}
}

func TestAnswersAndRounding(t *testing.T) {
	f := newFake(t, http.StatusOK, canned)
	got, err := newClient(t, f.URL, nil).Answers(requests())
	if err != nil {
		t.Fatal(err)
	}
	want := []annotate.Answer{
		{Doc: "d", Sentence: 0, Question: "kind", Label: "beta", Score: 877},
		{Doc: "d", Sentence: 0, Question: "urgent", Label: "no", Score: 800},
		{Doc: "d", Sentence: 0, Question: "tone", Label: "", Score: 750},
		{Doc: "d", Sentence: 1, Question: "urgent", Label: "yes", Score: 1000},
		{Doc: "e", Sentence: 0, Question: "kind", Label: "alpha", Score: 1000},
		{Doc: "e", Sentence: 0, Question: "urgent", Label: "yes", Score: 500},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestBatchingAndAPIKey(t *testing.T) {
	f := newFake(t, http.StatusOK, canned)
	c := newClient(t, f.URL+"/", func(c *Config) { c.BatchSize = 2; c.APIKey = "sekret"; c.Model = "m1" })
	if _, err := c.Answers(requests()); err != nil {
		t.Fatal(err)
	}
	if f.calls != 2 {
		t.Fatalf("calls %d", f.calls)
	}
	if len(f.bodies[0].States) != 2 || len(f.bodies[1].States) != 1 || f.bodies[0].States[0] != "one" || f.bodies[1].States[0] != "three" {
		t.Fatalf("bodies %+v", f.bodies)
	}
	if f.bodies[0].Model != "m1" || len(f.bodies[0].Questions) != 3 {
		t.Fatalf("body %+v", f.bodies[0])
	}
	if f.bodies[0].Questions["urgent"].Type != "noul" || f.bodies[0].Questions["tone"].Type != "score" {
		t.Fatalf("questions %+v", f.bodies[0].Questions)
	}
	for _, h := range f.headers {
		if h.Get("Authorization") != "Bearer sekret" {
			t.Fatalf("header %v", h)
		}
	}
}

func TestNoAPIKeyHeader(t *testing.T) {
	f := newFake(t, http.StatusOK, canned)
	if _, err := newClient(t, f.URL, nil).Answers(requests()[:1]); err != nil {
		t.Fatal(err)
	}
	if v := f.headers[0].Get("Authorization"); v != "" {
		t.Fatalf("header %q", v)
	}
}

func TestServerError(t *testing.T) {
	f := newFake(t, http.StatusInternalServerError, canned)
	got, err := newClient(t, f.URL, nil).Answers(requests())
	if err == nil || !strings.Contains(err.Error(), "500") || got != nil {
		t.Fatalf("got %v %v", got, err)
	}
}

func TestNewValidation(t *testing.T) {
	for _, u := range []string{"", "ftp://x", "http://", "nonsense"} {
		if _, err := New(Config{URL: u, Questions: testQuestions}); err != ErrURL {
			t.Fatalf("%q: %v", u, err)
		}
	}
	if _, err := New(Config{URL: "http://x", Questions: []Question{{Name: "c", Kind: Choice}}}); err != ErrQuestion {
		t.Fatal(err)
	}
}

func TestRecordAndReplay(t *testing.T) {
	f := newFake(t, http.StatusOK, canned)
	var buf bytes.Buffer
	rec := annotate.NewRecorder(newClient(t, f.URL, nil), &buf)
	live, err := rec.Answers(requests())
	if err != nil {
		t.Fatal(err)
	}
	read, err := annotate.Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := annotate.NewFile(read).Answers(requests())
	if err != nil {
		t.Fatal(err)
	}
	annotate.Sort(live)
	if !reflect.DeepEqual(live, replayed) {
		t.Fatalf("live %+v replayed %+v", live, replayed)
	}
	d := annotate.Compare(live, replayed)
	if d != (annotate.Diff{Same: int64(len(live))}) {
		t.Fatalf("diff %+v", d)
	}
}
