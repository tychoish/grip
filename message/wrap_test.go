package message

import (
	"testing"

	"github.com/tychoish/grip/level"
)

func sizeOfGroup(t *testing.T, comp Composer) int {
	t.Helper()

	gc, ok := comp.(*GroupComposer)
	if !ok {
		t.Fatalf("%T is not a group composer", comp)
	}

	return len(gc.messages)
}

func TestWrap(t *testing.T) {
	t.Run("Unwrap", func(t *testing.T) {
		comp := MakeString("hello")
		comp = Wrap(comp, MakeString("world"))

		if sizeOfGroup(t, Unwrap(comp)) != 2 {
			t.Fatal("wrap message does not unwrap correctly")

		}
	})
	t.Run("LevelPreservingSimple", func(t *testing.T) {
		comp := MakeString("hello")
		comp.SetPriority(level.Alert)
		comp = Wrap(comp, MakeString("world"))
		comp.SetPriority(level.Info)

		comp = Unwrap(comp)
		if sizeOfGroup(t, comp) != 2 {
			t.Fatal("wrap message does not unwrap correctly")
		}

		msgs := comp.(*GroupComposer).Messages()

		if msgs[1].Priority() != level.Alert {
			t.Errorf("wrapped message priority not maintained: %s", msgs[0].Priority())
		}
		if msgs[0].Priority() != level.Info {
			t.Errorf("outter message priority not maintained: %s", msgs[1].Priority())
		}
	})
	t.Run("MultiWrap", func(t *testing.T) {
		comp := MakeString("hello")
		comp = Wrap(comp, Wrap(MakeString("world"), "earthling"))

		comp = Unwrap(comp)

		if size := sizeOfGroup(t, comp); size != 3 {
			t.Fatalf("incorrect number of messages: %d", size)
		}

		msgs := comp.(*GroupComposer).Messages()
		set := map[string]struct{}{}
		for _, m := range msgs {
			set[m.String()] = struct{}{}
		}
		if len(set) != 3 {
			t.Fatal("non-unique messages")
		}
	})
	t.Run("Nil", func(t *testing.T) {
		out := unwindGroup(nil)
		if out == nil {
			t.Fatal("must produce slice")
		}

		if len(out) != 0 {
			t.Fatal("should not propagate nil")
		}
	})

}
