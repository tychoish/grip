package message

import "github.com/tychoish/fun"

type wrappedImpl struct {
	parent Composer
	Composer
}

func (wi *wrappedImpl) Unwrap() Composer { return wi.parent }

// Wrap creates a new composer, converting the message to the
// appropriate Composer type, using the Convert() function, while
// preserving the parent composer. The Unwrap() function unwinds a
// stack of composers, flattening it into a single group composer.
func Wrap(parent Composer, msg any) Composer {
	switch {
	case parent == nil && msg == nil:
		return MakeKV()
	case msg == nil:
		return parent
	case parent == nil:
		return Convert(msg)
	default:
		return &wrappedImpl{
			parent:   parent,
			Composer: Convert(msg),
		}
	}
}

// Unwind takes a composer and, if it has been wrapped, unwraps it
// and produces a group composer of all the constituent messages. If
// there are group messages in the stack, they are added (flattened)
// in the new output group.
func Unwind(comp Composer) []Composer {
	switch c := comp.(type) {
	case *wrappedImpl:
		return fun.Unwind(comp)
	case *GroupComposer:
		return c.Messages()
	default:
		return []Composer{c}
	}
}
