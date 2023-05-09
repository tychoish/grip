package send

import (
	"fmt"
	"io"
	"os"
)

// WrapWriterPlain produces a simple writer that does not modify the log
// lines passed to the writer.
//
// The underlying mechanism uses the standard library's logging facility.
func WrapWriterPlain(wr io.Writer) Sender {
	s := makeNativeFromWriter(wr, 0)
	s.SetFormatter(MakePlainFormatter())
	return s
}

// MakePlainFile writes all output to a file, but does not
// prepend any log formatting to each message.
//
// The underlying mechanism uses the standard library's logging facility.
func MakePlainFile(filePath string) (Sender, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file: %w", err)
	}
	s := makeNativeFromWriter(f, 0)
	s.SetFormatter(MakePlainFormatter())
	s.SetCloseHook(func() error { return f.Close() })
	return s, nil
}

// MakePlain returns an unconfigured sender without a prefix,
// using the plain log formatter. This Sender writes all output to
// standard error.
//
// The underlying mechanism uses the standard library's logging facility.
func MakePlain() Sender { return WrapWriterPlain(os.Stdout) }

// MakePlainStdError returns an unconfigured sender without a prefix,
// using the plain log formatter. This Sender writes all output to
// standard error.
//
// The underlying mechanism uses the standard library's logging facility.
func MakePlainStdError() Sender { return WrapWriterPlain(os.Stderr) }
