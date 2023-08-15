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
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file: %w", err)
	}

	s := makeNativeFromWriter(f, log.LstdFlags)
	s.SetFormatter(MakeDefaultFormatter())
	s.SetCloseHook(func() error { return f.Close() })

	return s, nil
}

// MakeStdOutput returns an unconfigured native standard-out logger.
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
	s := makeNativeFromWriter(wr, log.LstdFlags)
	s.SetFormatter(MakeDefaultFormatter())
	return s
}

func makeNativeFromWriter(wr io.Writer, stdFlags int) *nativeLogger {
	s := &nativeLogger{}
	s.logger = log.New(wr, "", stdFlags)
	s.SetErrorHandler(ErrorHandlerFromLogger(s.logger))
	s.SetPriority(level.Trace)
	return s
}

func (s *nativeLogger) Send(m message.Composer) {
	if ShouldLog(s, m) {
		out, err := s.Format(m)
		if !s.HandleErrorOK(WrapError(err, m)) {
			return
		}

		s.logger.Print(out)
	}
}
