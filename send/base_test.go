package send

import (
	"testing"

	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/grip/message"
)

func TestBaseCustomFormatter(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		b := &Base{}
		m := b.Converter()("hello")
		check.Equal(t, "hello", m.String())
	})
	t.Run("Injectable", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			b := &Base{}
			b.SetConverter(func(in any) (message.Composer, bool) {
				m := message.MakeFields(message.Fields{
					"input": in,
				})
				m.SetOption(message.OptionSkipAllMetadata)
				return m, true
			})
			m := b.Converter()("hello")
			check.Equal(t, "input='hello'", m.String())
		})
		t.Run("Passthrough", func(t *testing.T) {
			b := &Base{}
			b.SetConverter(func(in any) (message.Composer, bool) {
				m := message.MakeFields(message.Fields{
					"input": in,
				})
				m.SetOption(message.OptionSkipAllMetadata)
				return m, false
			})
			m := b.Converter()("hello words")
			check.Equal(t, "hello words", m.String())
		})
	})
}
