package canon

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"testing"
)

func TestAppend(t *testing.T) {
	tests := []struct {
		name   string
		fields []Field
		want   string
	}{
		{"empty", nil, `{}`},
		{"sorted", []Field{Int("b", 1), Str("a", "x"), Int("é", 2), Int("B", 3)}, `{"B":3,"a":"x","b":1,"é":2}`},
		{"ints", []Field{Int("neg", -42), Int("min", math.MinInt64), Int("max", math.MaxInt64), Int("zero", 0)}, `{"max":9223372036854775807,"min":-9223372036854775808,"neg":-42,"zero":0}`},
		{"quote", []Field{Str("k", `a"b`)}, `{"k":"a\"b"}`},
		{"backslash", []Field{Str("k", `a\b`)}, `{"k":"a\\b"}`},
		{"short escapes", []Field{Str("k", "\b\t\n\f\r")}, `{"k":"\b\t\n\f\r"}`},
		{"control hex", []Field{Str("k", "\x00\x01\x0b\x0e\x1f")}, `{"k":"\u0000\u0001\u000b\u000e\u001f"}`},
		{"unescaped", []Field{Str("k", "/\x7f  ~")}, "{\"k\":\"/\x7f  ~\"}"},
		{"multibyte", []Field{Str("k", "héllo 世界 😀")}, `{"k":"héllo 世界 😀"}`},
		{"key escaped", []Field{Int("a\"\n", 1)}, `{"a\"\n":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Append(nil, tt.fields)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestAppendErrors(t *testing.T) {
	tests := []struct {
		name   string
		fields []Field
		want   error
	}{
		{"empty key", []Field{Int("", 1)}, ErrKey},
		{"duplicate", []Field{Int("a", 1), Str("b", "x"), Str("a", "y")}, ErrDuplicate},
		{"bad value", []Field{Str("a", "\xff")}, ErrText},
		{"bad key", []Field{Int("\xc3", 1)}, ErrText},
		{"key beats text", []Field{Str("a", "\xff"), Int("", 1)}, ErrKey},
		{"text beats duplicate", []Field{Int("a", 1), Int("a", 2), Str("b", "\xff")}, ErrText},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := []byte("keep")
			got, err := Append(dst, tt.fields)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err %v want %v", err, tt.want)
			}
			if string(got) != "keep" || len(got) != len(dst) {
				t.Fatalf("dst changed: %q", got)
			}
		})
	}
}

type countWriter struct {
	buf   bytes.Buffer
	calls int
	short bool
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.calls++
	if c.short {
		return len(p) - 1, nil
	}
	return c.buf.Write(p)
}

func TestWriter(t *testing.T) {
	cw := &countWriter{}
	w := NewWriter(cw)
	if err := w.Write(Int("b", 1), Int("a", 2)); err != nil {
		t.Fatal(err)
	}
	if err := w.Write(); err != nil {
		t.Fatal(err)
	}
	if err := w.Write(Str("s", "x")); err != nil {
		t.Fatal(err)
	}
	if cw.calls != 3 {
		t.Fatalf("calls %d", cw.calls)
	}
	want := "{\"a\":2,\"b\":1}\n{}\n{\"s\":\"x\"}\n"
	if cw.buf.String() != want {
		t.Fatalf("got %q want %q", cw.buf.String(), want)
	}
	if err := w.Write(Int("", 1)); !errors.Is(err, ErrKey) {
		t.Fatalf("err %v", err)
	}
	if cw.calls != 3 || cw.buf.String() != want {
		t.Fatal("write on error")
	}
}

func TestWriterShort(t *testing.T) {
	w := NewWriter(&countWriter{short: true})
	if err := w.Write(Int("a", 1)); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("err %v", err)
	}
}

func TestWriterKeepsOrder(t *testing.T) {
	fields := []Field{Int("c", 1), Int("a", 2), Int("b", 3)}
	orig := append([]Field(nil), fields...)
	if err := NewWriter(io.Discard).Write(fields...); err != nil {
		t.Fatal(err)
	}
	for i := range fields {
		if fields[i] != orig[i] {
			t.Fatalf("order changed at %d", i)
		}
	}
}

func TestValidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := NewWriter(&buf).Write(Str("s", "a\"\\\x00\n é"), Int("n", math.MinInt64))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSuffix(buf.Bytes(), []byte("\n")), &m); err != nil {
		t.Fatal(err)
	}
	if m["s"] != "a\"\\\x00\n é" {
		t.Fatalf("got %q", m["s"])
	}
}
