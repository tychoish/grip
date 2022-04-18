package level

import (
	"strings"
	"testing"
)

func assert(t *testing.T, cond bool, args ...any) {
	t.Helper()
	if !cond {
		if len(args) > 0 {
			t.Log(args...)
		}
		t.Fatal("condition was false")
	}
}

func TestLevel(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		t.Run("TooLow", func(t *testing.T) {
			assert(t, !Priority(0).IsValid())
			assert(t, !Priority(-1).IsValid())
		})
		t.Run("TooHigh", func(t *testing.T) {
			assert(t, !Priority(101).IsValid())
			assert(t, !Priority(1000).IsValid())
		})
		t.Run("Edges", func(t *testing.T) {
			assert(t, Priority(1).IsValid())
			assert(t, Priority(100).IsValid())
		})
		t.Run("Range", func(t *testing.T) {
			for i := 1; i < 101; i++ {
				assert(t, Priority(i).IsValid())
			}
		})
	})
	t.Run("StringForm", func(t *testing.T) {
		t.Run("Levels", func(t *testing.T) {
			for _, val := range []Priority{Emergency, Alert, Critical, Error, Warning, Notice, Info, Debug, Trace, Invalid} {
				assert(t, !strings.HasPrefix(val.String(), "<"), val)
				assert(t, !strings.HasSuffix(val.String(), ">"), val)
			}
		})
		t.Run("InBetween", func(t *testing.T) {
			for i := 1; i < 101; i++ {
				if i%10 == 0 {
					continue
				}
				assert(t, strings.HasPrefix(Priority(i).String(), "<"), i)
				assert(t, strings.HasSuffix(Priority(i).String(), ">"), i)
			}
		})

	})
	t.Run("RoundTrip", func(t *testing.T) {
		for i := 1; i < 101; i++ {
			assert(t, Priority(i) == FromString(Priority(i).String()), i)
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
