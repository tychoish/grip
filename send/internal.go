package send

import (
	"sync"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// InternalSender implements a Sender object that makes it possible to
// access logging messages, in the InternalMessage format without
// logging to an output method. The Send method does not filter out
// under-priority and unloggable messages. Used  for testing
// purposes.
type InternalSender struct {
	Base
	Filter func(message.Composer)

	name   string
	output chan *InternalMessage
	mu     sync.RWMutex
}

// InternalMessage provides a complete representation of all
// information associated with a logging event.
type InternalMessage struct {
	Message  message.Composer
	Logged   bool
	Priority level.Priority
	Rendered string
}

// NewInternal creates and returns a Sender implementation that
// does not log messages, but converts them to the InternalMessage
// format and puts them into an internal channel, that allows you to
// access the massages via the extra "GetMessage" method. Useful for
// testing.
func NewInternal(size int) *InternalSender {
	return &InternalSender{output: make(chan *InternalMessage, size)}
}

// MakeInternal constructs an internal sender object, typically
// for use in testing.
func MakeInternal() *InternalSender {
	return NewInternal(1000)
}

// GetMessage pops the first message in the queue and returns.
func (s *InternalSender) GetMessage() *InternalMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return <-s.output
}

func (s *InternalSender) GetMessageSafe() (*InternalMessage, bool) {
	select {
	case m := <-s.output:
		return m, true
	default:
		return nil, false
	}
}

// HasMessage returns true if there is at least one message that has
// not be removed.
func (s *InternalSender) HasMessage() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.output) > 0
}

// Len returns the number of sent messages that have not been retrieved.
func (s *InternalSender) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.output)
}

// Send sends a message. Unlike all other sender implementations, all
// messages are sent, but the InternalMessage format tracks
// "loggability" for testing purposes.
func (s *InternalSender) Send(m message.Composer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Filter != nil {
		s.Filter(m)
	}
	s.output <- &InternalMessage{
		Message:  m,
		Priority: m.Priority(),
		Rendered: m.String(),
		Logged:   ShouldLog(s, m),
	}
}
