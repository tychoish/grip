package message

type wrappedImpl struct {
	parent Composer
	Composer
}

func Wrap(parent Composer, msg interface{}) Composer {
	return &wrappedImpl{
		parent:   parent,
		Composer: Convert(msg),
	}
}

func ResolveWrapped(comp Composer) Composer {
	switch cp := comp.(type) {
	case Composer:
		return cp
	case *wrappedImpl:
		cps := []Composer{}

	UNWRAP:
		for {
			switch c := comp.(type) {
			case *wrappedImpl:
				cps = append(cps, unwindGroup(c.Composer)...)
				if c.parent == nil {
					break UNWRAP
				}
				comp = c.parent
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
