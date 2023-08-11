package send

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func convertWithPriority(p level.Priority, m any) message.Composer {
	out := message.Convert(m)
	out.SetPriority(p)
	return out
}

func TestBufferedSend(t *testing.T) {
	t.Parallel()

	s := MakeInternal()
	s.SetName("buffs")
	s.SetPriority(level.Debug)

	t.Run("RespectsPriority", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		defer bs.cancel()

		bs.Send(convertWithPriority(level.Trace, "should not send"))
		if len(bs.buffer) != 0 {
			t.Fatal("buffer should be empty")
		}

		_, ok := s.GetMessageSafe()
		if ok {
			t.Error("should be false")
		}
	})
	t.Run("FlushesAtCapactiy", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		defer bs.cancel()

		for i := 0; i < 12; i++ {
			if len(bs.buffer) != i%10 {
				t.Fatalf("buffer should have messages: %d", len(bs.buffer))
			}
			bs.Send(convertWithPriority(level.Debug, fmt.Sprintf("message %d", i+1)))
		}
		if l := len(bs.buffer); l != 2 {
			t.Errorf("length should be %d but was %d", 2, l)
		}
		msg, ok := s.GetMessageSafe()
		if !ok {
			t.Fatal("value should be true")
		}
		msgs := strings.Split(msg.Message.String(), "\n")
		if l := len(msgs); l != 10 {
			t.Errorf("length should be %d but was %d", 10, l)
		}
		for i, msg := range msgs {
			if fmt.Sprintf("message %d", i+1) != msg {
				t.Fatal("message should be well formed")
			}
		}
	})
	t.Run("FlushesOnInterval", func(t *testing.T) {
		bs := newBufferedSender(s, 5*time.Second, 10)
		defer bs.cancel()

		bs.Send(convertWithPriority(level.Debug, "should flush"))
		time.Sleep(6 * time.Second)
		bs.mu.Lock()
		if time.Since(bs.lastFlush) > 2*time.Second {
			t.Error("should be true")
		}
		bs.mu.Unlock()
		msg, ok := s.GetMessageSafe()
		if !ok {
			t.Fatal("value should be true")
		}
		if msg.Message.String() != "should flush" {
			t.Error("elements should be equal")
		}
	})
	t.Run("ClosedSender", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		bs.closed = true
		defer bs.cancel()

		bs.Send(convertWithPriority(level.Debug, "should not send"))
		if len(bs.buffer) != 0 {
			t.Fatal("buffer should be empty")
		}
		_, ok := s.GetMessageSafe()
		if ok {
			t.Error("should be false")
		}
	})
}

func TestFlush(t *testing.T) {
	t.Parallel()

	s := MakeInternal()
	s.SetName("buffs")
	s.SetPriority(level.Debug)

	t.Run("ForceFlush", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		defer bs.cancel()

		bs.Send(convertWithPriority(level.Debug, "message"))
		if l := len(bs.buffer); l != 1 {
			t.Errorf("length should be %d but was %d", 1, l)
		}
		if err := bs.Flush(context.TODO()); err != nil {
			t.Fatal(err)
		}
		bs.mu.Lock()
		if time.Since(bs.lastFlush) > time.Second {
			t.Error("should be true")
		}
		bs.mu.Unlock()
		if len(bs.buffer) != 0 {
			t.Fatal("buffer should be empty")
		}
		msg, ok := s.GetMessageSafe()
		if !ok {
			t.Fatal("value should be true")
		}
		if msg.Message.String() != "message" {
			t.Error("elements should be equal")
		}
	})
	t.Run("ClosedSender", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)
		bs.buffer = append(bs.buffer, convertWithPriority(level.Debug, "message"))
		bs.cancel()
		bs.closed = true

		if err := bs.Flush(context.TODO()); err != nil {
			t.Error(err)
		}
		if l := len(bs.buffer); l != 1 {
			t.Errorf("length should be %d but was %d", 1, l)
		}
		_, ok := s.GetMessageSafe()
		if ok {
			t.Error("should be false")
		}
	})
}

func TestBufferedClose(t *testing.T) {
	t.Parallel()

	s := MakeInternal()
	s.SetName("buffs")
	s.SetPriority(level.Debug)

	t.Run("EmptyBuffer", func(t *testing.T) {
		bs := newBufferedSender(s, time.Minute, 10)

		if err := bs.Close(); err != nil {
			t.Fatal("should not error to close")
		}
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
			convertWithPriority(level.Debug, "message1"),
			convertWithPriority(level.Debug, "message2"),
			convertWithPriority(level.Debug, "message3"),
		)

		if err := bs.Close(); err != nil {
			t.Fatal("should not error to close")
		}

		if !bs.closed {
			t.Error("should be true")
		}
		if len(bs.buffer) != 0 {
			t.Fatal("buffer should be empty")
		}

		msgs, ok := s.GetMessageSafe()
		if !ok {
			t.Fatal("value should be true")
		}
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
	t.Parallel()

	s := MakeInternal()
	s.SetName("buffs")
	s.SetPriority(level.Debug)

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
	bs := MakeBuffered(sender, interval, size)
	return bs.(*bufferedSender)
}
