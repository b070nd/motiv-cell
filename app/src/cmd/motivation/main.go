package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/b070nd/motiv-cell/src/annotate"
	"github.com/b070nd/motiv-cell/src/laya"
	"github.com/b070nd/motiv-cell/src/newsfeed"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	errUsage    Error = "usage: motivation run|compare [flags]"
	errProvider Error = "run: give exactly one of -laya or -answers"
	errKind     Error = "questions: kind must be choice, yesno or score"
)

type questionFile struct {
	Model      string `json:"model"`
	None       string `json:"none"`
	PursuitMin int64  `json:"pursuit_min"`
	Questions  []struct {
		Name    string   `json:"name"`
		Kind    string   `json:"kind"`
		Text    string   `json:"text"`
		Options []string `json:"options"`
	} `json:"questions"`
}

func main() {
	if err := dispatch(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "motivation:", err)
		os.Exit(1)
	}
}

func dispatch(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errUsage
	}
	switch args[0] {
	case "run":
		return run(args[1:], stdout)
	case "compare":
		return compare(args[1:], stdout)
	}
	return errUsage
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	newsPath := flags.String("news", "", "news JSONL file")
	questionsPath := flags.String("questions", "", "questions JSON file")
	layaURL := flags.String("laya", "", "laya-serve base URL (live mode)")
	answersPath := flags.String("answers", "", "recorded answers JSONL (file mode)")
	recordPath := flags.String("record", "", "write live answers to this JSONL file")
	outPath := flags.String("out", "", "result JSONL file (default stdout)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if (*layaURL == "") == (*answersPath == "") {
		return errProvider
	}
	newsFile, err := os.Open(*newsPath)
	if err != nil {
		return err
	}
	docs, err := newsfeed.Read(newsFile)
	newsFile.Close()
	if err != nil {
		return err
	}
	questions, err := loadQuestions(*questionsPath)
	if err != nil {
		return err
	}
	var provider annotate.Provider
	var recorded bytes.Buffer
	if *answersPath != "" {
		data, err := os.ReadFile(*answersPath)
		if err != nil {
			return err
		}
		stored, err := annotate.Read(bytes.NewReader(data))
		if err != nil {
			return err
		}
		provider = annotate.NewFile(stored)
	} else {
		config, err := layaConfig(questions, *layaURL)
		if err != nil {
			return err
		}
		client, err := laya.New(config)
		if err != nil {
			return err
		}
		provider = client
		if *recordPath != "" {
			provider = annotate.NewRecorder(client, &recorded)
		}
	}
	answers, err := provider.Answers(requests(ordered(docs)))
	if err != nil {
		return err
	}
	if *recordPath != "" && *layaURL != "" {
		if err := os.WriteFile(*recordPath, recorded.Bytes(), 0o644); err != nil {
			return err
		}
	}
	var result bytes.Buffer
	if _, err := simulate(docs, answers, questions.None, questions.PursuitMin, &result); err != nil {
		return err
	}
	if *outPath == "" {
		_, err = stdout.Write(result.Bytes())
		return err
	}
	return os.WriteFile(*outPath, result.Bytes(), 0o644)
}

func compare(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("compare", flag.ContinueOnError)
	aPath := flags.String("a", "", "first answers JSONL")
	bPath := flags.String("b", "", "second answers JSONL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	a, err := readAnswers(*aPath)
	if err != nil {
		return err
	}
	b, err := readAnswers(*bPath)
	if err != nil {
		return err
	}
	diff := annotate.Compare(a, b)
	_, err = fmt.Fprintf(stdout, "same %d\nlabel differs %d\nscore differs %d\nonly a %d\nonly b %d\nmax score delta %d\n",
		diff.Same, diff.LabelDiffers, diff.ScoreDiffers, diff.OnlyA, diff.OnlyB, diff.MaxScoreDelta)
	return err
}

func readAnswers(path string) ([]annotate.Answer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return annotate.Read(bytes.NewReader(data))
}

func loadQuestions(path string) (questionFile, error) {
	var file questionFile
	data, err := os.ReadFile(path)
	if err != nil {
		return file, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return file, errors.Join(Error("questions: invalid file"), err)
	}
	return file, nil
}

func layaConfig(file questionFile, url string) (laya.Config, error) {
	config := laya.Config{URL: url, Model: file.Model, APIKey: os.Getenv("LAYA_API_KEY")}
	for _, q := range file.Questions {
		var kind laya.QuestionKind
		switch strings.ToLower(q.Kind) {
		case "choice":
			kind = laya.Choice
		case "yesno":
			kind = laya.YesNo
		case "score":
			kind = laya.Score
		default:
			return config, errKind
		}
		config.Questions = append(config.Questions, laya.Question{Name: q.Name, Kind: kind, Text: q.Text, Options: q.Options})
	}
	return config, nil
}
