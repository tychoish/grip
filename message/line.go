package message

import (
	"fmt"
	"strings"
)

type lineMessenger struct {
	lines   []any
	Base    `bson:"meta" json:"meta" yaml:"meta"`
	Message string `bson:"msg" json:"msg" yaml:"msg"`
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

	return m
}

func newLinesFromStrings(args []string) Composer {
	m := &lineMessenger{}
	m.lines = make([]any, 0, len(args))
	for _, arg := range args {
		if arg != "" {
			m.lines = append(m.lines, arg)
		}
	}

	return m
}

func (l *lineMessenger) Loggable() bool {
	switch {
	case l.Context.Len() > 0:
		return true
	case len(l.lines) > 0:
		return true
	default:
		return false
	}
}

func (l *lineMessenger) String() string {
	if l.Message == "" {
		l.resolve()
		if size := l.Context.Len(); size > 0 {
			l.Message = fmt.Sprintf("%s=%s %s", FieldsMsgName, l.Message, makeSimpleFieldsString(l.Context.Iterator(), true, size))
		}
	}
	return l.Message
}

func (l *lineMessenger) Raw() any {
	l.resolve()
	return struct {
		Msg string `bson:"msg" json:"msg" yaml:"msg"`
	}{
		// TODO export annotated fields
		Msg: l.Message,
	}
}

func (l *lineMessenger) resolve() {
	if l.Message == "" {
		l.Message = strings.TrimSpace(fmt.Sprintln(l.lines...))
		l.Collect()
	}
}
