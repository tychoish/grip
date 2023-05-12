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
		if arg != nil {
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
	for idx := range l.lines {
		if l.lines[idx] != "" {
			return true
		}
	}

	return len(l.Base.Context) > 0 || (l.fm != nil && l.fm.Loggable())
}

func (l *lineMessenger) String() string {
	l.resolve()
	if l.fm != nil {
		return l.fm.String()
	} else if len(l.Base.Context) > 0 {
		l.setupField()
		return l.fm.String()
	}

	return l.Message
}

func (l *lineMessenger) Raw() any {
	l.resolve()
	if l.fm != nil {
		return l.fm.Raw()
	} else if len(l.Base.Context) > 0 {
		l.setupField()
		return l.fm.Raw()
	}

	if l.SkipMetadata {
		return struct {
			Msg string `bson:"msg" json:"msg" yaml:"msg"`
		}{
			Msg: l.Message,
		}
	}

	if !l.SkipCollection {
		l.Collect()
	}

	return l
}

func (l *lineMessenger) resolve() {
	if l.Message == "" {
		l.Message = strings.Trim(fmt.Sprintln(l.lines...), "\n ")
	}
}

func (l *lineMessenger) setupField() {
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
