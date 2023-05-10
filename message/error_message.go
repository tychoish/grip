package message

import (
	"errors"
	"fmt"
	"sync"
)

type errorComposerWrap struct {
	err    error
	cached string
	Composer
	populate sync.Once
}

// WrapError wraps an error and creates a composer converting the
// argument into a composer in the same manner as the front end logging methods.
func WrapError(err error, m any) Composer {
	return &errorComposerWrap{
		err:      err,
		Composer: MakeProducer(func() Composer { return Convert(m) }),
	}
}

// WrapErrorf wraps an error and creates a composer using a
// Sprintf-style formated composer.
func WrapErrorf(err error, msg string, args ...any) Composer {
	return WrapError(err, MakeFormat(msg, args...))
}

func (m *errorComposerWrap) String() string {
	if m.cached == "" {
		m.cached = fmt.Sprintf("%s: %v", m.Composer, m.err)
	}

	return m.cached
}

func (m *errorComposerWrap) Error() string            { return m.String() }
func (m *errorComposerWrap) Unwrap() Composer         { return m.Composer }
func (m *errorComposerWrap) Is(err error) bool        { return errors.Is(m.err, err) }
func (m *errorComposerWrap) As(err any) bool          { return errors.As(m.err, err) }
func (m *errorComposerWrap) Loggable() bool           { return m.err != nil && m.Composer.Loggable() }
func (m *errorComposerWrap) Annotate(k string, v any) { m.Composer.Annotate(k, v) }

func (m *errorComposerWrap) Raw() any {
	m.populate.Do(func() { m.Composer.Annotate("error", m.err) })

	return m.Composer.Raw()
}
