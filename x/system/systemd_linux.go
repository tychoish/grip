package system

import (
	"github.com/coreos/go-systemd/journal"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type systemdJournal struct {
	fallback send.Sender

	options map[string]string
	send.Base
}

// MakeSystemdSender creates a Sender object that writes log messages
// to the systemd journal service directly. If such a service does not
// exist on the current system, returns a sender that writes all
// messages to standard output.
func MakeSystemdSender() send.Sender {
	if !journal.Enabled() {
		return send.MakeStdError()
	}

	s := &systemdJournal{
		options:  make(map[string]string),
		fallback: send.MakeStdError(),
	}

	s.SetErrorHandler(send.ErrorHandlerFromSender(s.fallback))
	s.SetFormatter(send.MakePlainFormatter())

	s.reconfig()

	return s
}

func (s *systemdJournal) reconfig() {
	s.fallback.SetFormatter(s.GetFormatter())
	s.fallback.SetName(s.Name())
}

func (s *systemdJournal) SetName(name string) { s.Base.SetName(name); s.reconfig() }

func (s *systemdJournal) SetFormater(fmtr send.MessageFormatter) {
	s.Base.SetFormatter(fmtr)
	s.reconfig()
}

func (s *systemdJournal) Send(m message.Composer) {
	if !send.ShouldLog(s, m) {
		return
	}

	ec := &erc.Collector{}
	ec.WithRecover(func() {
		outstr, err := s.Format(m)
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}

		if err := journal.Send(outstr, convertPrioritySystemd(m.Priority(), 0), s.options); err != nil {
			s.HandleError(send.WrapError(err, m))
		}
	})
	if err := ec.Resolve(); err != nil {
		// there was a panic
		s.HandleError(send.WrapError(err, m))
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
		switch {
		case prio <= 25:
			return journal.PriDebug
		case prio%25 == 0 && int(prio)-25 >= 0:
			prio -= 25
			return convertPrioritySystemd(prio, depth+1)
		default:
			val := int(prio) - int(prio)%25
			if val >= 0 {
				return convertPrioritySystemd(level.Priority(val), depth+1)
			}

			return journal.PriDebug
		}
	}
}
