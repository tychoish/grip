package message

import (
	"testing"

	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/grip/level"
)

func TestGroupComposer(t *testing.T) {
	t.Run("LevelsDefaultToInvalidButCannotBeMadeInvalid", func(t *testing.T) {
		cmp := BuildGroupComposer(
			MakeLines("on", "off"),
			MakeLines("one", "two"),
		)

		check.Equal(t, level.Invalid, cmp.Priority())
		cmp.SetPriority(level.Priority(200))
		check.Equal(t, level.Critical, cmp.Priority())
		cmp.SetPriority(level.Info)
		check.Equal(t, level.Info, cmp.Priority())
		cmp.SetPriority(level.Priority(200))
		check.Equal(t, level.Critical, cmp.Priority())
		cmp.SetPriority(level.Priority(200))
		check.Equal(t, level.Critical, cmp.Priority())

		nm := MakeString("higher")
		nm.SetPriority(level.Alert)
		cmp.Add(nm)
		check.Equal(t, level.Alert, cmp.Priority())

		cmp.SetPriority(level.Info)

		list := cmp.Messages()
		check.Equal(t, level.Info, list[0].Priority())
		check.Equal(t, level.Info, list[1].Priority())
		check.Equal(t, level.Info, list[2].Priority())
	})
	t.Run("GroupsCanGrow", func(t *testing.T) {
		cmp := BuildGroupComposer()
		check.True(t, !cmp.Loggable())
		check.True(t, len(cmp.Messages()) == 0)

		cmp.Add(MakeLines("one"))
		check.True(t, cmp.Loggable())
		check.True(t, len(cmp.Messages()) == 1)

		cmp.Append(MakeLines("two"), MakeString("three"))
		check.True(t, len(cmp.Messages()) == 3)
	})
	t.Run("Annotate", func(t *testing.T) {
		cmp := BuildGroupComposer(
			MakeFields(Fields{"hello": "world", "one": 1}),
			MakeFields(Fields{"goodbye": "moon", "two": 2}),
		)
		cmp.Annotate("mars", "venus")
		for idx, m := range cmp.Messages() {
			mp := m.Raw().(*dt.OrderedMap[string, any])
			val, ok := mp.Load("mars")
			t.Log(idx)
			check.True(t, ok)
			check.Equal(t, "venus", val.(string))
		}
	})
}
