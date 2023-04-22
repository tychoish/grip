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

// MakeFile creates a file-based logger, writing output to
// the specified file. The Sender instance is not configured: Pass to
// Journaler.SetSender or call SetName before using.
func MakeFile(filePath string) (Sender, error) {
	s := &nativeLogger{}
	s.SetFormatter(MakeDefaultFormatter())

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file: %w", err)
	}
	s.SetPriority(level.Trace)

	s.SetResetHook(func() {
		prefix := fmt.Sprintf("[%s] ", s.Name())
		s.logger = log.New(f, prefix, log.LstdFlags)
		s.SetErrorHandler(ErrorHandlerFromLogger(log.New(os.Stderr, prefix, log.LstdFlags)))
	})
	s.SetCloseHook(func() error {
		return f.Close()
	})
	s.doReset()

	return s, nil
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

// WrapWriter constructs a new unconfigured sender that directly
// wraps any writer implementation. These loggers prepend time and
// logger name information to the beginning of log lines.
//
// As a special case, if the writer is a *WriterSender, then this
// method will unwrap and return the underlying sender from the writer.
func WrapWriter(wr io.Writer) Sender {
	if s, ok := wr.(WriterSender); ok {
		return s
	}

	s := &nativeLogger{}

	s.SetResetHook(func() {
		s.logger = log.New(wr, fmt.Sprintf("[%s] ", s.Name()), log.LstdFlags)
		s.SetErrorHandler(ErrorHandlerFromLogger(s.logger))
	})
	s.SetPriority(level.Trace)
	s.SetErrorHandler(ErrorHandlerFromLogger(s.logger))
	s.SetFormatter(MakeDefaultFormatter())
	s.doReset()

	return s
}

func (s *nativeLogger) Send(m message.Composer) {
	if ShouldLog(s, m) {
		out, err := s.Formatter()(m)
		if err != nil {
			s.ErrorHandler()(err, m)
			return
		}

		s.logger.Print(out)
	}
}
