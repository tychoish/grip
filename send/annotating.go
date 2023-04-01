package send

import (
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/message"
)

type annotatingSender struct {
	Sender
	annotations map[string]any
}

// MakeAnnotating adds the annotations defined in the annotations
// map to every argument.
//
// Calling code should assume that the sender owns the annotations map
// and it should not attempt to modify that data after calling the
// sender constructor. Furthermore, since it owns the sender, callin Close on
// this sender will close the underlying sender.
//
// While you can wrap an existing sender with the annotator, changes
// to the annotating sender (e.g. level, formater, error handler) will
// propagate to the embedded sender.
func MakeAnnotating(s Sender, annotations map[string]any) Sender {
	return &annotatingSender{
		Sender:      s,
		annotations: annotations,
	}
}
func (s *annotatingSender) Unwrap() Sender { return s.Sender }

func (s *annotatingSender) Send(m message.Composer) {
	if !ShouldLog(s, m) {
		return
	}

	ec := &erc.Collector{}

	for k, v := range s.annotations {
		ec.Add(m.Annotate(k, v))
	}

	if ec.HasErrors() {
		s.ErrorHandler()(ec.Resolve(), m)
		return
	}

	s.Sender.Send(m)
}
