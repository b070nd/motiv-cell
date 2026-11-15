package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/b070nd/motiv-cell/src/actors"
	"github.com/b070nd/motiv-cell/src/annotate"
	"github.com/b070nd/motiv-cell/src/evidence"
	"github.com/b070nd/motiv-cell/src/laya"
	"github.com/b070nd/motiv-cell/src/narrative"
	"github.com/b070nd/motiv-cell/src/newsfeed"
	"github.com/b070nd/motiv-cell/src/report"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	errUsage    Error = "usage: motivation run|compare|version [flags]"
	errProvider Error = "run: give exactly one of -laya or -answers"
	errRecord   Error = "run: -record needs -laya"
)

var version = "dev"

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
	case "version":
		_, err := fmt.Fprintln(stdout, version)
		return err
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
	actorsPath := flags.String("actors", "", "actors gazetteer JSONL (optional)")
	outPath := flags.String("out", "", "result JSONL file (default stdout)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if (*layaURL == "") == (*answersPath == "") {
		return errProvider
	}
	if *recordPath != "" && *layaURL == "" {
		return errRecord
	}
	newsData, err := os.ReadFile(*newsPath)
	if err != nil {
		return err
	}
	docs, err := newsfeed.Read(bytes.NewReader(newsData))
	if err != nil {
		return err
	}
	questionsData, err := os.ReadFile(*questionsPath)
	if err != nil {
		return err
	}
	spec, err := evidence.Parse(bytes.NewReader(questionsData))
	if err != nil {
		return err
	}
	var recorded bytes.Buffer
	provider, err := provider(spec, *layaURL, *answersPath, *recordPath != "", &recorded)
	if err != nil {
		return err
	}
	var gazetteer *actors.Gazetteer
	if *actorsPath != "" {
		actorsData, err := os.ReadFile(*actorsPath)
		if err != nil {
			return err
		}
		if gazetteer, err = actors.Parse(bytes.NewReader(actorsData)); err != nil {
			return err
		}
	}
	result, err := narrative.Build(docs, provider, spec, gazetteer)
	if err != nil {
		return err
	}
	if *recordPath != "" {
		if err := os.WriteFile(*recordPath, recorded.Bytes(), 0o644); err != nil {
			return err
		}
	}
	var out bytes.Buffer
	if err := report.Write(&out, result); err != nil {
		return err
	}
	if *outPath == "" {
		_, err = stdout.Write(out.Bytes())
		return err
	}
	return os.WriteFile(*outPath, out.Bytes(), 0o644)
}

func provider(spec evidence.Spec, layaURL, answersPath string, record bool, recorded io.Writer) (annotate.Provider, error) {
	if answersPath != "" {
		stored, err := readAnswers(answersPath)
		if err != nil {
			return nil, err
		}
		return annotate.NewFile(stored), nil
	}
	config := laya.Config{URL: layaURL, Model: spec.Model, APIKey: os.Getenv("LAYA_API_KEY")}
	for _, q := range spec.Questions {
		kind := laya.Choice
		switch q.Kind {
		case evidence.KindYesNo:
			kind = laya.YesNo
		case evidence.KindScore:
			kind = laya.Score
		}
		config.Questions = append(config.Questions, laya.Question{Name: q.Name, Kind: kind, Text: q.Text, Options: q.Options})
	}
	client, err := laya.New(config)
	if err != nil {
		return nil, err
	}
	if record {
		return annotate.NewRecorder(client, recorded), nil
	}
	return client, nil
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
