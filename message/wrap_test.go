package message

import (
	"testing"

	"github.com/tychoish/fun"
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
	t.Run("Check", func(t *testing.T) {
		t.Run("Wrapped", func(t *testing.T) {
			comp := MakeString("hello")
			comp = Wrap(comp, MakeString("world"))

			if !IsWrapped(comp) {
				t.Fatal("wrapped message not detected")
			}
		})
		t.Run("NilParent", func(t *testing.T) {
			if IsWrapped(Wrap(nil, MakeString("hello"))) {
				t.Fatal("nil parent wrapped messagese aren't wrapped")
			}
		})
		t.Run(" goUnwrapped", func(t *testing.T) {
			if IsWrapped(MakeString("hello")) {
				t.Fatal("unwrapped messages should not be detected")
			}
		})
	})

	t.Run("Unwrap", func(t *testing.T) {
		comp := MakeString("hello")
		comp = Wrap(comp, MakeString("world"))

		if sizeOfGroup(t, Unwrap(comp)) != 2 {
			t.Fatal("wrap message does not unwrap correctly")

		}
	})
	t.Run("LevelPreservingSimple", func(t *testing.T) {
		comp := MakeString("hello")
		if err := comp.SetPriority(level.Alert); err != nil {
			t.Fatal(err)
		}
		comp = Wrap(comp, MakeString("world"))
		if err := comp.SetPriority(level.Info); err != nil {
			t.Fatal(err)
		}

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

		if size := sizeOfGroup(t, comp); size != 2 {
			t.Log(comp)
			t.Fatalf("incorrect number of messages: %d", size)
		}

		msgs := comp.(*GroupComposer).Messages()
		set := map[string]struct{}{}
		for _, m := range msgs {
			set[m.String()] = struct{}{}
		}
		if len(set) != len(msgs) {
			t.Fatal("non-unique messages", len(set), len(msgs))
		}
	})
	t.Run("Nil", func(t *testing.T) {
		out := fun.Unwind[Composer](nil)
		if out == nil {
			t.Fatal("must produce slice")
		}

		if len(out) != 1 {
			t.Fatal("nil messages are weird but shouldn't be dropped")
		}
	})

}
