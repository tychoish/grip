package desktop

import (
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
func MakeSender() send.Sender { return &desktopNotify{} }

func (s *desktopNotify) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
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
