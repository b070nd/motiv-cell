package canon

import (
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrKey       Error = "canon: empty key"
	ErrDuplicate Error = "canon: duplicate key"
	ErrText      Error = "canon: invalid UTF-8"
)

const hexDigits = "0123456789abcdef"

type Field struct {
	key   string
	str   string
	num   int64
	isStr bool
}

func Int(key string, value int64) Field {
	return Field{key: key, num: value}
}

func Str(key string, value string) Field {
	return Field{key: key, str: value, isStr: true}
}

func Append(dst []byte, fields []Field) ([]byte, error) {
	for i := range fields {
		if fields[i].key == "" {
			return dst, ErrKey
		}
	}
	for i := range fields {
		if !utf8.ValidString(fields[i].key) {
			return dst, ErrText
		}
		if fields[i].isStr && !utf8.ValidString(fields[i].str) {
			return dst, ErrText
		}
	}
	sorted := slices.Clone(fields)
	slices.SortFunc(sorted, func(a, b Field) int {
		return strings.Compare(a.key, b.key)
	})
	for i := 1; i < len(sorted); i++ {
		if sorted[i-1].key == sorted[i].key {
			return dst, ErrDuplicate
		}
	}
	dst = append(dst, '{')
	for i, f := range sorted {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendString(dst, f.key)
		dst = append(dst, ':')
		if f.isStr {
			dst = appendString(dst, f.str)
		} else {
			dst = strconv.AppendInt(dst, f.num, 10)
		}
	}
	dst = append(dst, '}')
	return dst, nil
}

func appendString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			dst = append(dst, '\\', '"')
		case c == '\\':
			dst = append(dst, '\\', '\\')
		case c == '\b':
			dst = append(dst, '\\', 'b')
		case c == '\t':
			dst = append(dst, '\\', 't')
		case c == '\n':
			dst = append(dst, '\\', 'n')
		case c == '\f':
			dst = append(dst, '\\', 'f')
		case c == '\r':
			dst = append(dst, '\\', 'r')
		case c < 0x20:
			dst = append(dst, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
		default:
			dst = append(dst, c)
		}
	}
	return append(dst, '"')
}

type Writer struct {
	w   io.Writer
	buf []byte
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) Write(fields ...Field) error {
	line, err := Append(w.buf[:0], fields)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	w.buf = line
	n, err := w.w.Write(line)
	if err != nil {
		return err
	}
	if n < len(line) {
		return io.ErrShortWrite
	}
	return nil
}
