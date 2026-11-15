package newsfeed

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/b070nd/motiv-cell/src/model"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrID        Error = "newsfeed: missing id"
	ErrTimestamp Error = "newsfeed: ts is not RFC 3339"
	ErrDuplicate Error = "newsfeed: duplicate id"
)

type LineError struct {
	Line int
	Err  error
}

func (e *LineError) Error() string {
	return "newsfeed: line " + strconv.Itoa(e.Line) + ": " + e.Err.Error()
}

func (e *LineError) Unwrap() error {
	return e.Err
}

type record struct {
	ID       string `json:"id"`
	TS       string `json:"ts"`
	Source   string `json:"source"`
	Headline string `json:"headline"`
	Body     string `json:"body"`
}

func decode(text string, seen map[string]struct{}) (model.Doc, error) {
	var rec record
	if err := json.Unmarshal([]byte(text), &rec); err != nil {
		return model.Doc{}, err
	}
	if rec.ID == "" {
		return model.Doc{}, ErrID
	}
	t, err := time.Parse(time.RFC3339Nano, rec.TS)
	if err != nil {
		return model.Doc{}, ErrTimestamp
	}
	if _, ok := seen[rec.ID]; ok {
		return model.Doc{}, ErrDuplicate
	}
	seen[rec.ID] = struct{}{}
	return model.Doc{
		ID:       rec.ID,
		Source:   rec.Source,
		Headline: rec.Headline,
		Body:     rec.Body,
		At:       t.UnixNano(),
	}, nil
}

func Read(r io.Reader) ([]model.Doc, error) {
	reader := bufio.NewReader(r)
	seen := make(map[string]struct{})
	var docs []model.Doc
	line := 0
	for {
		text, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}
		if readErr == io.EOF && text == "" {
			return docs, nil
		}
		line++
		if strings.TrimSpace(text) != "" {
			doc, err := decode(text, seen)
			if err != nil {
				return nil, &LineError{Line: line, Err: err}
			}
			docs = append(docs, doc)
		}
		if readErr == io.EOF {
			return docs, nil
		}
	}
}
