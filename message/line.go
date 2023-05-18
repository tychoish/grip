package message

import (
	"fmt"
	"strings"
)

type lineMessenger struct {
	lines   []any
	Base    `bson:"meta" json:"meta" yaml:"meta"`
	Message string `bson:"msg" json:"msg" yaml:"msg"`

	fm *fieldMessage
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
	case (l.fm != nil && l.fm.Loggable()):
		return true
	case len(l.Base.Context) > 0:
		return true
	case len(l.lines) > 0:
		return true
	default:
		return false
	}
}

func (l *lineMessenger) String() string {
	switch {
	case l.fm != nil:
		return l.fm.String()
	case len(l.Base.Context) > 0:
		l.setupField()
		return l.fm.String()
	case l.Message == "":
		l.resolve()
	}

	return l.Message
}

func (l *lineMessenger) Raw() any {
	switch {
	case l.fm != nil:
		return l.fm.Raw()
	case len(l.Base.Context) > 0:
		l.setupField()
		return l.fm.Raw()
	case l.IncludeMetadata:
		l.resolve()
		return l
	default:
		l.resolve()
		return struct {
			Msg string `bson:"msg" json:"msg" yaml:"msg"`
		}{
			Msg: l.Message,
		}
	}
}

func (l *lineMessenger) resolve() {
	if l.Message == "" {
		l.Message = strings.TrimSpace(fmt.Sprintln(l.lines...))
		l.Collect()
	}
}

func (l *lineMessenger) setupField() {
	l.resolve()
	l.fm = &fieldMessage{
		fields:  l.Base.Context,
		Base:    l.Base,
		message: l.Message,
	}
}

func (l *lineMessenger) Annotate(k string, v any) {
	if l.fm == nil {
		l.Base.Annotate(k, v)
		return
	}
	l.fm.Annotate(k, v)
}

func (l *lineMessenger) SetOption(opts ...Option) {
	if l.fm == nil {
		l.Base.SetOption(opts...)
		return
	}
	l.fm.SetOption(opts...)
}
