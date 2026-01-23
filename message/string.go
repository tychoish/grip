package message

import (
	"fmt"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
)

type stringMessage struct {
	rendered adt.Once[*renderedString]
	Message  string
	Base
}

func (m *stringMessage) Loggable() bool {	return m.rendered.Called() || m.Context.Len() > 0 || m.Message != ""}
func (m *stringMessage) String() string { return m.rendered.Resolve().Message }
func (m *stringMessage) Raw() any {	return m.rendered.Resolve().Payload.Resolve()}

type renderedString struct {
	Message string
	Payload adt.Once[*stringishPayload]
	Context *dt.OrderedMap[string, any]
}

type stringishPayload struct {
	Msg     string         `bson:"msg" json:"msg" yaml:"msg"`
	Context map[string]any `bson:"context,omitempty" json:"context,omitempty" yaml:"context,omitempty"`
}

// MakeString provides a basic message consisting of a single line.
func MakeString(m string) Composer {
	msg := &stringMessage{Message: m}
	msg.rendered.Set(msg.render)
	return msg
}

func (m *stringMessage) render() *renderedString {
	m.Collect()
	out := &renderedString{
		Context: &m.Base.Context,
	}
	if size := m.Context.Len(); size > 0 {
		out.Message = fmt.Sprintf("%s %s", m.Message, makeSimpleFieldsString(m.Context.Iterator(), true, size))
	} else {
		out.Message = m.Message
	}
	out.Payload.Set(m.innerRaw)
	return out
}

func (m *stringMessage) innerRaw() *stringishPayload {
	out := &stringishPayload{
		Msg: m.Message,
	}
	if size := m.Context.Len(); size > 0 {
		out.Context = irt.Collect2(m.Context.Iterator())
	}
	return out
}

