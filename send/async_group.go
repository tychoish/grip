package send

import (
	"context"
	"sync"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/grip/message"
)

type asyncGroupSender struct {
	broker         *pubsub.Broker[message.Composer]
	senders        *pubsub.Deque[Sender]
	wg             fun.WaitGroup
	cancel         context.CancelFunc
	ctx            context.Context
	baseCtx        context.Context
	shutdownSignal <-chan struct{}
	doClose        sync.Once
	Base
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
		baseCtx:        ctx,
		shutdownSignal: make(chan struct{}),
		senders:        fun.Must(pubsub.NewDeque[Sender](pubsub.DequeOptions{Unlimited: true})),
		broker: pubsub.NewBroker[message.Composer](ctx, pubsub.BrokerOptions{
			BufferSize:       bufferSize,
			ParallelDispatch: true,
		}),
	}
	for idx := range senders {
		fun.InvariantMust(s.senders.PushBack(senders[idx]), "populate senders")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	shutdown := make(chan struct{})
	for i := 0; i < len(senders); i++ {
		s.startSenderWorker(senders[i])
	}

	wg := &s.wg
	s.closer.Set(func() error {
		catcher := &erc.Collector{}
		s.doClose.Do(func() {
			s.cancel()
			catcher.Add(s.senders.Close())

			closeAll := fun.ObserveAll(ctx, s.senders.Iterator(), func(sender Sender) {
				catcher.Add(sender.Close())
			})
			closeAll.Add(ctx, wg)

			close(shutdown)
			wg.Wait(ctx)
		})
		return catcher.Resolve()
	})
	return s
}

func (s *asyncGroupSender) startSenderWorker(newSender Sender) {
	s.wg.Add(1)
	go func(pipe chan message.Composer, sender Sender) {
		defer s.wg.Done()
		for {
			select {
			case <-s.shutdownSignal:
				s.broker.Unsubscribe(s.baseCtx, pipe)
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
	}(s.broker.Subscribe(s.baseCtx), newSender)
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

	fun.Observe(s.ctx, s.senders.Iterator(), func(sender Sender) {
		catcher.Add(sender.SetLevel(l))
	})

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

	fun.ObserveAll(ctx, s.senders.Iterator(), func(sender Sender) {
		catcher.Add(sender.Flush(ctx))
	})(ctx)

	return catcher.Resolve()
}
