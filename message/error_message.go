package message

import (
	"errors"
	"iter"
	"sync"

	"github.com/tychoish/fun/irt"
)

type errorComposerWrap struct {
	err error
	Composer
	populate sync.Once
}

func JoinErrors(seq iter.Seq[error]) Composer {
	return MakeFuture(func() error { return irt.JoinErrors(seq) })
}

// WrapError wraps an error and creates a composer converting the
// argument into a composer in the same manner as the front end logging methods.
func WrapError(err error, m any) Composer {
	return &errorComposerWrap{
		err: err,
		Composer: MakeFuture(func() Composer {
			c := Convert(m)
			c.SetOption(OptionMessageIsNotStructuredField)
			return c
		}),
	}
}

// WrapErrorf wraps an error and creates a composer using a
// Sprintf-style formated composer.
func WrapErrorf(err error, msg string, args ...any) Composer {
	return WrapError(err, MakeFormat(msg, args...))
}

func (m *errorComposerWrap) String() string {
	m.populate.Do(func() { m.Composer.Annotate("error", m.err) })

	return m.Composer.String()
}

func (*errorComposerWrap) Structured() bool           { return true }
func (m *errorComposerWrap) Error() string            { return m.String() }
func (m *errorComposerWrap) Unwind() Composer         { return m.Composer } // nolint
func (m *errorComposerWrap) Is(err error) bool        { return errors.Is(m.err, err) }
func (m *errorComposerWrap) As(err any) bool          { return errors.As(m.err, err) }
func (m *errorComposerWrap) Loggable() bool           { return m.err != nil && m.Composer.Loggable() }
func (m *errorComposerWrap) Annotate(k string, v any) { m.Composer.Annotate(k, v) }

func (m *errorComposerWrap) Raw() any {
	m.populate.Do(func() { m.Composer.Annotate("error", m.err) })

	return m.Composer.Raw()
}
