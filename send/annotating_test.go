package send

import (
	"strings"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestAnnotatingSender(t *testing.T) {
	insend := MakeInternal()
	insend.SetName("annotatingSender")
	insend.SetPriority(level.Debug)

	annotate := MakeAnnotating(insend, map[string]any{"a": "b"})
	m := message.MakeSimpleFields(message.Fields{"b": "a"})
	m.SetPriority(level.Info)
	annotate.Send(m)

	msg, ok := insend.GetMessageSafe()
	if !ok {
		t.Fatal("should get message")
	}
	if !strings.Contains(msg.Rendered, "a='b'") {
		t.Errorf("%q should contain %q", msg.Rendered, "a='b'")
	}
	if !strings.Contains(msg.Rendered, "b='a'") {
		t.Errorf("%q should contain %q", msg.Rendered, "b='a'")
	}

}
