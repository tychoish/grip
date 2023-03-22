package message

import (
	"testing"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/grip/level"
)

func TestFunctionMessage(t *testing.T) {
	t.Run("SetPriority", func(t *testing.T) {
		p := MakeProducer(func() Composer { return MakeString("works") })
		assert.True(t, !p.Priority().IsValid())
		p.SetPriority(level.Error)
		assert.True(t, p.Priority().IsValid())
		assert.True(t, p.(*composerProducerMessage).cached == nil)
		check.True(t, p.Loggable()) // calse resolve
		assert.True(t, p.(*composerProducerMessage).cached != nil)
		assert.Equal(t, level.Error, p.(*composerProducerMessage).cached.Priority())
		p.SetPriority(level.Info)
		assert.Equal(t, level.Info, p.(*composerProducerMessage).cached.Priority())

		check.Equal(t, "works", p.String())
	})

}
