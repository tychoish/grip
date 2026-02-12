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

	internal := NewInternal(1)
	internal.SetPriority(level.Info)
	std := MakeStandard(internal)
	std.Print(printableMessage)

	testt.Logf(t, "std=[%T]%+v", std, std)
	assert.True(t, internal.HasMessage())
	msg := internal.GetMessage()
	check.Equal(t, msg.Rendered, printableMessage)

	wrapped := FromStandard(std)
	printableMessage = strings.Repeat("hi grip!", 10)
	tosend := message.MakeString(printableMessage)
	tosend.SetPriority(level.Alert)
	wrapped.Send(tosend)
	testt.Logf(t, "wrapped=[%T]%+v", wrapped, wrapped)

	assert.True(t, internal.HasMessage())
	msg = internal.GetMessage()
	check.Equal(t, msg.Rendered, printableMessage)
}
