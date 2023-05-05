package message

import (
	"strings"
	"sync"

	"github.com/tychoish/grip/level"
)

// GroupComposer handles groups of composers as a single message,
// joining messages with a new line for the string format and returning a
// slice of interfaces for the Raw() form.
//
// Unlike most composer types, the GroupComposer is exported, and
// provides the additional Messages() method to access the composer
// objects as a slice.
type GroupComposer struct {
	mutex    sync.RWMutex
	messages []Composer
	cache    string
}

// BuildGroupComposer provides a variadic interface for creating a
// GroupComposer.
func BuildGroupComposer(msgs ...Composer) *GroupComposer {
	return MakeGroupComposer(msgs)
}

// MakeGroupComposer returns a GroupComposer object from a slice of
// Composers.
func MakeGroupComposer(msgs []Composer) *GroupComposer {
	return &GroupComposer{messages: msgs}
}

// String satisfies the fmt.Stringer interface, and returns a string
// of the string form of all constituent composers joined with a newline.
func (g *GroupComposer) String() string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	if g.cache != "" {
		return g.cache
	}

	out := make([]string, 0, len(g.messages))
	for _, m := range g.messages {
		if m != nil && m.Loggable() {
			out = append(out, m.String())
		}
	}
	g.cache = strings.Join(out, "\n")
	return g.cache
}

// Raw returns a slice of interfaces containing the raw form of all
// the constituent composers.
func (g *GroupComposer) Raw() any {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	out := make([]any, 0, len(g.messages))
	for _, m := range g.messages {
		if m != nil && m.Loggable() {
			out = append(out, m.Raw())
		}
	}

	return out
}

// Loggable returns true if at least one of the constituent Composers
// is loggable.
func (g *GroupComposer) Loggable() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	for _, m := range g.messages {
		if m != nil && m.Loggable() {
			return true
		}
	}

	return false
}

func (g *GroupComposer) Structured() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	for _, m := range g.messages {
		if m != nil && m.Structured() {
			return true
		}
	}

	return false
}

// Priority returns the highest priority of the constituent Composers.
func (g *GroupComposer) Priority() level.Priority {
	var highest level.Priority

	g.mutex.RLock()
	defer g.mutex.RUnlock()

	for _, m := range g.messages {
		if m != nil {
			pri := m.Priority()
			if pri > highest {
				highest = pri
			}
		}
	}

	return highest
}

// SetPriority sets the priority of all constituent Composers *only*
// if the existing level is unset (or otherwise invalid), and will
// *not* unset the level of a constituent composer.
func (g *GroupComposer) SetPriority(l level.Priority) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	for _, m := range g.messages {
		if m != nil {
			m.SetPriority(l)
		}
	}

	return
}

// Messages returns a the underlying collection of messages.
func (g *GroupComposer) Messages() []Composer {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return g.messages
}

func (g *GroupComposer) Unwrap() Composer {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	switch len(g.messages) {
	case 0:
		return nil
	case 1:
		return g.messages[0]
	case 2:
		return &wrappedImpl{parent: g.messages[0], Composer: g.messages[1]}
	default:
		var stack Composer

		for idx := len(g.messages) - 1; idx >= 0; idx-- {
			stack = Wrap(stack, g.messages[idx])
		}

		return stack
	}
}

// Extend makes it possible to add a group of messages to an existing
// group composer.
func (g *GroupComposer) Extend(msg []Composer) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.cache = ""
	g.messages = append(g.messages, msg...)
}

// Add supports adding messages to an existing group composer.
func (g *GroupComposer) Add(msg Composer) { g.Append(msg) }

// Append provides a variadic alternative to the Extend method.
func (g *GroupComposer) Append(msgs ...Composer) { g.Extend(msgs) }

// Annotate calls the Annotate method of every non-nil component
// Composer.
func (g *GroupComposer) Annotate(k string, v any) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for _, m := range g.messages {
		if m != nil {
			m.Annotate(k, v)
		}
	}
}
