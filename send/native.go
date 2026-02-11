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
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
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
	return MakeWriter(os.Stdout)
}

// MakeStdError returns an unconfigured Sender implementation that
// writes all logging output to standard error.
func MakeStdError() Sender {
	return MakeWriter(os.Stderr)
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
