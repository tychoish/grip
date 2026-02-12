package send

import (
	"fmt"
	"io"
	"os"
)

// MakeStdOutput returns an unconfigured native standard-out logger.
func MakeStdOutput() Sender { return MakeWriter(os.Stdout) }

// MakeStdError returns an unconfigured Sender implementation that
// writes all logging output to standard error.
func MakeStdError() Sender { return MakeWriter(os.Stderr) }

// WrapWriterPlain produces a simple writer that does not modify the log
// lines passed to the writer.
//
// The underlying mechanism uses the standard library's logging facility.
func WrapWriterPlain(wr io.Writer) Sender {
	s := MakeWriter(wr)
	s.SetFormatter(MakePlainFormatter())
	return s
}

// MakeStdOut returns an unconfigured sender without a prefix,
// using the plain log formatter. This Sender writes all output to
// standard error.
//
// The underlying mechanism uses the standard library's logging facility.
func MakeStdOut() Sender { return MakeWriter(os.Stdout) }

func MakeFile(path string) (Sender, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file: %w", err)
	}
	s := newWriter(f)
	s.SetCloseHook(f.Close)
	return s, nil
}

// MakeWriter constructs a Sender that writes all messages to the
// underlying writer.
//
// Leading and trailing space is trimmed and a single newline
// separates every message.
//
// Writes are fully synchronized with regards to eachother.
func MakeWriter(wr io.Writer) Sender { return newWriter(wr) }
