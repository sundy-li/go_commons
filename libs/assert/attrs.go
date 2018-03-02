package assert

import (
	"fmt"
	"io"
	"strings"
)

const (
	Reset = "0"

	Bold      = "1"
	Underline = "4"
	Blink     = "5"

	Red    = "31"
	Green  = "32"
	Yellow = "33"
	Blue   = "34"
	Purple = "35"
	Cyan   = "36"
	Grey   = "37"
)

type fancyWriter struct {
	writer io.Writer
	attrs  string
}

func (w fancyWriter) Write(p []byte) (n int, err error) {
	return fmt.Fprintf(w.writer, "%c[%sm%s%c[0m", 27, w.attrs, p, 27)
}

func NewWriter(writer io.Writer, attrs ...string) io.Writer {
	return fancyWriter{
		writer: writer,
		attrs:  strings.Join(append([]string{Reset}, attrs...), ";"),
	}
}
