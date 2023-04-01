package level

import (
	"strings"
	"testing"

	"github.com/tychoish/fun/testt"
)

func assert(t *testing.T, cond bool, args ...any) {
	t.Helper()
	if !cond {
		if len(args) > 0 {
			t.Log(args...)
		}
		t.Error("condition was false")
	}
}

func TestLevel(t *testing.T) {
	t.Run("StringForm", func(t *testing.T) {
		t.Run("Levels", func(t *testing.T) {
			for _, val := range []Priority{Emergency, Alert, Critical, Error, Warning, Notice, Info, Debug, Trace, Invalid} {
				assert(t, !strings.HasPrefix(val.String(), "<"), val)
				assert(t, !strings.HasSuffix(val.String(), ">"), val)
			}
		})
		t.Run("InBetween", func(t *testing.T) {
			for i := 1; i < 101; i++ {
				if i%25 == 0 {
					continue
				}
				assert(t, strings.HasPrefix(Priority(i).String(), "level.Priority<"), i)
				assert(t, strings.HasSuffix(Priority(i).String(), ">"), i)
			}
		})

	})
	t.Run("RoundTrip", func(t *testing.T) {
		for i := 1; i < 101; i++ {
			str := FromString(Priority(i).String())
			assert(t, Priority(i) == str, i, str)
			testt.Log(t, str, i)
		}
	})
	t.Run("EdgeCases", func(t *testing.T) {
		// too big
		assert(t, Invalid == FromString("<99999999>"))
		assert(t, Invalid == FromString("<9999999999999999999999999999999999>"))
		// not a number
		assert(t, Invalid == FromString("bob"))
		// not a number
		assert(t, Invalid == FromString("<bob>"))
	})

}
