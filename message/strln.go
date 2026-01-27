package message

import (
	"fmt"
	"iter"
	"strings"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/irt"
)

type strln struct {
	Base
	lines    []any
	rendered adt.Once[*strCache]
}

// MakeLines returns a message Composer roughly equivalent to
// fmt.Sprintln().
func MakeLines(args ...any) Composer                 { return NewLines(args) }
func NewLines[T any](args []T) Composer              { return newLines().with(iterToAny(irt.Slice(args))) }
func newLines() *strln                               { m := &strln{}; m.rendered.Set(m.render); return m }
func (m *strln) Loggable() bool                      { return m.rendered.Called() || m.size() > 0 }
func (m *strln) String() string                      { return m.rendered.Resolve().Message }
func (m *strln) Raw() any                            { return m.rendered.Resolve().Payload.Resolve() }
func (m *strln) size() int                           { return m.Context.Len() + len(m.lines) }
func (m *strln) with(s iter.Seq[any]) *strln         { m.lines = irt.Collect(removeEmpty(s)); return m }
func toAny[T any](in T) any                          { return in }
func iterToAny[T any](seq iter.Seq[T]) iter.Seq[any] { return irt.Convert(seq, toAny) }
func removeEmpty(seq iter.Seq[any]) iter.Seq[any]    { return irt.Remove(seq, filterEmpty) }
func filterEmpty(in any) bool                        { return in == nil || in == "" }

func (m *strln) render() *strCache {
	m.Collect()
	out := &strCache{Context: &m.Context}
	if m.RenderExtendedStrings && m.Context.Len() > 0 {
		out.Message = fmt.Sprintf("%s %s", strings.TrimSpace(fmt.Sprintln(m.lines...)), makeSimpleFieldsString(m.Context.Iterator(), true, m.Context.Len()))
	} else {
		out.Message = strings.TrimSpace(fmt.Sprintln(m.lines...))
	}
	out.Payload.Set(m.innerRaw)
	return out
}

func (m *strln) innerRaw() *strRendered {
	out := &strRendered{
		Msg: strings.TrimSpace(fmt.Sprintln(m.lines...)),
	}
	if size := m.Context.Len(); size > 0 {
		out.Context = irt.Collect2(m.Context.Iterator())
	}
	return out
}
