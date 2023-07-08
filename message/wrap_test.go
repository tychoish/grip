package message

import (
	"testing"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/grip/level"
)

func sizeOfGroup(t *testing.T, comp Composer) int {
	t.Helper()

	gc, ok := comp.(*GroupComposer)
	if !ok {
		t.Fatalf("%T is not a group composer", comp)
	}

	var num int

	gc.messages.With(func(list *dt.List[Composer]) {
		num = list.Len()
	})

	return num
}

func TestWrap(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		t.Run("Wrapped", func(t *testing.T) {
			comp := MakeString("hello")
			comp = Wrap(comp, MakeString("world"))

			if !IsMulti(comp) {
				t.Fatal("wrapped message not detected")
			}
		})
		t.Run("NilParent", func(t *testing.T) {
			if IsMulti(Wrap(nil, MakeString("hello"))) {
				t.Fatal("nil parent wrapped messagese aren't wrapped")
			}
		})
		t.Run(" goUnwrapped", func(t *testing.T) {
			if IsMulti(MakeString("hello")) {
				t.Fatal("unwrapped messages should not be detected")
			}
		})
	})

	t.Run("Unwrap", func(t *testing.T) {
		comp := MakeString("hello")
		comp = Wrap(comp, MakeString("world"))

		if sizeOfGroup(t, MakeGroupComposer(Unwind(comp))) != 2 {
			t.Log(MakeGroupComposer(Unwind(comp)))
			t.Log(comp)
			t.Fatal("wrap message does not unwrap correctly")

		}
	})
	t.Run("LevelPreservingSimple", func(t *testing.T) {
		comp := MakeString("hello")
		comp.SetPriority(level.Alert)

		comp = Wrap(comp, MakeString("world"))
		comp.SetPriority(level.Info)

		comp = MakeGroupComposer(Unwind(comp))
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
		t.Log(comp)

		ucomp := Unwind(comp)
		comp = MakeGroupComposer(Unwind(comp))

		if size := sizeOfGroup(t, comp); size != 2 {
			t.Log(comp)
			t.Errorf("incorrect number of messages: %d", size)
		}

		msgs := comp.(*GroupComposer).Messages()
		set := map[string]struct{}{}
		for _, m := range msgs {
			set[m.String()] = struct{}{}
		}
		if len(set) != len(msgs) {
			t.Log(comp)
			t.Log(ucomp)
			t.Log(set)
			t.Log(msgs)
			t.Fatal("non-unique messages", len(set), len(msgs))
		}
	})
	t.Run("Nil", func(t *testing.T) {
		out := Unwind(nil)

		if len(out) != 1 {
			t.Fatal("nil messages are weird but shouldn't be dropped")
		}
	})

}
