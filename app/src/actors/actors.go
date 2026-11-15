package actors

import (
	"bufio"
	"cmp"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrID        Error = "actors: missing id"
	ErrDuplicate Error = "actors: duplicate id"
	ErrName      Error = "actors: actor has no name"
	ErrRecord    Error = "actors: invalid record"
	ErrAmbiguous Error = "actors: the same name belongs to two actors"
)

type LineError struct {
	Line int
	Err  error
}

func (e *LineError) Error() string {
	return "actors: line " + strconv.Itoa(e.Line) + ": " + e.Err.Error()
}

func (e *LineError) Unwrap() error { return e.Err }

type Actor struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
	Tickers []string `json:"tickers"`
}

type surface struct {
	text string
	id   string
}

type Gazetteer struct {
	surfaces []surface
}

func Parse(r io.Reader) (*Gazetteer, error) {
	reader := bufio.NewReader(r)
	var list []Actor
	for line := 1; ; line++ {
		text, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			var actor Actor
			decoder := json.NewDecoder(strings.NewReader(trimmed))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&actor) != nil {
				return nil, &LineError{Line: line, Err: ErrRecord}
			}
			list = append(list, actor)
			if _, failure := New(list); failure != nil {
				return nil, &LineError{Line: line, Err: failure}
			}
		}
		if err == io.EOF {
			return New(list)
		}
	}
}

func New(list []Actor) (*Gazetteer, error) {
	var surfaces []surface
	for i, actor := range list {
		if actor.ID == "" {
			return nil, ErrID
		}
		for _, earlier := range list[:i] {
			if earlier.ID == actor.ID {
				return nil, ErrDuplicate
			}
		}
		if strings.TrimSpace(actor.Name) == "" {
			return nil, ErrName
		}
		names := append([]string{actor.Name}, actor.Aliases...)
		names = append(names, actor.Tickers...)
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			surfaces = append(surfaces, surface{text: name, id: actor.ID})
		}
	}
	slices.SortStableFunc(surfaces, func(a, b surface) int {
		return cmp.Or(cmp.Compare(len(b.text), len(a.text)), cmp.Compare(a.text, b.text), cmp.Compare(a.id, b.id))
	})
	kept := surfaces[:0]
	for i, s := range surfaces {
		if i > 0 && surfaces[i-1].text == s.text {
			if surfaces[i-1].id != s.id {
				return nil, ErrAmbiguous
			}
			continue
		}
		kept = append(kept, s)
	}
	return &Gazetteer{surfaces: kept}, nil
}

func (g *Gazetteer) Find(text string) []string {
	if g == nil {
		return nil
	}
	var found []string
	for at := 0; at < len(text); {
		r, size := utf8.DecodeRuneInString(text[at:])
		if !wordRune(r) || (at > 0 && wordBefore(text, at)) {
			at += size
			continue
		}
		matched := 0
		for _, s := range g.surfaces {
			if !strings.HasPrefix(text[at:], s.text) {
				continue
			}
			end := at + len(s.text)
			if end < len(text) {
				next, _ := utf8.DecodeRuneInString(text[end:])
				if wordRune(next) {
					continue
				}
			}
			if !slices.Contains(found, s.id) {
				found = append(found, s.id)
			}
			matched = len(s.text)
			break
		}
		if matched > 0 {
			at += matched
			continue
		}
		at += size
	}
	return found
}

func wordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func wordBefore(text string, at int) bool {
	r, _ := utf8.DecodeLastRuneInString(text[:at])
	return wordRune(r)
}
