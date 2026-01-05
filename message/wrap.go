package message

import (
	"fmt"
)

type wrappedImpl struct {
	parent Composer
	Composer
	cached string
}

func (wi *wrappedImpl) Unwind() Composer { return wi.parent }

// Wrap creates a new composer, converting the message to the
// appropriate Composer type, using the Convert() function, while
// preserving the parent composer. The Unwrap() function unwinds a
// stack of composers, flattening it into a single group composer.
func Wrap(parent Composer, msg any) Composer {
	switch {
	case parent == nil && msg == nil:
		return Noop()
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

func (wi *wrappedImpl) String() string {
	if wi.cached != "" {
		return wi.cached
	}

	wi.cached = wi.Composer.String()
	if wi.parent != nil {
		wi.cached = fmt.Sprintf("%s\n%s", wi.cached, wi.parent.String())
	}

	return wi.cached
}

func (wi *wrappedImpl) Raw() any {
	msgs := Unwind(wi)
	switch len(msgs) {
	case 0:
		return nil
	case 1:
		return msgs[0].Raw()
	default:
		return msgs
	}
}

func IsMulti(comp Composer) bool {
	switch comp.(type) {
	case *wrappedImpl:
		return true
	case *GroupComposer:
		return true
	case interface{ Unwind() Composer }:
		return true
	default:
		return false
	}
}

// Unwind takes a composer and, if it has been wrapped, unwraps it
// and produces a group composer of all the constituent messages. If
// there are group messages in the stack, they are added (flattened)
// in the new output group.
func Unwind(comp Composer) []Composer {
	switch c := comp.(type) {
	case *wrappedImpl:
		var out []Composer
		out = append(out, c.Composer)

		var last Composer
		last = c.parent
		for {
			next, ok := last.(*wrappedImpl)
			if !ok {
				out = append(out, last)
				break
			}
			out = append(out, next.Composer)
			last = next.parent
		}
		return out
	case *GroupComposer:
		return c.Messages()
	case interface{ Unwind() Composer }:
		return []Composer{c.Unwind()}
	case interface{ Unwind() []Composer }:
		return c.Unwind()
	case interface{ Unwrap() []Composer }:
		return c.Unwrap()
	case interface{ Unwrap() Composer }:
		return []Composer{c.Unwrap()}
	default:
		return []Composer{c}
	}
}
