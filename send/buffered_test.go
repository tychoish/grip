package send

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestBufferedSend(t *testing.T) {
	s, err := NewInternalLogger("buffs", LevelInfo{level.Debug, level.Debug})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("RespectsPriority", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		defer bs.cancel()

		bs.Send(message.ConvertWithPriority(level.Trace, fmt.Sprintf("should not send")))
		assert.Empty(t, bs.buffer)
		_, ok := s.GetMessageSafe()
		if ok {
			t.Error("should be false")
		}
	})
	t.Run("FlushesAtCapactiy", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		defer bs.cancel()

		for i := 0; i < 12; i++ {
			require.Len(t, bs.buffer, i%10)
			bs.Send(message.ConvertWithPriority(level.Debug, fmt.Sprintf("message %d", i+1)))
		}
		assert.Len(t, bs.buffer, 2)
		msg, ok := s.GetMessageSafe()
		require.True(t, ok)
		msgs := strings.Split(msg.Message.String(), "\n")
		assert.Len(t, msgs, 10)
		for i, msg := range msgs {
			require.Equal(t, fmt.Sprintf("message %d", i+1), msg)
		}
	})
	t.Run("FlushesOnInterval", func(t *testing.T) {
		bs := newBufferedSender(s, 5*time.Second, 10)
		defer bs.cancel()

		bs.Send(message.ConvertWithPriority(level.Debug, "should flush"))
		time.Sleep(6 * time.Second)
		bs.mu.Lock()
		if time.Since(bs.lastFlush) > 2*time.Second {
			t.Error("should be true")
		}
		bs.mu.Unlock()
		msg, ok := s.GetMessageSafe()
		require.True(t, ok)
		if msg.Message.String() != "should flush" {
			t.Error("elements should be equal")
		}
	})
	t.Run("ClosedSender", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		bs.closed = true
		defer bs.cancel()

		bs.Send(message.ConvertWithPriority(level.Debug, "should not send"))
		assert.Empty(t, bs.buffer)
		_, ok := s.GetMessageSafe()
		if ok {
			t.Error("should be false")
		}
	})
}

func TestFlush(t *testing.T) {
	s, err := NewInternalLogger("buffs", LevelInfo{level.Debug, level.Debug})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ForceFlush", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		defer bs.cancel()

		bs.Send(message.ConvertWithPriority(level.Debug, "message"))
		assert.Len(t, bs.buffer, 1)
		if err := bs.Flush(context.TODO()); err != nil {
			t.Fatal(err)
		}
		bs.mu.Lock()
		if time.Since(bs.lastFlush) > time.Second {
			t.Error("should be true")
		}
		bs.mu.Unlock()
		assert.Empty(t, bs.buffer)
		msg, ok := s.GetMessageSafe()
		require.True(t, ok)
		if msg.Message.String() != "message" {
			t.Error("elements should be equal")
		}
	})
	t.Run("ClosedSender", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		bs.buffer = append(bs.buffer, message.ConvertWithPriority(level.Debug, "message"))
		bs.cancel()
		bs.closed = true

		if err := bs.Flush(context.TODO()); err != nil {
			t.Error(err)
		}
		assert.Len(t, bs.buffer, 1)
		_, ok := s.GetMessageSafe()
		if ok {
			t.Error("should be false")
		}
	})
}

func TestBufferedClose(t *testing.T) {
	s, err := NewInternalLogger("buffs", LevelInfo{level.Debug, level.Debug})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("EmptyBuffer", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)

		assert.Nil(t, bs.Close())
		if !bs.closed {
			t.Error("should be true")
		}
		_, ok := s.GetMessageSafe()
		if ok {
			t.Error("should be false")
		}
	})
	t.Run("NonEmptyBuffer", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		bs.buffer = append(
			bs.buffer,
			message.ConvertWithPriority(level.Debug, "message1"),
			message.ConvertWithPriority(level.Debug, "message2"),
			message.ConvertWithPriority(level.Debug, "message3"),
		)

		assert.Nil(t, bs.Close())
		if !bs.closed {
			t.Error("should be true")
		}
		assert.Empty(t, bs.buffer)
		msgs, ok := s.GetMessageSafe()
		require.True(t, ok)
		if msgs.Message.String() != "message1\nmessage2\nmessage3" {
			t.Error("elements should be equal")
		}
	})
	t.Run("NoopWhenClosed", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)

		if err := bs.Close(); err != nil {
			t.Error(err)
		}
		if !bs.closed {
			t.Error("should be true")
		}
		if err := bs.Close(); err != nil {
			t.Error(err)
		}
	})
}

func TestIntervalFlush(t *testing.T) {
	s, err := NewInternalLogger("buffs", LevelInfo{level.Debug, level.Debug})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ReturnsWhenClosed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		bs := &bufferedSender{
			Sender: s,
			buffer: []message.Composer{},
			cancel: cancel,
		}
		canceled := make(chan bool)

		go func() {
			bs.intervalFlush(ctx, time.Minute)
			canceled <- true
		}()
		if err := bs.Close(); err != nil {
			t.Error(err)
		}
		if !<-canceled {
			t.Error("should be true")
		}
	})
}

func newBufferedSender(sender Sender, interval time.Duration, size int) *bufferedSender {
	bs := NewBuffered(sender, interval, size)
	return bs.(*bufferedSender)
}
