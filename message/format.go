package message

import (
	"fmt"
)

type formatMessenger struct {
	Message string `bson:"msg" json:"msg" yaml:"msg"`
	Base    `bson:"meta" json:"meta" yaml:"meta"`

	base string
	args []any
	fm   *fieldMessage
}

// MakeFormat returns a message.Composer roughly equivalent to an
// fmt.Sprintf().
func MakeFormat(base string, args ...any) Composer {
	return &formatMessenger{
		base: base,
		args: args,
	}
}

func (f *formatMessenger) setupField() {
	f.fm = &fieldMessage{
		fields:  f.Base.Context,
		Base:    f.Base,
		message: f.Message,
	}
}

func (f *formatMessenger) setupMessage() {
	if f.Message == "" {
		f.Message = fmt.Sprintf(f.base, f.args...)
	}
}

func (f *formatMessenger) String() string {
	if f.fm != nil {
		return f.fm.String()
	}

	f.setupMessage()

	if len(f.Base.Context) > 0 {
		f.setupField()
		return f.fm.String()
	}
	return f.Message
}

func (f *formatMessenger) Annotate(k string, v any) {
	if f.fm == nil {
		f.Base.Annotate(k, v)
		return
	}
	f.fm.Annotate(k, v)
}

func (f *formatMessenger) SetOption(opts ...Option) {
	if f.fm == nil {
		f.Base.SetOption(opts...)
		return
	}
	f.fm.SetOption(opts...)
}

func (f *formatMessenger) Loggable() bool {
	return f.base != "" || f.Message != "" || len(f.Base.Context) > 0 || (f.fm != nil && f.fm.Loggable())
}

func (f *formatMessenger) Raw() any {
	if f.fm != nil {
		return f.fm.Raw()
	}

	f.setupMessage()

	if len(f.Base.Context) > 0 {
		f.setupField()
		return f.fm.Raw()
	}

	if f.SkipMetadata {
		return stringMessage{Message: f.Message}
	}

	if !f.SkipCollection {
		f.Collect()
	}

	return f
}
