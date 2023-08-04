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
// there are any events wrapped or embedded in the message

func Sender(s send.Sender, coll *Collector) send.Sender { return &senderImpl{Sender: s, coll: coll} }

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
