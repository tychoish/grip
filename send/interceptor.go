package send

import "github.com/tychoish/grip/message"

type interceptor struct {
	Sender
	filter func(message.Composer)
}

// MakeFilter constructs an intercepting sender implementation
// that wraps another sender, and passes all messages (regardless of
// loggability or level,) through a filtering function.
//
// This implementation and the filtering function exist mostly to be
// able to inject metrics collection into existing logging pipelines,
// though the interceptor may be used for filtering or pre-processing
// as well.
func MakeFilter(sender Sender, ifn func(message.Composer)) Sender {
	return &interceptor{
		Sender: sender,
		filter: ifn,
	}
}

func (s *interceptor) Unwrap() Sender          { return s.Sender }
func (s *interceptor) Send(m message.Composer) { s.filter(m); s.Sender.Send(m) }
