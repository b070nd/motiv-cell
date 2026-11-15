package newsfeed

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReadValid(t *testing.T) {
	input := `{"id":"a","ts":"2024-01-02T03:04:05.5Z","source":"s1","headline":"h1","body":"b1","extra":1}
{"id":"b","ts":"2024-01-02T05:04:05+02:00","source":"s2","headline":"h2","body":"b2"}
`
	docs, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs, want 2", len(docs))
	}
	wantA := time.Date(2024, 1, 2, 3, 4, 5, 500000000, time.UTC).UnixNano()
	wantB := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC).UnixNano()
	if docs[0].ID != "a" || docs[0].Source != "s1" || docs[0].Headline != "h1" || docs[0].Body != "b1" || docs[0].At != wantA {
		t.Fatalf("doc 0 = %+v, want At %d", docs[0], wantA)
	}
	if docs[1].ID != "b" || docs[1].Source != "s2" || docs[1].Headline != "h2" || docs[1].Body != "b2" || docs[1].At != wantB {
		t.Fatalf("doc 1 = %+v, want At %d", docs[1], wantB)
	}
}

func TestReadBlankLineNumbering(t *testing.T) {
	input := "{\"id\":\"a\",\"ts\":\"2024-01-02T03:04:05Z\"}\n \t \n{\"id\":\"b\",\"ts\":\"2024-01-02T03:04:05Z\"}\n\n{\"ts\":\"2024-01-02T03:04:05Z\"}\n"
	docs, err := Read(strings.NewReader(input))
	if !errors.Is(err, ErrID) {
		t.Fatalf("got %v, want ErrID", err)
	}
	var le *LineError
	if !errors.As(err, &le) || le.Line != 5 {
		t.Fatalf("got %v, want LineError at line 5", err)
	}
	if err.Error() != "newsfeed: line 5: newsfeed: missing id" {
		t.Fatalf("got message %q", err.Error())
	}
	if docs != nil {
		t.Fatalf("got docs %v on error, want nil", docs)
	}
}

func TestReadErrors(t *testing.T) {
	valid := `{"id":"a","ts":"2024-01-02T03:04:05Z"}`
	cases := []struct {
		name  string
		input string
		want  error
		line  int
	}{
		{"missing id", `{"ts":"2024-01-02T03:04:05Z"}`, ErrID, 1},
		{"empty id", `{"id":"","ts":"2024-01-02T03:04:05Z"}`, ErrID, 1},
		{"date only ts", `{"id":"a","ts":"2024-01-02"}`, ErrTimestamp, 1},
		{"ts without zone", `{"id":"a","ts":"2024-01-02T03:04:05"}`, ErrTimestamp, 1},
		{"duplicate id", valid + "\n" + valid, ErrDuplicate, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			docs, err := Read(strings.NewReader(c.input))
			if !errors.Is(err, c.want) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
			var le *LineError
			if !errors.As(err, &le) {
				t.Fatalf("got %T, want *LineError", err)
			}
			if le.Line != c.line {
				t.Fatalf("got line %d, want %d", le.Line, c.line)
			}
			if docs != nil {
				t.Fatalf("got docs %v on error, want nil", docs)
			}
		})
	}
}

func TestReadMalformedJSON(t *testing.T) {
	docs, err := Read(strings.NewReader("{\"id\":\n"))
	var le *LineError
	if !errors.As(err, &le) {
		t.Fatalf("got %v, want *LineError", err)
	}
	if le.Line != 1 || le.Err == nil || errors.Unwrap(le) == nil {
		t.Fatalf("got line %d, err %v", le.Line, le.Err)
	}
	if docs != nil {
		t.Fatalf("got docs %v on error, want nil", docs)
	}
}
