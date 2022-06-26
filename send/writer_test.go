package send

import (
	"testing"

	"github.com/tychoish/grip/level"
)

func TestSenderWriter(t *testing.T) {
	sink, err := NewInternalLogger("sink", LevelInfo{level.Debug, level.Debug})
	if err != nil {
		t.Error(err)
	}

	ws := MakeWriter(sink)
	if 0 != ws.buffer.Len() {
		t.Error("elements should be equal")
	}

	// writing something without a new line character will cause it to not send.
	msg := []byte("hello world")
	n, err := ws.Write(msg)
	if err != nil {
		t.Error(err)
	}
	if len(msg) != n {
		t.Error("elements should be equal")
	}
	if n != ws.buffer.Len() {
		t.Error("elements should be equal")
	}
	if sink.HasMessage() {
		t.Error("should be false")
	}

	newLine := []byte{'\n'}
	// if we add a new line character, then it'll flush
	n, err = ws.Write(newLine)
	if err != nil {
		t.Error(err)
	}
	if len(newLine) != n {
		t.Error("elements should be equal")
	}
	if 12 != ws.buffer.Len() {
		t.Error("elements should be equal")
	}

	if err := ws.doSend(); err != nil {
		t.Error(err)
	}

	if !sink.HasMessage() {
		t.Fatal("shuld have message")
	}
	m := sink.GetMessage()
	if !m.Logged {
		t.Error("should be true")
	}
	if "hello world" != m.Message.String() {
		t.Error("elements should be equal")
	}
	// the above trimmed the final new line off, which is correct,
	// given how senders will actually newline deelimit messages anyway.
	//
	// at the same time, we should make sure that we preserve newlines internally
	msg = []byte("hello world\nhello grip\n")
	n, err = ws.Write(msg)
	if err != nil {
		t.Error(err)
	}
	if len(msg) != n {
		t.Error("elements should be equal")
	}
	if len(msg) != ws.buffer.Len() {
		t.Error("elements should be equal")
	}

	if err := ws.doSend(); err != nil {
		t.Error(err)
	}

	if !sink.HasMessage() {
		t.Error("should be true")
	}
	if 2 != sink.Len() {
		t.Error("elements should be equal")
	}
	m = sink.GetMessage()
	m2 := sink.GetMessage()
	if !m.Logged {
		t.Error("should be true")
	}
	if !m2.Logged {
		t.Error("should be true")
	}
	if "hello world" != m.Message.String() {
		t.Error("elements should be equal")
	}
	if "hello grip" != m2.Message.String() {
		t.Error("elements should be equal")
	}

	// send a message, but no new line, means it lives in the buffer.
	msg = []byte("hello world")
	n, err = ws.Write(msg)
	if err != nil {
		t.Error(err)
	}
	if len(msg) != n {
		t.Error("elements should be equal")
	}
	if n != ws.buffer.Len() {
		t.Error("elements should be equal")
	}
	if sink.HasMessage() {
		t.Error("should be false")
	}

	if ws.buffer.Len() != 0 {
		t.Fatal("buffer should be empty")
	}
	if err := ws.Close(); err != nil {
		t.Error(err)
	}
	if !sink.HasMessage() {
		t.Error("should be true")
	}
	m = sink.GetMessage()
	if !m.Logged {
		t.Error("should be true")
	}
	if "hello world" != m.Message.String() {
		t.Error("elements should be equal")
	}
	numMessages := sink.Len()
	if 0 != ws.buffer.Len() {
		t.Error("elements should be equal")
	}
	if sink.Len() != numMessages {
		t.Error("elements should be equal")
	}

	for i := 0; i < 10; i++ {
		if err := ws.Close(); err != nil {
			t.Error(err)
		}
		if sink.GetMessage().Logged {
			t.Error("should be false")
		}
	}

	if 0 != ws.buffer.Len() {
		t.Error("elements should be equal")
	}
}
