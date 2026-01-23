package message

import (
	"fmt"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/irt"
)

type formatMessenger struct {
	Base
	base     string
	args     []any
	rendered adt.Once[*renderedString]
}

func (m *formatMessenger) Loggable() bool {
	return m.rendered.Called() || m.Context.Len() > 0 || m.base != "" || len(m.args) > 0
}
func (m *formatMessenger) String() string { return m.rendered.Resolve().Message }
func (m *formatMessenger) Raw() any       { return m.rendered.Resolve().Payload.Resolve() }

// MakeFormat returns a message.Composer roughly equivalent to an
// fmt.Sprintf().
func MakeFormat(base string, args ...any) Composer {
	m := &formatMessenger{
		base: base,
		args: args,
	}
	m.rendered.Set(m.render)
	return m
}

func (m *formatMessenger) render() *renderedString {
	m.Collect()
	out := &renderedString{
		Context: &m.Base.Context,
	}
	if size := m.Context.Len(); size > 0 {
		out.Message = fmt.Sprintf("%s %s", fmt.Sprintf(m.base, m.args...), makeSimpleFieldsString(m.Context.Iterator(), true, size))
	} else {
		out.Message = fmt.Sprintf(m.base, m.args...)
	}
	out.Payload.Set(m.innerRaw)
	return out
}

func (m *formatMessenger) innerRaw() *stringishPayload {
	out := &stringishPayload{
		Msg: fmt.Sprintf(m.base, m.args...),
	}
	if size := m.Context.Len(); size > 0 {
		out.Context = irt.Collect2(m.Context.Iterator())
	}
	return out
}
