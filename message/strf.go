package message

import (
	"fmt"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/irt"
)

type strf struct {
	Base
	template string
	args     []any
	rendered adt.Once[*strCache]
}

func (m *strf) Loggable() bool {
	return m.rendered.Called() || m.Context.Len() > 0 || m.template != "" || len(m.args) > 0
}
func (m *strf) String() string { return m.rendered.Resolve().Message }
func (m *strf) Raw() any       { return m.rendered.Resolve().Payload.Resolve() }

// MakeFormat returns a message.Composer roughly equivalent to an
// fmt.Sprintf().
func MakeFormat(base string, args ...any) Composer {
	m := &strf{
		template: base,
		args:     args,
	}
	m.rendered.Set(m.render)
	return m
}

func (m *strf) render() *strCache {
	m.Collect()
	out := &strCache{Context: &m.Context}
	if size := m.Context.Len(); size > 0 {
		out.Message = fmt.Sprintf("%s %s", fmt.Sprintf(m.template, m.args...), makeSimpleFieldsString(m.Context.Iterator(), true, size))
	} else {
		out.Message = fmt.Sprintf(m.template, m.args...)
	}
	out.Payload.Set(m.innerRaw)
	return out
}

func (m *strf) innerRaw() *strRendered {
	out := &strRendered{
		Msg: fmt.Sprintf(m.template, m.args...),
	}
	if size := m.Context.Len(); size > 0 {
		out.Context = irt.Collect2(m.Context.Iterator())
	}
	return out
}
