package send

import (
	"strings"
	"testing"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/testt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestStdLogging(t *testing.T) {
	printableMessage := strings.Repeat("hello world", 8)

	internal := NewInternalLogger(1)
	assert.NotError(t, internal.SetLevel(LevelInfo{Default: level.Info, Threshold: level.Info}))
	std := MakeStandard(internal)
	std.Print(printableMessage)
	testt.Logf(t, "std=%+v", std)
	assert.True(t, internal.HasMessage())
	msg := internal.GetMessage()
	check.Equal(t, msg.Rendered, printableMessage)

	wrapped := FromStandard(std)
	wrapped.Send(message.MakeString("hi grip"))
	assert.True(t, internal.HasMessage())
	msg = internal.GetMessage()
	check.Equal(t, msg.Rendered, "hi grip")
}
