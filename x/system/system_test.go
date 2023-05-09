package system

import (
	"fmt"
	"testing"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/grip/level"
)

func TestLogging(t *testing.T) {
	for i := -10; i < 256; i++ {
		t.Run(fmt.Sprintf("Level_%d", i), func(t *testing.T) {
			assert.NotPanic(t, func() {
				convertPrioritySystemd(level.Priority(i), 0)
			})
		})
	}
}
