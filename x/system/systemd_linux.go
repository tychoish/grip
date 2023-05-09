package system

import (
	"os"

	"github.com/coreos/go-systemd/journal"
	"github.com/tychoish/fun"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type systemdJournal struct {
	fallback      send.Sender
	fallbackSetup adt.Once[send.Sender]

	options map[string]string
	send.Base
}

// MakeSystemdSender creates a Sender object that writes log messages
// to the sysemd journal service directly. If such a service does not
// exist on the current system, returns a sender that writes all
// messages to standard output.
func MakeSystemdSender() send.Sender {
	if !journal.Enabled() {
		return send.WrapWriter(os.Stderr)
	}

	s := &systemdJournal{
		options: make(map[string]string),
	}

	s.SetResetHook(func() {
		s.fallback = s.fallbackSetup.Do(func() send.Sender {
			return send.WrapWriterPlain(os.Stderr)
		})
		s.SetErrorHandler(send.ErrorHandlerFromSender(s.fallback))
		s.fallback.SetFormatter(s.Formatter())
	})

	s.SetFormatter(send.MakePlainFormatter())

	return s
}

func (s *systemdJournal) Send(m message.Composer) {
	if !send.ShouldLog(s, m) {
		return
	}

	if err := fun.Check(func() {
		outstr, err := s.Formatter()(m)
		if err != nil {
			s.ErrorHandler()(err, m)
			return
		}

		if err := journal.Send(outstr, convertPrioritySystemd(m.Priority(), 0), s.options); err != nil {
			s.ErrorHandler()(err, m)
		}
	}); err != nil {
		// there was a panic
		s.ErrorHandler()(err, m)
	}
}

func convertPrioritySystemd(prio level.Priority, depth int) journal.Priority {
	switch prio {
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
	case level.Debug, level.Trace, level.Invalid:
		return journal.PriDebug
	default:
		// levels increase by 25(ish); if we're going to be
		// invalid by being too low, just return debug now,
		// otherwise, attempt to round down to the nearest 25,
		// should only need 1 or 2 recursions to get to some
		// return.
		if prio%25 == 0 {
			prio -= 25
			convertPrioritySystemd(prio, depth+1)
		}

		if l := (prio - (prio % 25)); l < 0 {
			return journal.PriDebug
		} else {
			return convertPrioritySystemd(l, depth+1)
		}
	}
}
