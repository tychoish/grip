package system

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/coreos/go-systemd/journal"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type systemdJournal struct {
	options map[string]string
	*send.Base
}

// NewSystemdSender creates a Sender object that writes log messages
// to the system's systemd journald logging facility. If there's an
// error with the sending to the journald, messages fallback to
// writing to standard output.
func NewSystemdSender(name string, l send.LevelInfo) (send.Sender, error) {
	s, err := MakeSystemdSender()
	if err != nil {
		return nil, err
	}
	if err := s.SetLevel(l); err != nil {
		return nil, err
	}

	s.SetName(name)

	return s, nil
}

// MakeSystemdSender constructs an unconfigured systemd journald
// logger. Pass to Journaler.SetSender or call SetName before using.
func MakeSystemdSender() (send.Sender, error) {
	if !journal.Enabled() {
		return nil, errors.New("systemd journal logging is not available on this platform")
	}

	s := &systemdJournal{
		options: make(map[string]string),
		Base:    send.NewBase(""),
	}

	fallback := log.New(os.Stdout, "", log.LstdFlags)
	s.SetErrorHandler(send.ErrorHandlerFromLogger(fallback))

	s.SetResetHook(func() {
		fallback.SetPrefix(fmt.Sprintf("[%s] ", s.Name()))
	})

	return s, nil
}

func (s *systemdJournal) Send(m message.Composer) {
	defer func() {
		if err := recover(); err != nil {
			s.ErrorHandler()(fmt.Errorf("panic: %v", err), m)
		}
	}()

	if level := s.Level(); level.ShouldLog(m) {
		err := journal.Send(m.String(), convertPrioritySystemd(level, m.Priority()), s.options)
		if err != nil {
			s.ErrorHandler()(err, m)
		}
	}
}

func convertPrioritySystemd(l send.LevelInfo, p level.Priority) journal.Priority {
	switch p {
	case level.Emergency:
		return journal.PriEmerg
	case level.Alert:
		return journal.PriAlert
	case level.Critical:
		return journal.PriCrit
	case level.Error:
		return journal.PriErr
	case level.Warning:
		return journal.PriWarning
	case level.Notice:
		return journal.PriNotice
	case level.Info:
		return journal.PriInfo
	case level.Debug, level.Trace:
		return journal.PriDebug
	default:
		return convertPrioritySystemd(l, l.Default)
	}
}
