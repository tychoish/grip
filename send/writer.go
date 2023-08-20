package send

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"unicode"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

type writerSenderImpl struct {
	Sender
	adt.Atomic[level.Priority]

	writer *bufio.Writer
	buffer *bytes.Buffer
	mu     sync.Mutex
}

// WriterSender wraps another sender and also provides an io.Writer.
// (and because Sender is an io.Closer) the type also implements
// io.WriteCloser. Set the Level field to control the level that the
// data is logged at. If not specified, the sender will use the
// Sender's configured priority threshold.
//
// If you do not use the `MakeWriter
type WriterSender interface {
	Sender
	io.WriteCloser
	// the Get/Set methods on the WriterSender control the
	// priority of messages sent to the sender.
	adt.AtomicValue[level.Priority]
}

// MakeWriter wraps another sender and also provides an io.Writer.
// (and because Sender is an io.Closer) the type also implements
// io.WriteCloser.
//
// While WriteSender itself implements Sender, it also provides a
// Writer method, which allows you to use this Sender to capture
// file-like write operations.
//
// Data sent via the Write method is buffered internally until its
// passed a byte slice that ends with the new line character. If the
// string form of the bytes passed to the write method (including all
// buffered messages) is only whitespace, then it is not sent.
//
// If there are any bytes in the buffer when the Close method is
// called, this sender flushes the buffer. WriterSender does not own the
// underlying Sender, so users are responsible for closing the underlying Sender
// if/when it is appropriate to release its resources.
func MakeWriter(s Sender) *writerSenderImpl {
	buffer := new(bytes.Buffer)

	return &writerSenderImpl{
		Sender: s,

		writer: bufio.NewWriter(buffer),
		buffer: buffer,
	}
}

func (s *writerSenderImpl) Unwrap() Sender { return s.Sender }

// Write captures a sequence of bytes to the send interface. It never errors.
func (s *writerSenderImpl) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.writer.Write(p)
	if err != nil {
		return n, err
	}
	_ = s.writer.Flush()

	if s.buffer.Len() > 80 {
		err = s.doSend()
	}

	return n, err
}

func (s *writerSenderImpl) doSend() error {
	pri := s.Sender.Priority()
	for {
		line, err := s.buffer.ReadBytes('\n')
		if err == io.EOF {
			s.buffer.Write(line)
			return nil
		}

		lncp := make([]byte, len(line))
		copy(lncp, line)

		if err == nil {
			m := message.MakeBytes(bytes.TrimRightFunc(lncp, unicode.IsSpace))
			m.SetPriority(pri)
			s.Send(m)
			continue
		}

		m := message.MakeBytes(bytes.TrimRightFunc(lncp, unicode.IsSpace))
		m.SetPriority(pri)
		s.Send(m)
		return err
	}
}

// Close writbes any buffered messages to the underlying Sender. This does
// not close the underlying sender.
func (s *writerSenderImpl) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.writer.Flush(); err != nil {
		return err
	}

	m := message.MakeBytes(bytes.TrimRightFunc(s.buffer.Bytes(), unicode.IsSpace))
	m.SetPriority(s.Priority())
	s.Send(m)
	s.buffer.Reset()
	s.writer.Reset(s.buffer)
	return nil
}
