package message

import (
	"fmt"

	"github.com/tychoish/grip/level"
)

type formatMessenger struct {
	base    string
	args    []any
	Base    `bson:"metadata" json:"metadata" yaml:"metadata"`
	Message string `bson:"message" json:"message" yaml:"message"`
}

// NewFormat takes arguments as fmt.Sprintf(), and returns
// an object that only runs the format operation as part of the
// String() method.
func NewFormat(p level.Priority, base string, args ...any) Composer {
	m := &formatMessenger{
		base: base,
		args: args,
	}
	_ = m.SetPriority(p)

	return m
}

// MakeFormat returns a message.Composer roughly equivalent to an
// fmt.Sprintf().
func MakeFormat(base string, args ...any) Composer {
	return &formatMessenger{
		base: base,
		args: args,
	}
}

func (f *formatMessenger) String() string {
	if f.Message == "" {
		f.Message = fmt.Sprintf(f.base, f.args...)
	}

	return f.Message
}

func (f *formatMessenger) Loggable() bool { return f.base != "" }
func (f *formatMessenger) Raw() any       { f.Collect(); _ = f.String(); return f }
