package message

import (
	"fmt"
)

type formatMessenger struct {
	base    string
	args    []any
	Base    `bson:"meta" json:"meta" yaml:"meta"`
	Message string `bson:"msg" json:"msg" yaml:"msg"`
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
func (f *formatMessenger) Raw() any {
	_ = f.String()

	if f.SkipMetadata {
		return struct {
			Message string `bson:"msg" json:"msg" yaml:"msg"`
		}{
			Message: f.Message,
		}
	}

	if !f.SkipCollection {
		f.Collect()
	}

	return f
}
