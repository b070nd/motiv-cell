package evidence

import (
	"encoding/json"
	"io"
	"slices"

	"github.com/b070nd/motiv-cell/src/annotate"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrFile       Error = "evidence: questions file is not valid"
	ErrName       Error = "evidence: question name is empty or repeated"
	ErrKind       Error = "evidence: question kind must be choice, yesno or score"
	ErrVoid       Error = "evidence: a choice question named void is required"
	ErrNone       Error = "evidence: the none label is not an option of the void question"
	ErrIntensity  Error = "evidence: a score question named intensity with at least two levels is required"
	ErrPursuit    Error = "evidence: the pursuit question must be yesno"
	ErrThreshold  Error = "evidence: pursuit_min must be between 0 and 1000"
	ErrCapability Error = "evidence: the capability question must be a choice with at least two options"
)

const (
	KindChoice = "choice"
	KindYesNo  = "yesno"
	KindScore  = "score"
)

const (
	RoleVoid      = "void"
	RolePursuit   = "pursuit"
	RoleIntensity = "intensity"
	// RoleCapability asks whether the text presents the actor as able to get
	// what it wants. Its answers are validated and recorded but not yet read
	// into findings, so they change no cell.
	RoleCapability = "capability"
)

const scale = 1000

type Question struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Text    string   `json:"text"`
	Options []string `json:"options"`
}

type Spec struct {
	Model      string     `json:"model"`
	None       string     `json:"none"`
	PursuitMin int64      `json:"pursuit_min"`
	Questions  []Question `json:"questions"`
}

type Finding struct {
	Void      string
	Intensity int64
}

func Parse(r io.Reader) (Spec, error) {
	var spec Spec
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, ErrFile
	}
	if err := spec.Validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func (s Spec) Validate() error {
	var void, intensity bool
	for i, q := range s.Questions {
		if q.Name == "" {
			return ErrName
		}
		for _, earlier := range s.Questions[:i] {
			if earlier.Name == q.Name {
				return ErrName
			}
		}
		if q.Kind != KindChoice && q.Kind != KindYesNo && q.Kind != KindScore {
			return ErrKind
		}
		switch q.Name {
		case RoleVoid:
			if q.Kind != KindChoice || len(q.Options) == 0 {
				return ErrVoid
			}
			if s.None != "" && !slices.Contains(q.Options, s.None) {
				return ErrNone
			}
			void = true
		case RoleIntensity:
			if q.Kind != KindScore || len(q.Options) < 2 {
				return ErrIntensity
			}
			intensity = true
		case RolePursuit:
			if q.Kind != KindYesNo {
				return ErrPursuit
			}
		case RoleCapability:
			if q.Kind != KindChoice || len(q.Options) < 2 {
				return ErrCapability
			}
		}
	}
	if !void {
		return ErrVoid
	}
	if !intensity {
		return ErrIntensity
	}
	if s.PursuitMin < 0 || s.PursuitMin > scale {
		return ErrThreshold
	}
	return nil
}

func (s Spec) Read(answers []annotate.Answer) (Finding, bool) {
	var finding Finding
	var hasVoid, hasIntensity bool
	for _, answer := range answers {
		switch answer.Question {
		case RoleVoid:
			finding.Void, hasVoid = answer.Label, true
		case RoleIntensity:
			finding.Intensity, hasIntensity = answer.Score, true
		case RolePursuit:
			yes := answer.Score
			if answer.Label != annotate.Yes {
				yes = scale - answer.Score
			}
			if yes < s.PursuitMin {
				return Finding{}, false
			}
		}
	}
	if !hasVoid || !hasIntensity || finding.Void == s.None {
		return Finding{}, false
	}
	return finding, true
}
