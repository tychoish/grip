package zerolog

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestZeroSender(t *testing.T) {
	t.Run("NotPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("should not have panic'd")
			}

		}()
		s := MakeSender(zerolog.Nop())
		if s == nil {
			t.Fatal("s should not be nil")
		}

	})

}
