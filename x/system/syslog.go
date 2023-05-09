// +go:build linux freebsd solaris darwin

package system

import (
	"fmt"
	"log/syslog"
	"os"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type syslogger struct {
	logger        *syslog.Writer
	fallback      send.Sender
	fallbackSetup adt.Once[send.Sender]
	send.Base
}

// MakeSyslogSender constructs a minimal and unconfigured logger that
// sends all log message over a socket to a syslog instance at the
// specified address. If no connection can be made, the
func MakeSyslogSender(network, raddr string) send.Sender {
	s := &syslogger{}

	s.SetResetHook(func() {
		s.fallback = s.fallbackSetup.Do(func() send.Sender {
			return send.WrapWriterPlain(os.Stderr)
		})

		s.SetErrorHandler(send.ErrorHandlerFromSender(s.fallback))
		s.fallback.SetFormatter(s.Formatter())

		if s.logger != nil {
			if err := s.logger.Close(); err != nil {
				s.ErrorHandler()(err, message.MakeString("problem closing syslogger"))
			}
		}

		w, err := syslog.Dial(network, raddr, syslog.LOG_DEBUG, s.Name())
		if err != nil {
			s.ErrorHandler()(err, message.WrapErrorf(err,
				"error restarting syslog [%s] for logger: %s", err.Error(), s.Name()))
			return
		}

		s.SetCloseHook(func() error {
			return w.Close()
		})

		s.logger = w
	})

	s.SetFormatter(send.MakeDefaultFormatter())

	return s
}

// MakeLocalSyslog is a constructor for creating the same kind of
// Sender instance as NewSyslogLogger, except connecting directly to
// the local syslog service. If there is no local syslog service, or
// there are issues connecting to it, writes logging messages to
// standard error. Pass to Journaler.SetSender or call SetName before using.
func MakeLocalSyslog() send.Sender { return MakeSyslogSender("", "") }
func (s *syslogger) Close() error  { return s.logger.Close() }
func (s *syslogger) Send(m message.Composer) {
	if !send.ShouldLog(s, m) {
		return
	}
	if err := fun.Check(func() {
		outstr, err := s.Formatter()(m)
		if err != nil {
			s.ErrorHandler()(err, m)
			return
		}

		if err := s.sendToSysLog(m.Priority(), outstr); err != nil {
			s.ErrorHandler()(err, m)
		}
	}); err != nil {
		// there was a panic
		s.ErrorHandler()(err, m)
	}
}

func (s *syslogger) sendToSysLog(p level.Priority, message string) error {
	switch p {
	case level.Emergency:
		return s.logger.Emerg(message)
	case level.Alert:
		return s.logger.Alert(message)
	case level.Critical:
		return s.logger.Crit(message)
	case level.Error:
		return s.logger.Err(message)
	case level.Warning:
		return s.logger.Warning(message)
	case level.Notice:
		return s.logger.Notice(message)
	case level.Info:
		return s.logger.Info(message)
	case level.Debug, level.Trace:
		return s.logger.Debug(message)
	}

	return fmt.Errorf("encountered error trying to send: {%s}. Possibly, priority related", message)
}
