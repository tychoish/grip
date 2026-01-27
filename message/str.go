package message

import (
	"fmt"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
)

type str struct {
	Base
	content  string
	rendered adt.Once[*strCache]
}

func (m *str) Loggable() bool {
	return m.rendered.Called() || m.Context.Len() > 0 || m.content != ""
}
func (m *str) String() string { return m.rendered.Resolve().Message }
func (m *str) Raw() any       { return m.rendered.Resolve().Payload.Resolve() }

type strCache struct {
	Message string
	Payload adt.Once[*strRendered]
	Context *dt.OrderedMap[string, any]
}

type strRendered struct {
	Msg     string         `bson:"msg" json:"msg" yaml:"msg"`
	Context map[string]any `bson:"context,omitempty" json:"context,omitempty" yaml:"context,omitempty"`
}

// MakeString provides a basic message consisting of a single line.
func MakeString(m string) Composer {
	msg := &str{content: m}
	msg.rendered.Set(msg.render)
	return msg
}

func (m *str) render() *strCache {
	m.Collect()
	out := &strCache{Context: &m.Base.Context}
	if size := m.Context.Len(); size > 0 {
		out.Message = fmt.Sprintf("%s %s", m.content, makeSimpleFieldsString(m.Context.Iterator(), true, size))
	} else {
		out.Message = m.content
	}
	out.Payload.Set(m.innerRaw)
	return out
}

func (m *str) innerRaw() *strRendered {
	out := &strRendered{
		Msg: m.content,
	}
	if size := m.Context.Len(); size > 0 {
		out.Context = irt.Collect2(m.Context.Iterator())
	}
	return out
}
