package stdlog

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type nativeLogger struct {
	logger *log.Logger
	send.Base
}

// MakeStdOutput returns an unconfigured native standard-out logger.
func MakeStdOutput() send.Sender {
	return WrapWriter(os.Stdout)
}

// MakeStdError returns an unconfigured Sender implementation that
// writes all logging output to standard error.
func MakeStdError() send.Sender {
	return WrapWriter(os.Stderr)
}

// MakePlainFile writes all output to a file, but does not
// prepend any log formatting to each message.
//
// The underlying mechanism uses the standard library's logging facility.
func MakePlainFile(filePath string) (send.Sender, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file: %w", err)
	}
	s := makeNativeFromWriter(f, 0)
	s.SetFormatter(send.MakePlainFormatter())
	s.SetCloseHook(f.Close)

	return s, nil
}

// MakeFile creates a file-based logger, writing output to
// the specified file. The Sender instance is not configured: Pass to
// Journaler.SetSender or call SetName before using.
func MakeFile(filePath string) (send.Sender, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, fmt.Errorf("error opening logging file: %w", err)
	}

	s := makeNativeFromWriter(f, log.LstdFlags)
	s.SetFormatter(send.MakeDefaultFormatter())
	s.SetCloseHook(f.Close)

	return s, nil
}

// WrapWriter constructs a new unconfigured sender that directly
// wraps any writer implementation. These loggers prepend time and
// logger name information to the beginning of log lines.
//
// As a special case, if the writer is a *WriterSender, then this
// method will unwrap and return the underlying sender from the writer.
func WrapWriter(wr io.Writer) send.Sender {
	if s, ok := wr.(send.WriterSender); ok {
		return s
	}

	s := makeNativeFromWriter(wr, log.LstdFlags)
	s.SetFormatter(send.MakeDefaultFormatter())

	return s
}

func makeNativeFromWriter(wr io.Writer, stdFlags int) *nativeLogger {
	s := &nativeLogger{}

	s.logger = log.New(wr, "", stdFlags)
	s.SetErrorHandler(send.ErrorHandlerFromLogger(s.logger))
	s.SetPriority(level.Trace)

	return s
}

func (s *nativeLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
		out, err := s.Format(m)
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}

		s.logger.Print(out)
	}
}
