package send

import (
	"testing"

	"github.com/tychoish/grip/level"
)

func TestSenderWriter(t *testing.T) {
	sink := MakeInternal()
	sink.SetName("sink")
	sink.SetPriority(level.Debug)

	ws := MakeWriter(sink)
	if ws.buffer.Len() != 0 {
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
	if ws.buffer.Len() != 12 {
		t.Error("elements should be equal")
	}

	if err = ws.doSend(); err != nil {
		t.Error(err)
	}

	if !sink.HasMessage() {
		t.Fatal("shuld have message")
	}
	m := sink.GetMessage()
	if !m.Logged {
		t.Error("should be true")
	}
	if m.Message.String() != "hello world" {
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

	if err = ws.doSend(); err != nil {
		t.Error(err)
	}

	if !sink.HasMessage() {
		t.Error("should be true")
	}
	if sink.Len() != 2 {
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
	if m.Message.String() != "hello world" {
		t.Error("elements should be equal")
	}
	if m2.Message.String() != "hello grip" {
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

	if err := ws.Close(); err != nil {
		t.Error(err)
	}
	if !sink.HasMessage() {
		t.Error("should be true")
	}
	if ws.buffer.Len() != 0 {
		t.Fatalf("buffer should be empty [%d]", ws.buffer.Len())
	}

	m = sink.GetMessage()
	if !m.Logged {
		t.Error("should be true")
	}
	if m.Message.String() != "hello world" {
		t.Error("elements should be equal")
	}
	numMessages := sink.Len()
	if ws.buffer.Len() != 0 {
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

	if ws.buffer.Len() != 0 {
		t.Error("elements should be equal")
	}
}
