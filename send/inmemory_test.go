package send

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

const maxCap = 10

type InMemorySuite struct {
	maxCap int
	msgs   []message.Composer
	sender *InMemorySender
}

func msgsToString(t *testing.T, sender Sender, msgs []message.Composer) []string {
	t.Helper()

	strs := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		str, err := sender.Formatter()(msg)
		if err != nil {
			t.Fatal(err)
		}
		strs = append(strs, str)
	}

	return strs
}

func msgsToRaw(msgs []message.Composer) []interface{} {
	raw := make([]interface{}, 0, len(msgs))
	for _, msg := range msgs {
		raw = append(raw, msg.Raw())
	}
	return raw
}

func setupFixture(t *testing.T) *InMemorySuite {
	// t.Helper()

	s := &InMemorySuite{}

	info := LevelInfo{Default: level.Debug, Threshold: level.Debug}
	sender, err := NewInMemorySender("inmemory", info, maxCap)
	if err != nil {
		t.Fatal(err)
	}

	if sender == nil {
		t.Fatal("sender must not be nil")
	}

	s.sender = sender.(*InMemorySender)

	if len(s.sender.buffer) != 0 {
		t.Fatal("buffer should be empty")
	}

	if readHeadNone != s.sender.readHead {
		t.Fatal("read ahead should not be set")
	}

	if s.sender.readHeadCaughtUp {
		t.Fatal("read should not be caught up")
	}

	if s.sender.writeHead != 0 {
		t.Fatal("write should not have started")
	}

	if s.sender.totalBytesSent != 0 {
		t.Fatal("should not have sent bytes")
	}

	s.msgs = make([]message.Composer, 2*maxCap)
	for i := range s.msgs {
		s.msgs[i] = message.NewString(info.Default, fmt.Sprint(i))
	}

	return s
}

func TestInvalidCapacityErrors(t *testing.T) {
	badCap := -1
	sender, err := NewInMemorySender("inmemory", LevelInfo{Default: level.Debug, Threshold: level.Debug}, badCap)
	if err == nil {
		t.Fatal(err)
	}
	if sender != nil {
		t.Fatal("sender should not be nil")
	}
}

func TestSendIgnoresMessagesWithPrioritiesBelowThreshold(t *testing.T) {
	s := setupFixture(t)

	msg := message.NewString(level.Trace, "foo")
	s.sender.Send(msg)
	if 0 != len(s.sender.buffer) {
		t.Error("values should be equal")
	}
}

func TestGetEmptyBuffer(t *testing.T) {
	s := setupFixture(t)

	if l := len(s.sender.Get()); l != 0 {
		t.Errorf("lenght is %d not %d", l, s.sender.Get())
	}
}

func TestGetCountInvalidCount(t *testing.T) {
	s := setupFixture(t)

	msgs, n, err := s.sender.GetCount(-1)
	if err == nil {
		t.Fatal("error should not be nil", err)
	}
	if n != 0 {
		t.Fatal("n should be zero", n)
	}
	if msgs != nil {
		t.Fatal("messages should be nil", msgs)
	}

	msgs, n, err = s.sender.GetCount(0)
	if err == nil {
		t.Fatal("error should not be nil", err)
	}
	if n != 0 {
		t.Fatal("n should be zero", n)
	}
	if msgs != nil {
		t.Fatal("messages should be nil")
	}
}

func TestGetCountOne(t *testing.T) {
	s := setupFixture(t)

	for i := 0; i < maxCap-1; i++ {
		s.sender.Send(s.msgs[i])
	}
	for i := 0; i < maxCap-1; i++ {
		msgs, n, err := s.sender.GetCount(1)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatal("values should be equal", n, 1)
		}
		if s.msgs[i] != msgs[0] {
			t.Error("values should be equal")
		}
	}

	s.sender.Send(s.msgs[maxCap])

	msgs, n, err := s.sender.GetCount(1)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("values should be equal", n, 1)
	}
	if s.msgs[maxCap] != msgs[0] {
		t.Error("values should be equal")
	}

	msgs, n, err = s.sender.GetCount(1)
	if !errors.Is(err, io.EOF) {
		t.Fatal("error should be EOF", err)
	}
	if n != 0 {
		t.Fatal("values should be equal", n, 0)
	}
	if len(msgs) != 0 {
		t.Fatal("messages should be empty")
	}
}

func TestGetCountMultiple(t *testing.T) {
	s := setupFixture(t)

	for i := 0; i < maxCap; i++ {
		s.sender.Send(s.msgs[i])
	}

	for count := 1; count <= maxCap; count++ {
		s.sender.ResetRead()
		for i := 0; i < maxCap; i += count {
			msgs, n, err := s.sender.GetCount(count)
			if err != nil {
				t.Fatal(err)
			}
			remaining := count
			start := i
			end := start + count
			if end > maxCap {
				end = maxCap
				remaining = end - start
			}
			if remaining != n {
				t.Error("values should be equal")
			}
			bmsg := s.msgs[start:end]
			for idx := range bmsg {
				if bmsg[idx] != msgs[idx] {
					t.Fatal("values at index should be equal:", idx)
				}
			}
		}
		if !s.sender.readHeadCaughtUp {
			t.Error("value should be true")
		}

		_, _, err := s.sender.GetCount(count)
		if err != io.EOF {
			t.Fatal("values should be equal", err, io.EOF)
		}
	}
}

func TestGetCountMultipleWithOverflow(t *testing.T) {
	s := setupFixture(t)

	for _, msg := range s.msgs {
		s.sender.Send(msg)
	}

	for count := 1; count <= maxCap; count++ {
		s.sender.ResetRead()
		for i := 0; i < maxCap; i += count {
			msgs, n, err := s.sender.GetCount(count)
			if err != nil {
				t.Fatal(err)
			}
			remaining := count
			start := len(s.msgs) - maxCap + i
			end := start + count
			if end > len(s.msgs) {
				end = len(s.msgs)
				remaining = end - start
			}
			if remaining != n {
				t.Error("values should be equal")
			}
			bmsg := s.msgs[start:end]
			for idx := range bmsg {
				if bmsg[idx] != msgs[idx] {
					t.Fatal("values at index should be equal:", idx)
				}
			}
		}
		if !s.sender.readHeadCaughtUp {
			t.Error("value should be true")
		}

		_, _, err := s.sender.GetCount(count)
		if err != io.EOF {
			t.Fatal("values should be equal", err, io.EOF)
		}
	}
}

func TestGetCountTruncated(t *testing.T) {
	s := setupFixture(t)

	s.sender.Send(s.msgs[0])
	s.sender.Send(s.msgs[1])

	msgs, n, err := s.sender.GetCount(1)
	if err != nil {
		t.Fatal(err)
	}
	if 1 != n {
		t.Error("values should be equal")
	}
	if s.msgs[0] != msgs[0] {
		t.Error("values should be equal")
	}
	if s.sender.readHeadCaughtUp {
		t.Fatal("should not be caught up")
	}

	for i := 0; i < maxCap; i++ {
		if readHeadTruncated == s.sender.readHead {
			t.Fatal("should not be equal")
		}
		s.sender.Send(s.msgs[i])
	}
	if s.sender.readHead != readHeadTruncated {
		t.Fatal("values should be equal", s.sender.readHead, readHeadTruncated)
	}
	_, _, err = s.sender.GetCount(1)
	if err != ErrorTruncated {
		t.Fatal("values should be equal", err, ErrorTruncated)
	}
}

func TestGetCountWithCatchupTruncated(t *testing.T) {
	s := setupFixture(t)

	s.sender.Send(s.msgs[0])
	msgs, n, err := s.sender.GetCount(1)
	if err != nil {
		t.Fatal(err)
	}
	if 1 != n {
		t.Error("values should be equal")
	}
	if s.msgs[0] != msgs[0] {
		t.Error("values should be equal")
	}
	if !s.sender.readHeadCaughtUp {
		t.Error("value should be true")
	}

	for i := 0; i < maxCap; i++ {
		if readHeadTruncated == s.sender.readHead {
			t.Fatal("should not be equal")

		}
		s.sender.Send(s.msgs[i])
		if s.sender.readHeadCaughtUp {
			t.Fatal("should not be caught up")
		}
	}
	if s.sender.readHeadCaughtUp {
		t.Fatal("should not be caught up")
	}
	if readHeadTruncated == s.sender.readHead {
		t.Fatal("should not be equal")
	}

	s.sender.Send(s.msgs[0])
	if s.sender.readHeadCaughtUp {
		t.Fatal("should not be caught up")
	}

	if s.sender.readHead != readHeadTruncated {
		t.Fatal("values should be equal", s.sender.readHead, readHeadTruncated)
	}

	_, _, err = s.sender.GetCount(1)
	if ErrorTruncated != err {
		t.Error("values should be equal")
	}
}

func TestGetCountWithCatchupWithOverflowTruncated(t *testing.T) {
	s := setupFixture(t)

	for i := 0; i < maxCap; i++ {
		s.sender.Send(s.msgs[i])
	}
	for i := 0; i < maxCap; i++ {
		msgs, n, err := s.sender.GetCount(1)
		if err != nil {
			t.Fatal(err)
		}
		if 1 != n {
			t.Error("values should be equal")
		}
		if s.msgs[i] != msgs[0] {
			t.Error("values should be equal")
		}
	}
	if !s.sender.readHeadCaughtUp {
		t.Error("value should be true")
	}

	for i := 0; i < maxCap+1; i++ {
		if readHeadTruncated == s.sender.readHead {
			t.Fatal("should not be equal")
		}
		s.sender.Send(s.msgs[i])
		if s.sender.readHeadCaughtUp {
			t.Fatal("should not be caught up")
		}
	}
	if s.sender.readHead != readHeadTruncated {
		t.Fatal("values should be equal", s.sender.readHead, readHeadTruncated)
	}

	_, _, err := s.sender.GetCount(1)
	if ErrorTruncated != err {
		t.Error("values should be equal")
	}
}

func TestGetCountWithOverflowTruncated(t *testing.T) {
	s := setupFixture(t)

	for i := 0; i < maxCap; i++ {
		s.sender.Send(s.msgs[i])
	}
	for i := 0; i < maxCap; i++ {
		msgs, n, err := s.sender.GetCount(1)
		if err != nil {
			t.Fatal(err)
		}
		if 1 != n {
			t.Error("values should be equal")
		}
		if s.msgs[i] != msgs[0] {
			t.Error("values should be equal")
		}
	}
	if !s.sender.readHeadCaughtUp {
		t.Error("value should be true")
	}

	for i := 0; i < maxCap+1; i++ {
		if readHeadTruncated == s.sender.readHead {
			t.Fatal("should not be equal")
		}

		s.sender.Send(s.msgs[i])
		if s.sender.readHeadCaughtUp {
			t.Fatal("should be false")
		}
	}
	if s.sender.readHead != readHeadTruncated {
		t.Fatal("values should be equal", s.sender.readHead, readHeadTruncated)
	}

	_, _, err := s.sender.GetCount(1)
	if ErrorTruncated != err {
		t.Error("values should be equal")
	}
}

func TestGetCountWithWritesAfterEOF(t *testing.T) {
	s := setupFixture(t)

	s.sender.Send(s.msgs[0])
	msgs, n, err := s.sender.GetCount(1)
	if err != nil {
		t.Fatal(err)
	}
	if 1 != n {
		t.Error("values should be equal")
	}
	if s.msgs[0] != msgs[0] {
		t.Error("values should be equal")
	}
	if !s.sender.readHeadCaughtUp {
		t.Error("value should be true")
	}
	_, _, err = s.sender.GetCount(1)
	if io.EOF != err {
		t.Error("values should be equal")
	}

	s.sender.Send(s.msgs[1])
	if s.sender.readHeadCaughtUp {
		t.Fatal("read pointer should not be caught up")
	}
	msgs, n, err = s.sender.GetCount(1)
	if err != nil {
		t.Fatal(err)
	}
	if 1 != n {
		t.Error("values should be equal")
	}
	if s.msgs[1] != msgs[0] {
		t.Error("values should be equal")
	}
	if !s.sender.readHeadCaughtUp {
		t.Error("value should be true")
	}
	_, _, err = s.sender.GetCount(1)
	if io.EOF != err {
		t.Error("values should be equal")
	}
}

func TestResetRead(t *testing.T) {
	s := setupFixture(t)

	for i := 0; i < maxCap-1; i++ {
		s.sender.Send(s.msgs[i])
	}

	var err error
	var n int
	var msgs []message.Composer
	for i := 0; i < maxCap-1; i++ {
		msgs, n, err = s.sender.GetCount(1)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatal("values should be equal", n, 1)
		}
		if s.msgs[i] != msgs[0] {
			t.Error("values should be equal")
		}
	}
	if !s.sender.readHeadCaughtUp {
		t.Error("value should be true")
	}

	_, _, err = s.sender.GetCount(1)
	if io.EOF != err {
		t.Error("values should be equal")
	}

	s.sender.ResetRead()
	if readHeadNone != s.sender.readHead {
		t.Error("values should be equal")
	}
	if s.sender.readHeadCaughtUp {
		t.Error("read should not be caught up yet")
	}

	for i := 0; i < maxCap-1; i++ {
		msgs, n, err = s.sender.GetCount(1)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatal("values should be equal", n, 1)
		}
		if s.msgs[i] != msgs[0] {
			t.Error("values should be equal")
		}
	}

	_, _, err = s.sender.GetCount(1)
	if !errors.Is(err, io.EOF) {
		t.Fatal("should be EOF", err)
	}
}

func TestGetCountEmptyBuffer(t *testing.T) {
	s := setupFixture(t)

	msgs, n, err := s.sender.GetCount(1)
	if err != io.EOF {
		t.Fatal("values should be equal", err, io.EOF)
	}
	if n != 0 {
		t.Fatal("count should be zero")
	}
	if len(msgs) != 0 {
		t.Fatal("messages should be empty")
	}
}

func TestGetWithOverflow(t *testing.T) {
	s := setupFixture(t)

	for i, msg := range s.msgs {
		s.sender.Send(msg)
		found := s.sender.Get()

		if i < maxCap {
			for j := 0; j < i+1; j++ {
				if s.msgs[j] != found[j] {
					t.Error("values should be equal")
				}
			}
		} else {
			for j := 0; j < maxCap; j++ {
				if s.msgs[i+1-maxCap+j] != found[j] {
					t.Error("values should be equal")
				}
			}
		}
	}
}

func TestGetStringEmptyBuffer(t *testing.T) {
	s := setupFixture(t)

	str, err := s.sender.GetString()
	if err := err; err != nil {
		t.Fatal(err)
	}
	if len(str) != 0 {
		t.Fatal("string form should be empty")
	}
}

func TestGetStringWithOverflow(t *testing.T) {
	s := setupFixture(t)

	for i, msg := range s.msgs {
		s.sender.Send(msg)
		found, err := s.sender.GetString()
		if err != nil {
			t.Fatal(err)
		}

		var expected []string
		if i+1 < maxCap {
			if len(found) != i+1 {
				t.Fatal("values should be equal", len(found), i+1)
			}
			expected = msgsToString(t, s.sender, s.msgs[:i+1])
		} else {
			if len(found) != maxCap {
				t.Fatal("values should be equal", len(found), maxCap)
			}
			expected = msgsToString(t, s.sender, s.msgs[i+1-maxCap:i+1])
		}
		if len(found) != len(expected) {
			t.Fatal("values should be equal", len(found), len(expected))
		}

		for j := 0; j < len(found); j++ {
			if expected[j] != found[j] {
				t.Error("values should be equal")
			}
		}
	}
}

func TestGetRawEmptyBuffer(t *testing.T) {
	s := setupFixture(t)

	if len(s.sender.GetRaw()) != 0 {
		t.Fatal("raw messages should be empty")
	}
}

func TestGetRawWithOverflow(t *testing.T) {
	s := setupFixture(t)

	for i, msg := range s.msgs {
		s.sender.Send(msg)
		found := s.sender.GetRaw()
		var expected []interface{}

		if i+1 < maxCap {
			if len(found) != i+1 {
				t.Fatal("values should be equal", len(found), i+1)
			}
			expected = msgsToRaw(s.msgs[:i+1])
		} else {
			if len(found) != maxCap {
				t.Fatal("values should be equal", len(found), maxCap)
			}
			expected = msgsToRaw(s.msgs[i+1-maxCap : i+1])
		}

		if len(expected) != len(found) {
			t.Error("values should be equal")
		}
		for j := 0; j < len(found); j++ {
			if expected[j] != found[j] {
				t.Error("values should be equal")
			}
		}
	}
}

func TestTotalBytes(t *testing.T) {
	s := setupFixture(t)

	var totalBytes int64
	for _, msg := range s.msgs {
		s.sender.Send(msg)
		totalBytes += int64(len(msg.String()))
		if totalBytes != s.sender.TotalBytesSent() {
			t.Error("values should be equal")
		}
	}
}
