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

	return false
}

func (l *lineMessenger) String() string {
	if l.Message == "" {
		l.Message = strings.Trim(fmt.Sprintln(l.lines...), "\n ")
	}

	return l.Message
}

func (l *lineMessenger) Raw() any {
	if !l.SkipCollection {
		l.Collect()
	}
	m := l.String()

	if l.SkipMetadata {
		return struct {
			Msg string `bson:"msg" json:"msg" yaml:"msg"`
		}{
			Msg: m,
		}
	}

	return l
}
