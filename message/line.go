package message

import (
	"fmt"
	"strings"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/irt"
)

type lineMessenger struct {
	Base
	lines    []any
	rendered adt.Once[*renderedString]
}

// MakeLines returns a message Composer roughly equivalent to
// fmt.Sprintln().
func MakeLines(args ...any) Composer {
	m := &lineMessenger{}
	m.lines = make([]any, 0, len(args))
	for _, arg := range args {
		if arg != nil && arg != "" {
			m.lines = append(m.lines, arg)
		}
	}
	m.rendered.Set(m.render)
	return m
}

func newLinesFromStrings(args []string) Composer {
	m := &lineMessenger{
		lines: irt.Collect(irt.Convert(irt.RemoveZeros(irt.Slice(args)), func(in string) any { return in })),
	}
	m.rendered.Set(m.render)
	return m
}

func (m *lineMessenger) Loggable() bool {
	return m.rendered.Called() || m.Context.Len() > 0 || len(m.lines) > 0
}
func (m *lineMessenger) String() string { return m.rendered.Resolve().Message }
func (m *lineMessenger) Raw() any       { return m.rendered.Resolve().Payload.Resolve() }

func (m *lineMessenger) render() *renderedString {
	m.Collect()
	out := &renderedString{
		Context: &m.Base.Context,
	}
	if size := m.Context.Len(); size > 0 {
		out.Message = fmt.Sprintf("%s %s", strings.TrimSpace(fmt.Sprintln(m.lines...)), makeSimpleFieldsString(m.Context.Iterator(), true, size))
	} else {
		out.Message = strings.TrimSpace(fmt.Sprintln(m.lines...))
	}
	out.Payload.Set(m.innerRaw)
	return out
}

func (m *lineMessenger) innerRaw() *stringishPayload {
	out := &stringishPayload{
		Msg: strings.TrimSpace(fmt.Sprintln(m.lines...)),
	}
	if size := m.Context.Len(); size > 0 {
		out.Context = irt.Collect2(m.Context.Iterator())
	}
	return out
}
