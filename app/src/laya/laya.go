package laya

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/b070nd/motiv-cell/src/annotate"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrURL        Error = "laya: the url must be http or https with a host"
	ErrQuestion   Error = "laya: a question has no name, a duplicate name, or options it needs"
	ErrNoQuestion Error = "laya: no questions configured"
	ErrRequest    Error = "laya: request failed"
	ErrReply      Error = "laya: the reply is not valid JSON of the expected shape"
	ErrCount      Error = "laya: the reply has a different number of results than states"
	ErrTooLarge   Error = "laya: the reply is larger than 8 MiB"
)

const (
	batchPath        = "/v1/systemone/batch"
	defaultBatchSize = 16
	maxReplyBytes    = 8 << 20
	perMille         = 1000
	typeNoul         = "noul"
	typeChoice       = "choice"
	typeScore        = "score"
	labelYes         = "yes"
	labelNo          = "no"
)

type QuestionKind uint8

const (
	Choice QuestionKind = iota
	Score
	YesNo
)

type Question struct {
	Name    string
	Kind    QuestionKind
	Text    string
	Options []string
}

type Config struct {
	URL       string
	APIKey    string
	Model     string
	Questions []Question
	BatchSize int
	HTTP      *http.Client
}

type Client struct {
	endpoint  string
	apiKey    string
	model     string
	questions []Question
	wire      map[string]wireQuestion
	batchSize int
	http      *http.Client
}

type wireQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type wireRequest struct {
	States    []string                `json:"states"`
	Questions map[string]wireQuestion `json:"questions"`
	Model     string                  `json:"model,omitempty"`
}

type wireAnswer struct {
	Type             string   `json:"type"`
	Noul             *float64 `json:"noul"`
	Choice           string   `json:"choice"`
	Score            *float64 `json:"score"`
	AnswerConfidence *float64 `json:"answer_confidence"`
}

type wireResult struct {
	Answers map[string]json.RawMessage `json:"answers"`
}

type wireResponse struct {
	Results []wireResult `json:"results"`
}

func New(config Config) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(config.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return nil, ErrURL
	}
	if len(config.Questions) == 0 {
		return nil, ErrNoQuestion
	}
	wire := make(map[string]wireQuestion, len(config.Questions))
	for _, q := range config.Questions {
		wq, ok := buildQuestion(q)
		if !ok {
			return nil, ErrQuestion
		}
		if _, dup := wire[q.Name]; dup {
			return nil, ErrQuestion
		}
		wire[q.Name] = wq
	}
	c := &Client{
		endpoint:  strings.TrimRight(u.String(), "/") + batchPath,
		apiKey:    config.APIKey,
		model:     config.Model,
		questions: append([]Question(nil), config.Questions...),
		wire:      wire,
		batchSize: config.BatchSize,
		http:      config.HTTP,
	}
	if c.batchSize <= 0 {
		c.batchSize = defaultBatchSize
	}
	if c.http == nil {
		c.http = &http.Client{}
	}
	return c, nil
}

func buildQuestion(q Question) (wireQuestion, bool) {
	if q.Name == "" {
		return wireQuestion{}, false
	}
	switch q.Kind {
	case Choice:
		if len(q.Options) == 0 {
			return wireQuestion{}, false
		}
		criteria := make(map[string]string, len(q.Options))
		for _, o := range q.Options {
			criteria[o] = o
		}
		return wireQuestion{Type: typeChoice, Instructions: q.Text, Criteria: criteria}, true
	case Score:
		if len(q.Options) < 2 {
			return wireQuestion{}, false
		}
		return wireQuestion{Type: typeScore, Instructions: q.Text, Criteria: append([]string(nil), q.Options...)}, true
	case YesNo:
		return wireQuestion{Type: typeNoul, Instructions: q.Text}, true
	}
	return wireQuestion{}, false
}

func (c *Client) Answers(requests []annotate.Request) ([]annotate.Answer, error) {
	return c.AnswersContext(context.Background(), requests)
}

func (c *Client) AnswersContext(ctx context.Context, requests []annotate.Request) ([]annotate.Answer, error) {
	var out []annotate.Answer
	for start := 0; start < len(requests); start += c.batchSize {
		batch := requests[start:min(start+c.batchSize, len(requests))]
		results, err := c.send(ctx, batch)
		if err != nil {
			return nil, err
		}
		for i, req := range batch {
			for _, q := range c.questions {
				raw, ok := results[i].Answers[q.Name]
				if !ok {
					continue
				}
				label, score, ok := convert(q, raw)
				if !ok {
					continue
				}
				out = append(out, annotate.Answer{
					Doc:      req.Doc,
					Sentence: req.Sentence,
					Question: q.Name,
					Label:    label,
					Score:    score,
				})
			}
		}
	}
	return out, nil
}

func (c *Client) send(ctx context.Context, batch []annotate.Request) ([]wireResult, error) {
	states := make([]string, len(batch))
	for i, r := range batch {
		states[i] = r.Text
	}
	body, err := json.Marshal(wireRequest{States: states, Questions: c.wire, Model: c.model})
	if err != nil {
		return nil, ErrRequest
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, ErrRequest
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if cerr := ctx.Err(); cerr != nil {
			return nil, cerr
		}
		return nil, ErrRequest
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return nil, Error("laya: request failed: status " + strconv.Itoa(resp.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReplyBytes+1))
	if err != nil {
		if cerr := ctx.Err(); cerr != nil {
			return nil, cerr
		}
		return nil, ErrRequest
	}
	if len(data) > maxReplyBytes {
		return nil, ErrTooLarge
	}
	var parsed wireResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, ErrReply
	}
	if len(parsed.Results) != len(batch) {
		return nil, ErrCount
	}
	return parsed.Results, nil
}

func permille(p float64) int64 {
	if !(p > 0) {
		return 0
	}
	if p > 1 {
		p = 1
	}
	v := int64(p*perMille + 0.5)
	return min(max(v, 0), perMille)
}

func convert(q Question, raw json.RawMessage) (string, int64, bool) {
	var a wireAnswer
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", 0, false
	}
	switch q.Kind {
	case Choice:
		if a.Type != typeChoice || a.AnswerConfidence == nil || !contains(q.Options, a.Choice) {
			return "", 0, false
		}
		return a.Choice, permille(*a.AnswerConfidence), true
	case YesNo:
		if a.Type != typeNoul || a.Noul == nil {
			return "", 0, false
		}
		p := *a.Noul
		if p >= 0.5 {
			return labelYes, permille(p), true
		}
		return labelNo, permille(1 - p), true
	case Score:
		if a.Type != typeScore || a.Score == nil || len(q.Options) < 2 {
			return "", 0, false
		}
		return "", permille(*a.Score / float64(len(q.Options)-1)), true
	}
	return "", 0, false
}

func contains(options []string, s string) bool {
	for _, o := range options {
		if o == s {
			return true
		}
	}
	return false
}
