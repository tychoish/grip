package send

import (
	"context"
	"fmt"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

type multiSender struct {
	senders []Sender
	Base
}

// NewMulti configures a new sender implementation that takes a
// slice of Sender implementations that dispatches all messages to all
// implementations. This constructor forces all member Senders to have
// the same name and Level configuration. Use NewConfiguredMultiSender
// to construct a similar Sender
//
// Use the AddToMulti helper to add additioanl senders to one of these
// multi Sender implementations after construction.
//
// The Sender takes ownership of the underlying Senders, so closing this Sender
// closes all underlying Senders.
func NewMulti(name string, senders []Sender) (Sender, error) {
	for _, sender := range senders {
		sender.SetName(name)
	}

	s := &multiSender{senders: senders}
	s.SetName(name)

	return s, nil
}

// MakeMulti returns a multi sender implementation with
// Sender members, but does not force the senders to have conforming
// name or level values. Use NewMultiSender to construct a list of
// senders with consistent names and level configurations.
//
// Use the AddToMulti helper to add additioanl senders to one of these
// multi Sender implementations after construction.
//
// The Sender takes ownership of the underlying Senders, so closing this Sender
// closes all underlying Senders.
func MakeMulti(senders ...Sender) Sender {
	s := &multiSender{senders: senders}
	_ = s.Base.SetLevel(LevelInfo{Default: level.Invalid, Threshold: level.Invalid})

	return s
}

// AddToMulti is a helper function that takes two Sender instances,
// the first of which must be a multi or async group sender. If this
// is true, then AddToMulti adds the second Sender to the first
// Sender's list of Senders.
//
// Returns an error if the first instance is not a multi sender, or if
// the async group sender has been closed.
func AddToMulti(multi Sender, s Sender) error {
	switch sender := multi.(type) {
	case *multiSender:
		sender.add(s)
		return nil
	case *asyncGroupSender:
		if err := sender.senders.PushBack(s); err != nil {
			return err
		}
		sender.startSenderWorker(s)
		return nil
	default:
		return fmt.Errorf("%s is not a multi sender", multi.Name())
	}
}

func (s *multiSender) Close() error {
	catcher := &erc.Collector{}
	for _, sender := range s.senders {
		catcher.Add(sender.Close())
	}
	return catcher.Resolve()
}

func (s *multiSender) add(sender Sender) {
	sender.SetName(s.Base.Name())
	// ignore the error here; if the Base value on the multiSender
	// is not set, then senders should just have their own level values.
	_ = sender.SetLevel(s.Base.Level())
	s.senders = append(s.senders, sender)
	return
}

func (s *multiSender) Name() string { return s.Base.Name() }
func (s *multiSender) SetName(n string) {
	s.Base.SetName(n)

	for _, sender := range s.senders {
		sender.SetName(n)
	}
}

func (s *multiSender) Level() LevelInfo { return s.Base.Level() }
func (s *multiSender) SetLevel(l LevelInfo) error {
	if err := s.Base.SetLevel(l); err != nil {
		return err
	}

	for _, sender := range s.senders {
		_ = sender.SetLevel(l)
	}

	return nil
}

func (s *multiSender) Send(m message.Composer) {
	// if the base level isn't valid, then we should let each
	// sender decide for itself, rather than short circuiting here
	bl := s.Base.Level()
	if bl.Valid() && !bl.ShouldLog(m) {
		return
	}

	for _, sender := range s.senders {
		sender.Send(m)
	}
}

func (s *multiSender) Flush(ctx context.Context) error {
	catcher := &erc.Collector{}

	for _, sender := range s.senders {
		catcher.Add(sender.Flush(ctx))
	}

	return catcher.Resolve()
}
