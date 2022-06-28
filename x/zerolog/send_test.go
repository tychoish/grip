package zerolog

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
)

func TestZeroSender(t *testing.T) {
	t.Run("NotPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("should not have panic'd")
			}

		}()
		NewSender("hello", send.LevelInfo{Threshold: level.Debug, Default: level.Info}, zerolog.Nop())
	})

}
