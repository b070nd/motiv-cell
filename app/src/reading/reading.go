package reading

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/b070nd/motiv-cell/src/model"
	"github.com/b070nd/motiv-cell/src/segment"
)

type Sentence struct {
	Index int64
	Text  string
	Mark  string
}

func Sentences(doc model.Doc) []Sentence {
	var texts []string
	if doc.Headline != "" {
		texts = append(texts, doc.Headline)
	}
	texts = append(texts, segment.Sentences(doc.Body)...)
	out := make([]Sentence, len(texts))
	for i, text := range texts {
		out[i] = Sentence{Index: int64(i), Text: text, Mark: Mark(text)}
	}
	return out
}

func Mark(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}
