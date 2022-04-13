package desktop

import (
	"fmt"

	"github.com/gen2brain/beeep"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type desktopNotify struct {
	*send.Base
}

// NewDesktopNotify constructs a sender that pushes messages
// to local system notification process.
func NewSender(name string, l send.LevelInfo) (send.Sender, error) {
	s := &desktopNotify{
		Base: send.NewBase(name),
	}

	if err := s.SetLevel(l); err != nil {
		return nil, fmt.Errorf("problem seeting level on new sender: %w", err)
	}

	return s, nil
}

// MakeDesktopNotify constructs a default sender that pushes messages
// to local system notification.
func MakeSender(name string) (send.Sender, error) {
	s, err := NewSender(name, send.LevelInfo{Threshold: level.Trace, Default: level.Debug})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *desktopNotify) Send(m message.Composer) {
	if s.Level().ShouldLog(m) {
		if m.Priority() >= level.Critical {
			if err := beeep.Alert(s.Name(), m.String(), ""); err != nil {
				s.ErrorHandler()(err, m)
			}
		} else {
			if err := beeep.Notify(s.Name(), m.String(), ""); err != nil {
				s.ErrorHandler()(err, m)
			}
		}
	}
}
