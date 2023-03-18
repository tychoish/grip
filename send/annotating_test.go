package send

import (
	"strings"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestAnnotatingSender(t *testing.T) {
	insend, err := NewInternalLogger("annotatingSender", LevelInfo{Threshold: level.Debug, Default: level.Debug})
	if err != nil {
		t.Fatal(err)
	}

	annotate := MakeAnnotating(insend, map[string]any{"a": "b"})

	annotate.Send(message.NewSimpleFields(level.Notice, message.Fields{"b": "a"}))
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
