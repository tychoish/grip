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
func Wrap(parent Composer, msg interface{}) Composer {
	return &wrappedImpl{
		parent:   parent,
		Composer: Convert(msg),
	}
}

// IsWrapped returns true if the composer is wrapped *and* if the
// parent composer is non-nil.
func IsWrapped(c Composer) bool { wc, ok := c.(*wrappedImpl); return ok && wc.parent != nil }

// Unwrap takes a composer and, if it has been wrapped, unwraps it
// and produces a group composer of all the constituent messages. If
// there are group messages in the stack, they are added (flattened)
// in the new output group.
func Unwrap(comp Composer) Composer {
	if fun.IsWrapped(comp) {
		return MakeGroupComposer(fun.Unwind(comp))
	}
	return comp
}
