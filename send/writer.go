package send

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"unicode"

	"github.com/tychoish/grip/message"
)

// WriterSender wraps another sender and also provides an io.Writer.
// (and because Sender is an io.Closer) the type also implements
// io.WriteCloser.
type WriterSender struct {
	Sender
	writer *bufio.Writer
	buffer *bytes.Buffer
	mu     sync.Mutex
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
func MakeWriter(s Sender) *WriterSender {
	buffer := new(bytes.Buffer)

	return &WriterSender{
		Sender: s,
		writer: bufio.NewWriter(buffer),
		buffer: buffer,
	}
}

func (s *WriterSender) Unwrap() Sender { return s.Sender }

// Write captures a sequence of bytes to the send interface. It never errors.
func (s *WriterSender) Write(p []byte) (int, error) {
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

func (s *WriterSender) doSend() error {
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

// Close writes any buffered messages to the underlying Sender. This does
// not close the underlying sender.
func (s *WriterSender) Close() error {
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
