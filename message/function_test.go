package message

import (
	"testing"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/grip/level"
)

func TestFunctionMessage(t *testing.T) {
	t.Run("SetPriority", func(t *testing.T) {
		p := MakeFuture(func() Composer { return MakeString("works") })
		p.SetPriority(level.Error)
		assert.True(t, p.Priority() == level.Error)
		assert.True(t, p.(*composerFutureMessage).cached == nil)
		check.True(t, p.Loggable()) // calse resolve
		assert.True(t, p.(*composerFutureMessage).cached != nil)
		assert.Equal(t, level.Error, p.(*composerFutureMessage).cached.Priority())

		check.Equal(t, "works", p.String())
	})

}
