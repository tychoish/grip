package send

import (
	"context"
	"fmt"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/message"
)

type asyncGroupSender struct {
	pipes   []chan message.Composer
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
	}
	s.ctx, s.cancel = context.WithCancel(ctx)

	for i := 0; i < len(senders); i++ {
		p := make(chan message.Composer, bufferSize)
		s.pipes = append(s.pipes, p)
		go func(pipe chan message.Composer, sender Sender) {
			for {
				select {
				case <-s.ctx.Done():
					return
				case m := <-pipe:
					if m == nil {
						continue
					}
					sender.Send(m)
				}
			}
		}(p, senders[i])
	}

	s.closer = func() error {
		s.cancel()
		catcher := &erc.Collector{}

		for _, sender := range s.senders {
			catcher.Add(sender.Close())
		}

		for idx, pipe := range s.pipes {
			if len(pipe) > 0 {
				catcher.Add(fmt.Errorf("buffer for sender #%d has %d items remaining",
					idx, len(pipe)))

			}
			close(pipe)
		}

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

	for _, p := range s.pipes {
		select {
		case <-s.ctx.Done():
		case p <- m:
			continue
		}
	}
}

func (s *asyncGroupSender) Flush(ctx context.Context) error {
	catcher := &erc.Collector{}
	for _, sender := range s.senders {
		catcher.Add(sender.Flush(ctx))
	}
	return catcher.Resolve()
}
