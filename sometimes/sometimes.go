package sometimes

import (
	"math/rand"
	"time"

	"github.com/tychoish/fun/fn"
)

var random fn.Future[int]

func init() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	random = fn.MakeFuture(func() int {
		return r.Intn(101)
	}).Lock()
}

// Fifth returns true 20% of the time.
func Fifth() bool { return random() > 80 }

// Half returns true 50% of the time.
func Half() bool { return random() > 50 }

// Third returns true 33% of the time.
func Third() bool { return random() > 67 }

// Quarter returns true 25% of the time.
func Quarter() bool { return random() > 75 }

// ThreeQuarters returns true 75% of the time.
func ThreeQuarters() bool { return random() > 25 }

// TwoThirds returns true 66% of the time.
func TwoThirds() bool { return random() > 34 }

// Percent takes a number (p) and returns true that percent of the
// time. If p is greater than or equal to 100, Percent always returns
// true. If p is less than or equal to 0, percent always returns false.
func Percent(p int) bool {
	if p >= 100 {
		return true
	}

	if p <= 0 {
		return false
	}

	return random() > (100 - p)
}
