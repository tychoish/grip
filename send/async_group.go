package send

import (
	"context"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/itertool"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/grip/message"
)

type asyncGroupSender struct {
	broker  *pubsub.Broker[message.Composer]
	senders []Sender
	cancel  context.CancelFunc
	ctx     context.Context
	*Base
}

// NewAsyncGroup produces an implementation of the Sender interface that,
// like the MultiSender, distributes a single message to a group of underlying
// sender implementations.
//
// This sender does not guarantee ordering of messages, and Send operations may
// if the underlying senders fall behind the buffer size.
//
// The sender takes ownership of the underlying Senders, so closing this sender
// closes all underlying Senders.
func NewAsyncGroup(ctx context.Context, bufferSize int, senders ...Sender) Sender {
	s := &asyncGroupSender{
		senders: senders,
		Base:    NewBase(""),
		broker: pubsub.NewBroker[message.Composer](ctx, pubsub.BrokerOptions{
			BufferSize:       bufferSize,
			ParallelDispatch: true,
		}),
	}
	s.ctx, s.cancel = context.WithCancel(ctx)

	shutdown := make(chan struct{})
	for i := 0; i < len(senders); i++ {
		go func(pipe chan message.Composer, sender Sender) {
			for {
				select {
				case <-shutdown:
					s.broker.Unsubscribe(ctx, pipe)
					return
				case <-s.ctx.Done():
					return
				case m := <-pipe:
					if m == nil {
						continue
					}
					sender.Send(m)
				}
			}
		}(s.broker.Subscribe(ctx), senders[i])
	}

	s.closer = func() error {
		catcher := &erc.Collector{}

		for _, sender := range s.senders {
			catcher.Add(sender.Close())
		}

		close(shutdown)
		s.cancel()

		return catcher.Resolve()
	}
	return s
}

func (s *asyncGroupSender) SetLevel(l LevelInfo) error {
	// if the base level isn't valid, then we shouldn't overwrite
	// constinuent senders (this is the indication that they were overridden.)
	if !s.Base.Level().Valid() {
		return nil
	}

	if err := s.Base.SetLevel(l); err != nil {
		return err
	}

	catcher := &erc.Collector{}

	for _, sender := range s.senders {
		catcher.Add(sender.SetLevel(l))
	}

	return catcher.Resolve()
}

func (s *asyncGroupSender) Send(m message.Composer) {
	bl := s.Base.Level()
	if bl.Valid() && !bl.ShouldLog(m) {
		return
	}
	s.broker.Publish(s.ctx, m)
}

func (s *asyncGroupSender) Flush(ctx context.Context) error {
	catcher := &erc.Collector{}

	fun.ObserveWorkerFuncs(ctx,
		itertool.Transform(ctx,
			itertool.Slice(s.senders),
			itertool.Transformer(func(s Sender) fun.WorkerFunc { return s.Flush }),
		),
		catcher.Add,
	).Run(ctx)

	return catcher.Resolve()
}
