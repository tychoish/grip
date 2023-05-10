package message

import (
	"context"
	"math"
	"strings"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/seq"
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
	messages *adt.Synchronized[*seq.List[Composer]]
	cache    *adt.Atomic[string]
}

// BuildGroupComposer provides a variadic interface for creating a
// GroupComposer.
func BuildGroupComposer(msgs ...Composer) *GroupComposer {
	return MakeGroupComposer(msgs)
}

// MakeGroupComposer returns a GroupComposer object from a slice of
// Composers.
func MakeGroupComposer(msgs []Composer) *GroupComposer {
	gc := &GroupComposer{
		messages: adt.NewSynchronized(&seq.List[Composer]{}),
		cache:    adt.NewAtomic(""),
	}

	gc.messages.With(func(list *seq.List[Composer]) { list.Append(msgs...) })

	return gc
}

// String satisfies the fmt.Stringer interface, and returns a string
// of the string form of all constituent composers joined with a newline.
func (g *GroupComposer) String() string {
	if cache := g.cache.Get(); cache != "" {
		return cache
	}
	g.messages.With(func(list *seq.List[Composer]) {
		if cache := g.cache.Get(); cache != "" {
			return
		}

		out := make([]string, 0, list.Len())
		iter := seq.ListValues(list.Iterator())
		for {
			val, err := fun.IterateOneBlocking(iter)
			if err != nil {
				break
			}
			if val != nil && val.Loggable() {
				out = append(out, val.String())
			}
		}
		g.cache.Set(strings.Join(out, "\n"))
	})

	return g.cache.Get()
}

// Raw returns a slice of interfaces containing the raw form of all
// the constituent composers.
func (g *GroupComposer) Raw() any {
	var out []any
	g.messages.With(func(list *seq.List[Composer]) {
		out = make([]any, 0, list.Len())

		iter := seq.ListValues(list.Iterator())
		for {
			m, err := fun.IterateOneBlocking(iter)
			if err != nil {
				break
			}

			out = append(out, m.Raw())
		}
	})

	return out
}

// Loggable returns true if at least one of the constituent Composers
// is loggable.
func (g *GroupComposer) Loggable() bool {
	var isLoggable bool

	g.messages.With(func(list *seq.List[Composer]) {
		iter := seq.ListValues(list.Iterator())
		for {
			m, err := fun.IterateOneBlocking(iter)
			if err != nil {
				break
			}
			if m.Loggable() {
				isLoggable = true
				break
			}
		}
	})

	return isLoggable
}

func (g *GroupComposer) Structured() bool {
	var isStructured bool
	g.messages.With(func(list *seq.List[Composer]) {
		iter := seq.ListValues(list.Iterator())
		for {
			m, err := fun.IterateOneBlocking(iter)
			if err != nil {
				break
			}
			isStructured = m.Structured()
			if isStructured {
				break
			}
		}
	})

	return isStructured
}

// Priority returns the highest priority of the constituent Composers.
func (g *GroupComposer) Priority() level.Priority {
	var highest level.Priority

	g.messages.With(func(list *seq.List[Composer]) {
		iter := seq.ListValues(list.Iterator())
		for {
			m, err := fun.IterateOneBlocking(iter)
			if err != nil {
				break
			}
			pri := m.Priority()
			if pri > highest {
				highest = pri
			}
			if highest == math.MaxUint8 {
				break
			}
		}
	})

	return highest
}

// SetPriority sets the priority of all constituent Composers *only*
// if the existing level is unset (or otherwise invalid), and will
// *not* unset the level of a constituent composer.
func (g *GroupComposer) SetPriority(l level.Priority) {
	g.messages.With(func(list *seq.List[Composer]) {
		_ = fun.WorkerFunc(func(ctx context.Context) error {
			return fun.Observe(ctx, seq.ListValues(list.Iterator()), func(m Composer) {
				m.SetPriority(l)
			})
		}).Block()
	})

	return
}

// Messages returns a the underlying collection of messages.
func (g *GroupComposer) Messages() []Composer {
	var out []Composer
	g.messages.With(func(list *seq.List[Composer]) {
		iter := seq.ListValues(list.Iterator())
		out = make([]Composer, 0, list.Len())
		for {
			val, err := fun.IterateOneBlocking(iter)
			if err != nil {
				break
			}
			out = append(out, val)
		}
	})

	return out
}

func (g *GroupComposer) Unwrap() Composer {
	var out Composer

	g.messages.With(func(list *seq.List[Composer]) {
		switch list.Len() {
		case 0:
			return
		case 1:
			out = list.Front().Value()
		case 2:
			out = &wrappedImpl{
				parent:   list.Front().Value(),
				Composer: list.Back().Value(),
			}
		default:
			iter := seq.ListValues(list.Iterator())
			val, err := fun.IterateOneBlocking(iter)
			if err != nil {
				return
			}

			wrapped := &wrappedImpl{parent: val}

			for {
				val, err := fun.IterateOneBlocking(iter)
				if err != nil {
					break
				}
				wrapped.Composer = val
				wrapped = &wrappedImpl{parent: wrapped}
			}
			out = wrapped
		}
	})
	return out
}

// Extend makes it possible to add a group of messages to an existing
// group composer.
func (g *GroupComposer) Extend(msg []Composer) {
	g.messages.With(func(list *seq.List[Composer]) {
		g.cache.Set("")
		list.Append(msg...)
	})
}

// Add supports adding messages to an existing group composer.
func (g *GroupComposer) Add(msg Composer) { g.Append(msg) }

// Append provides a variadic alternative to the Extend method.
func (g *GroupComposer) Append(msgs ...Composer) { g.Extend(msgs) }

// Annotate calls the Annotate method of every non-nil component
// Composer.
func (g *GroupComposer) Annotate(k string, v any) {
	g.messages.With(func(list *seq.List[Composer]) {
		_ = fun.WorkerFunc(func(ctx context.Context) error {
			return fun.Observe(ctx, seq.ListValues(list.Iterator()), func(m Composer) {
				m.Annotate(k, v)
			})
		}).Block()
	})
}

func (g *GroupComposer) Option(opts ...Option) {
	g.messages.With(func(list *seq.List[Composer]) {
		_ = fun.WorkerFunc(func(ctx context.Context) error {
			return fun.Observe(ctx, seq.ListValues(list.Iterator()), func(m Composer) {
				m.Option(opts...)
			})
		}).Block()
	})
}
