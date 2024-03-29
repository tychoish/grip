package send

import (
	"context"
	"sync"
	"time"

	"github.com/tychoish/grip/message"
)

const minInterval = 5 * time.Second

type bufferedSender struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	buffer    []message.Composer
	size      int
	lastFlush time.Time
	closed    bool

	Sender
}

// MakeBuffered provides a Sender implementation that wraps an existing
// Sender sending messages in batches, on a specified buffer size or after an
// interval has passed.
//
// If the interval is 0, the constructor sets an interval of 1 minute, and if
// it is less than 5 seconds, the constructor sets it to 5 seconds. If the
// size threshold is 0, then the constructor sets a threshold of 100.
//
// This Sender does not own the underlying Sender, so users are responsible for
// closing the underlying Sender if/when it is appropriate to release its
// resources.
func MakeBuffered(sender Sender, interval time.Duration, size int) Sender {
	if interval == 0 {
		interval = time.Minute
	} else if interval < minInterval {
		interval = minInterval
	}

	if size <= 0 {
		size = 100
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &bufferedSender{
		Sender: sender,
		cancel: cancel,
		buffer: []message.Composer{},
		size:   size,
	}

	go s.intervalFlush(ctx, interval)

	return s
}

func (s *bufferedSender) Unwrap() Sender { return s.Sender }

func (s *bufferedSender) Send(msg message.Composer) {
	if !ShouldLog(s, msg) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.buffer = append(s.buffer, msg)
	if len(s.buffer) >= s.size {
		s.flush()
	}
}

func (s *bufferedSender) Flush(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.flush()
	}

	return nil
}

// Close writes any buffered messages to the underlying Sender. This does not
// close the underlying sender.
func (s *bufferedSender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.cancel()
	if len(s.buffer) > 0 {
		s.flush()
	}
	s.closed = true

	return nil
}

func (s *bufferedSender) intervalFlush(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			func() {
				s.mu.Lock()
				defer s.mu.Unlock()
				if len(s.buffer) > 0 && time.Since(s.lastFlush) >= interval {
					s.flush()
				}
			}()
			_ = timer.Reset(interval)
		}
	}
}

func (s *bufferedSender) flush() {
	if len(s.buffer) == 1 {
		s.Sender.Send(s.buffer[0])
	} else {
		s.Sender.Send(message.MakeGroupComposer(s.buffer))
	}

	s.buffer = []message.Composer{}
	s.lastFlush = time.Now()
}
