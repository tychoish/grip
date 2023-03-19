package send

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/tychoish/fun"
	"github.com/tychoish/grip/level"
)

// WrapWriterPlain produces a simple writer that does not modify the log
// lines passed to the writer.
//
// As a special case, if the writer is a *WriterSender, then this
// method will unwrap and return the underlying sender from the writer.
func WrapWriterPlain(wr io.Writer) Sender {
	if s, ok := wr.(*WriterSender); ok {
		return s.Sender
	}

	s := &nativeLogger{}
	s.logger = log.New(wr, "", 0)

	s.SetFormatter(MakePlainFormatter())
	fun.InvariantMust(s.SetLevel(LevelInfo{Default: level.Trace, Threshold: level.Trace}))
	return s
}

// NewPlainStdOutput returns a configured sender that has no prefix and
// uses a plain formatter for messages, using only the string format
// for each message. This sender writes all output to standard output.
func NewPlainStdOutput(name string, l LevelInfo) (Sender, error) {
	return setup(MakePlain(), name, l)
}

// NewPlainStdError returns a configured sender that has no prefix and
// uses a plain formatter for messages, using only the string format
// for each message. This sender writes all output to standard error.
func NewPlainStdError(name string, l LevelInfo) (Sender, error) {
	return setup(MakePlainStdError(), name, l)
}

// NewPlainFile creates a new configured logger that writes log
// data to a file.
func NewPlainFile(name, file string, l LevelInfo) (Sender, error) {
	s, err := MakePlainFile(file)
	if err != nil {
		return nil, err
	}

	return setup(s, name, l)
}

// MakePlain returns an unconfigured sender without a prefix,
// using the plain log formatter. This Sender writes all output to
// standard error.
func MakePlain() Sender {
	s := &nativeLogger{}
	fun.InvariantMust(s.SetLevel(LevelInfo{level.Trace, level.Trace}))
	s.SetFormatter(MakePlainFormatter())
	s.SetResetHook(func() {
		s.logger = log.New(os.Stdout, "", 0)
		s.SetErrorHandler(ErrorHandlerFromLogger(s.logger))
	})

	return s
}

// MakePlainFile writes all output to a file, but does not
// prepend any log formatting to each message.
func MakePlainFile(filePath string) (Sender, error) {
	s := &nativeLogger{}
	s.SetFormatter(MakeDefaultFormatter())
	fun.InvariantMust(s.SetLevel(LevelInfo{level.Trace, level.Trace}))

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file, %s", err.Error())
	}

	fallback := log.New(os.Stderr, "", log.LstdFlags)

	s.SetErrorHandler(ErrorHandlerFromLogger(fallback))
	s.SetResetHook(func() { s.logger = log.New(f, "", 0) })
	s.SetCloseHook(func() error { return f.Close() })

	return s, nil
}

// MakePlainStdError returns an unconfigured sender without a prefix,
// using the plain log formatter. This Sender writes all output to
// standard error.
func MakePlainStdError() Sender {
	s := &nativeLogger{}
	fun.InvariantMust(s.SetLevel(LevelInfo{Default: level.Trace, Threshold: level.Trace}))
	s.SetFormatter(MakePlainFormatter())
	s.SetResetHook(func() {
		s.logger = log.New(os.Stderr, "", 0)
		s.SetErrorHandler(ErrorHandlerFromLogger(s.logger))
	})

	return s
}
