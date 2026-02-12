// +go:build linux freebsd solaris darwin

package system

import (
	"fmt"
	"log/syslog"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type syslogger struct {
	network string
	raddr   string

	logger   *syslog.Writer
	fallback send.Sender

	send.Base
}

// MakeSyslogSender constructs a minimal and unconfigured logger that
// sends all log message over a socket to a syslog instance at the
// specified address. If no connection can be made, the
func MakeSyslogSender(network, raddr string) send.Sender {
	s := &syslogger{
		fallback: send.MakeStdError(),
		raddr:    raddr,
		network:  network,
	}

	s.SetFormatter(send.MakeDefaultFormatter())
	s.SetErrorHandler(send.ErrorHandlerFromSender(s.fallback))
	s.reset()
	return s
}

// MakeLocalSyslog is a constructor for creating the same kind of
// Sender instance as NewSyslogLogger, except connecting directly to
// the local syslog service. If there is no local syslog service, or
// there are issues connecting to it, writes logging messages to
// standard error. Pass to Journaler.SetSender or call SetName before using.
func MakeLocalSyslog() send.Sender { return MakeSyslogSender("", "") }

func (s *syslogger) reconfig() {
	s.fallback.SetFormatter(s.GetFormatter())
	s.fallback.SetName(s.Name())
}

func (s *syslogger) SetName(name string)                    { s.Base.SetName(name); s.reconfig() }
func (s *syslogger) SetFormater(fmtr send.MessageFormatter) { s.Base.SetFormatter(fmtr); s.reconfig() }

func (s *syslogger) reset() {
	s.reconfig()

	if s.logger != nil {
		if err := s.logger.Close(); err != nil {
			s.HandleError(err)
		}
	}

	w, err := syslog.Dial(s.network, s.raddr, syslog.LOG_DEBUG, s.Name())
	if !s.HandleErrorOK(err) {
		return
	}

	s.SetCloseHook(func() error { return w.Close() })

	s.logger = w
}

func (s *syslogger) Send(m message.Composer) {
	if !send.ShouldLog(s, m) {
		return
	}
	ec := &erc.Collector{}
	ec.WithRecover(func() {
		outstr, err := s.Format(m)
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}

		if err := s.sendToSysLog(m.Priority(), outstr); err != nil {
			s.HandleError(send.WrapError(err, m))
		}
	})

	if err := ec.Resolve(); err != nil {
		// there was a panic
		s.HandleError(send.WrapError(err, m))
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
