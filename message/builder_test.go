package message

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/testt"
)

func mockSender(t *testing.T, expected int) func(Composer) {
	t.Helper()
	count := &atomic.Int64{}
	t.Cleanup(func() {
		t.Helper()
		check.Equal(t, expected, int(count.Load()))
	})
	return func(Composer) { count.Add(1) }
}

func mockSenderMessage(t *testing.T, expected string) func(Composer) {
	t.Helper()
	count := &atomic.Int64{}
	value := &adt.Atomic[string]{}
	t.Cleanup(func() {
		t.Helper()
		check.Equal(t, int(count.Load()), 1)
		check.Equal(t, expected, value.Get())
	})
	return func(c Composer) {
		t.Helper()
		count.Add(1)
		value.Set(c.String())
		testt.Logf(t, "%d> %T", count.Load(), c)
	}
}

func TestBuilder(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		b := NewBuilder(nil)
		b.Send()
		check.Error(t, b.catcher.Resolve())
	})
	t.Run("ErrorsBecomeMessages", func(t *testing.T) {
		b := NewBuilder(mockSenderMessage(t, "kip"))
		b.catcher.Add(errors.New("kip"))
		b.Send()
	})
	t.Run("ErrorsAreAnnotated", func(t *testing.T) {
		b := NewBuilder(mockSenderMessage(t, "bad cat: kip")).String("bad cat").SetGroup(true)
		b.catcher.Add(errors.New("kip"))
		b.Send()
	})
	t.Run("SetLevelInvalidIsAnError", func(t *testing.T) {
		NewBuilder(mockSender(t, 1)).String("msg").Level(0).Send()
		NewBuilder(mockSender(t, 1)).String("msg").Level(200).Send()
		NewBuilder(mockSender(t, 1)).Level(0).Send()
		NewBuilder(mockSender(t, 1)).Level(200).Send()
	})
	t.Run("SingleString", func(t *testing.T) {
		NewBuilder(mockSender(t, 1)).String("hello world").Send()
	})
	t.Run("Double", func(t *testing.T) {
		NewBuilder(mockSender(t, 2)).String("hello").String("world").Send()
	})
	t.Run("DoubleGroup", func(t *testing.T) {
		NewBuilder(mockSender(t, 1)).String("hello").String("world").Group().Send()
	})
	t.Run("DoubleGroupCallsAreSequential", func(t *testing.T) {
		NewBuilder(mockSender(t, 2)).String("hello").String("world").Group().Ungroup().Send()
		NewBuilder(mockSender(t, 2)).String("hello").String("world").Group().Group().Ungroup().Send()
		NewBuilder(mockSender(t, 1)).String("hello").String("world").Ungroup().Group().Send()
		NewBuilder(mockSender(t, 1)).String("hello").String("world").Ungroup().Group().Group().Send()
	})
	t.Run("SetGroup", func(t *testing.T) {
		NewBuilder(mockSender(t, 2)).String("hello").String("world").Group().SetGroup(false).Send()
		NewBuilder(mockSender(t, 1)).String("hello").String("world").Ungroup().SetGroup(true).Send()
	})

	t.Run("Values", func(t *testing.T) {
		t.Run("SingleStringValue", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello world")).String("hello world").Send()
		})
		t.Run("SingleFormat", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello 543 world")).F("hello %d world", 543).Send()
		})
		t.Run("SingleLines", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello world 543")).Ln("hello", "world", 543).Send()
		})
		t.Run("SingleError", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "kip: EOF")).Error(fmt.Errorf("kip: %w", io.EOF)).Send()
		})
		t.Run("SingleStringSlice", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello world 543")).Strings([]string{"hello", "world", "543"}).Send()
		})
		t.Run("FromMap", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello='world'")).StringMap(map[string]string{"hello": "world"}).Send()
		})
	})
	t.Run("Conditional", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hi kip")).String("hi kip").When(true).Send()
		})
		t.Run("False", func(t *testing.T) {
			NewBuilder(mockSender(t, 1)).String("hello").When(false).Group().Send()
		})

	})

}
