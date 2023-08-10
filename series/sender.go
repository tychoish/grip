package series

import (
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type senderImpl struct {
	send.Sender
	coll *Collector
}

// Sender wraps a send.Sender and a collector and unifies them: if
// there are any events wrapped or embedded in the message the sender
// extracts them, propagating the events to the collector and then
// separately passing the message along to the underlying sender.
//
// The events are, typically, not part of the message sent to the
// underlying sender: while Events do have a string form that can be
// logged, and most senders will handle them appropriately, events are
// not logged with *this* sender.
func Sender(s send.Sender, coll *Collector) send.Sender {
	return &senderImpl{Sender: s, coll: coll}
}

func (s senderImpl) Unwrap() send.Sender { return s.Sender }

func (s senderImpl) Send(m message.Composer) {
	em := extractMetrics(m, metricMessageWithComposer)
	for _, event := range em.Events {
		if event.m == nil {
			continue
		}
		s.coll.PushEvent(event)
	}
	s.Sender.Send(em.Composer)
}
