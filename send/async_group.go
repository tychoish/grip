package send

import (
	"context"
	"runtime"
	"sync"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/grip/level"
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

// MakeAsyncGroup produces an implementation of the Sender interface
// that, like the MultiSender, distributes a single message to a group
// of underlying sender implementations.
//
// This sender does not guarantee ordering of messages. The buffer
// size controls the size of the buffer between each sender and the
// individual senders.
//
// The sender takes ownership of the underlying Senders, so closing
// this sender closes all underlying Senders.
func MakeAsyncGroup(ctx context.Context, bufferSize int, senders ...Sender) Sender {
	s := &asyncGroupSender{
		baseCtx: ctx,
		// unlimited number of senders, bufferSize is
		// constrained buy the buffer size in the broker.
		senders: fun.Must(pubsub.NewDeque[Sender](pubsub.DequeOptions{Unlimited: true})),
		broker: pubsub.NewBroker[message.Composer](ctx, pubsub.BrokerOptions{
			BufferSize:       bufferSize,
			ParallelDispatch: true,
			WorkerPoolSize:   runtime.NumCPU(),
		}),
	}
	for idx := range senders {
		fun.InvariantMust(s.senders.PushBack(senders[idx]), "populate senders")
	}

	shutdown := make(chan struct{})
	s.shutdownSignal = shutdown
	s.ctx, s.cancel = context.WithCancel(ctx)

	for i := 0; i < len(senders); i++ {
		s.startSenderWorker(senders[i])
	}
	wg := &s.wg
	s.closer.Set(func() (err error) {
		s.doClose.Do(func() {
			catcher := &erc.Collector{}
			defer func() { err = catcher.Resolve() }()
			defer s.cancel()
			catcher.Add(s.senders.Close())

			wg.Add(1)
			go func() {
				defer wg.Done()
				catcher.Add(fun.Observe(ctx, s.senders.Iterator(), func(sender Sender) {
					catcher.Add(sender.Close())
				}))

			}()

			catcher.Add(s.senders.Close())
			close(shutdown)
			wg.Wait(ctx)
			s.cancel()
		})

		// let the defer in the closer set the err
		return
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

func (s *asyncGroupSender) SetPriority(p level.Priority) {
	s.Base.SetPriority(p)

	fun.InvariantMust(fun.Observe(s.ctx, s.senders.Iterator(), func(sender Sender) {
		sender.SetPriority(p)
	}))
}

func (s *asyncGroupSender) Send(m message.Composer) {
	if !ShouldLog(s, m) {
		return
	}
	s.broker.Publish(s.ctx, m)
}

func (s *asyncGroupSender) Flush(ctx context.Context) error {
	catcher := &erc.Collector{}

	catcher.Add(fun.Observe(ctx, s.senders.Iterator(), func(sender Sender) {
		catcher.Add(sender.Flush(ctx))
	}))

	return catcher.Resolve()
}
