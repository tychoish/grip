package message

type wrappedImpl struct {
	parent Composer
	Composer
}

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

// Unwrap takes a composer and, if it has been wrapped, unwraps it
// and produces a group composer of all the constituent messages. If
// there are group messages in the stack, they are added (flattened)
// in the new output group.
func Unwrap(comp Composer) Composer {
	switch cp := comp.(type) {
	case Composer:
		return cp
	case *wrappedImpl:
		cps := []Composer{cp.Composer}

		next := cp.parent
	UNWRAP:
		for {
			switch c := next.(type) {
			case *wrappedImpl:
				cps = append(cps, unwindGroup(c.Composer)...)
				if c.parent == nil {
					break UNWRAP
				}
				next = c.parent
			case Composer:
				cps = append(cps, unwindGroup(cp)...)
				break UNWRAP
			default:
				break UNWRAP
			}

		}

		return MakeGroupComposer(cps)
	default:
		return nil
	}
}

func unwindGroup(comp Composer) []Composer {
	switch cp := comp.(type) {
	case *GroupComposer:
		return cp.messages
	case Composer:
		return []Composer{comp}
	default:
		return []Composer{}
	}
}
