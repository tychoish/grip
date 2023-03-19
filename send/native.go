package send

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

type nativeLogger struct {
	logger *log.Logger
	Base
}

// NewFile creates a Sender implementation that writes log
// output to a file. Returns an error but falls back to a standard
// output logger if there's problems with the file. Internally using
// the go standard library logging system.
func NewFile(name, filePath string, l LevelInfo) (Sender, error) {
	s, err := MakeFile(filePath)
	if err != nil {
		return nil, err
	}

	return setup(s, name, l)
}

// MakeFile creates a file-based logger, writing output to
// the specified file. The Sender instance is not configured: Pass to
// Journaler.SetSender or call SetName before using.
func MakeFile(filePath string) (Sender, error) {
	s := &nativeLogger{}
	s.SetFormatter(MakeDefaultFormatter())

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file, %s", err.Error())
	}

	s.level.Set(LevelInfo{level.Trace, level.Trace})

	s.reset.Set(func() {
		prefix := fmt.Sprintf("[%s] ", s.Name())
		s.logger = log.New(f, prefix, log.LstdFlags)
		s.SetErrorHandler(ErrorHandlerFromLogger(log.New(os.Stderr, prefix, log.LstdFlags)))
	})

	s.closer.Set(func() error {
		return f.Close()
	})

	return s, nil
}

// NewStdOutput creates a new Sender interface that writes all
// loggable messages to a standard output logger that uses Go's
// standard library logging system.
func NewStdOutput(name string, l LevelInfo) (Sender, error) {
	return setup(MakeStdOutput(), name, l)
}

// MakeStdOutput returns an unconfigured native standard-out logger. You
// *must* call SetName on this instance before using it. (Journaler's
// SetSender will typically do this.)
func MakeStdOutput() Sender {
	return WrapWriter(os.Stdout)
}

// MakeStdError returns an unconfigured Sender implementation that
// writes all logging output to standard error.
func MakeStdError() Sender {
	return WrapWriter(os.Stderr)
}

// NewStdError constructs a configured Sender that writes all
// output to standard error.
func NewStdError(name string, l LevelInfo) (Sender, error) {
	return setup(MakeStdError(), name, l)
}

// WrapWriter constructs a new unconfigured sender that directly
// wraps any writer implementation. These loggers prepend time and
// logger name information to the beginning of log lines.
//
// As a special case, if the writer is a *WriterSender, then this
// method will unwrap and return the underlying sender from the writer.
func WrapWriter(wr io.Writer) Sender {
	if s, ok := wr.(*WriterSender); ok {
		return s.Sender
	}

	s := &nativeLogger{}
	_ = s.SetLevel(LevelInfo{level.Trace, level.Trace})

	s.SetResetHook(func() {
		s.logger = log.New(wr, fmt.Sprintf("[%s] ", s.Name()), log.LstdFlags)
		s.SetErrorHandler(ErrorHandlerFromLogger(s.logger))
	})
	s.SetErrorHandler(ErrorHandlerFromLogger(s.logger))
	s.SetFormatter(MakeDefaultFormatter())

	return s
}

// NewWrappedWriter constructs a fully configured Sender
// implementation that writes all data to the underlying writer.
// These loggers prepend time and logger name information to the
// beginning of log lines.
//
// As a special case, if the writer is a *WriterSender, then this
// method will unwrap and return the underlying sender from the writer.
func NewWrappedWriter(name string, wr io.Writer, l LevelInfo) (Sender, error) {
	return setup(WrapWriter(wr), name, l)
}

func (s *nativeLogger) Send(m message.Composer) {
	if s.Level().ShouldLog(m) {
		out, err := s.Formatter()(m)
		if err != nil {
			s.ErrorHandler()(err, m)
			return
		}

		s.logger.Print(out)
	}
}
