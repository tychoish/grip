package send

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func senderFixture(t *testing.T) (senders map[string]Sender) {
	t.Helper()
	tempDir := t.TempDir()
	if err := os.MkdirAll(tempDir, 0766); err != nil {
		t.Fatal(err)
	}

	l := LevelInfo{level.Info, level.Notice}
	senders = map[string]Sender{
		// "slack": &slackJournal{Base: NewBase("slack")},
		// "xmpp":  &xmppLogger{Base: NewBase("xmpp")},
	}

	internal := MakeInternalLogger()
	internal.name = "internal"
	internal.output = make(chan *InternalMessage)
	senders["internal"] = internal

	native, err := NewStdOutput("native", l)
	if err != nil {
		t.Fatal(err)
	}
	senders["native"] = native

	senders["writer"] = MakeWriter(native)

	var plain, plainerr, plainfile Sender
	plain, err = NewPlainStdOutput("plain", l)
	if err != nil {
		t.Fatal(err)
	}
	senders["plain"] = plain

	plainerr, err = NewPlainStdError("plain.err", l)
	if err != nil {
		t.Fatal(err)
	}
	senders["plain.err"] = plainerr

	plainfile, err = NewPlainFile("plain.file", filepath.Join(tempDir, "plain.file"), l)
	if err != nil {
		t.Fatal(err)
	}
	senders["plain.file"] = plainfile

	var asyncOne, asyncTwo Sender
	asyncOne, err = NewStdOutput("async-one", l)
	if err != nil {
		t.Fatal(err)
	}
	asyncTwo, err = NewStdOutput("async-two", l)
	if err != nil {
		t.Fatal(err)
	}
	senders["async"] = NewAsyncGroup(context.Background(), 16, asyncOne, asyncTwo)

	nativeErr, err := NewStdError("error", l)
	if err != nil {
		t.Fatal(err)
	}
	senders["error"] = nativeErr

	nativeFile, err := NewFile("native-file", filepath.Join(tempDir, "file"), l)
	if err != nil {
		t.Fatal(err)
	}
	senders["native-file"] = nativeFile

	callsite, err := NewCallSit("callsite", 1, l)
	if err != nil {
		t.Fatal(err)
	}
	senders["callsite"] = callsite

	callsiteFile, err := NewCallSiteFile("callsite", filepath.Join(tempDir, "cs"), 1, l)
	if err != nil {
		t.Fatal(err)
	}
	senders["callsite-file"] = callsiteFile

	jsons, err := NewJSON("json", LevelInfo{level.Info, level.Notice})
	if err != nil {
		t.Fatal(err)
	}
	senders["json"] = jsons

	jsonf, err := NewJSONFile("json", filepath.Join(tempDir, "js"), l)
	if err != nil {
		t.Fatal(err)
	}
	senders["json"] = jsonf

	var sender Sender
	multiSenders := []Sender{}
	for i := 0; i < 4; i++ {
		sender, err = NewStdOutput(fmt.Sprintf("native-%d", i), l)
		if err != nil {
			t.Fatal(err)
		}
		multiSenders = append(multiSenders, sender)
	}

	multi, err := NewMulti("multi", l, multiSenders)
	if err != nil {
		t.Fatal(err)
	}
	senders["multi"] = multi

	bufferedInternal, err := NewStdOutput("buffered", l)
	if err != nil {
		t.Fatal(err)
	}
	senders["buffered"] = NewBuffered(bufferedInternal, minInterval, 1)

	annotatingBase, err := NewStdOutput("async-one", l)
	if err != nil {
		t.Fatal(err)
	}
	senders["annotating"] = MakeAnnotating(annotatingBase, map[string]any{
		"one":    1,
		"true":   true,
		"string": "string",
	})

	for _, size := range []int{1, 100, 10000, 1000000} {
		name := fmt.Sprintf("inmemory-%d", size)
		senders[name], err = NewInMemorySender(name, l, size)
		if err != nil {
			t.Fatal(err)
		}
		senders[name].SetFormatter(MakeDefaultFormatter())
	}
	t.Cleanup(func() {
		if runtime.GOOS == "windows" {
			_ = senders["native-file"].Close()
			_ = senders["callsite-file"].Close()
			_ = senders["json"].Close()
			_ = senders["plain.file"].Close()
		}
		if err := senders["internal"].Close(); err != nil {
			t.Error(err)
		}
	})
	return senders
}

func functionalMockSenders(t *testing.T, in map[string]Sender) map[string]Sender {
	t.Helper()

	out := map[string]Sender{}
	for t, sender := range in {
		if t == "slack" || t == "internal" || t == "xmpp" || t == "buildlogger" {
			continue
		} else if strings.HasPrefix(t, "github") {
			continue

		} else {
			out[t] = sender
		}
	}
	return out
}

func TestSenderImplementsInterface(t *testing.T) {
	// this actually won't catch the error; the compiler will in
	// the fixtures, but either way we need to make sure that the
	// tests actually enforce this.
	for name, sender := range senderFixture(t) {
		if _, ok := sender.(Sender); !ok {
			t.Errorf("sender %q does not implement interface Sender", name)
		}
	}
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*()"

func randomString(n int, r *rand.Rand) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[r.Int63()%int64(len(letters))]
	}
	return string(b)
}

func TestNameSetterRoundTrip(t *testing.T) {
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	for _, sender := range senderFixture(t) {
		for i := 0; i < 100; i++ {
			name := randomString(12, rand)
			if name == sender.Name() {
				t.Error("values should NOT be equal")
			}
			sender.SetName(name)
			if name != sender.Name() {
				t.Error("values should be equal")
			}
		}
	}
}

func TestLevelSetterRejectsInvalidSettings(t *testing.T) {
	levels := []LevelInfo{
		{level.Invalid, level.Invalid},
		{level.Priority(-10), level.Priority(-1)},
		{level.Debug, level.Priority(-1)},
		{level.Priority(800), level.Priority(-2)},
	}

	for n, sender := range senderFixture(t) {
		if n == "async" {
			// the async sender doesn't meaningfully have
			// its own level because it passes this down
			// to its constituent senders.
			continue
		}

		if err := sender.SetLevel(LevelInfo{level.Debug, level.Alert}); err != nil {
			t.Fatal(err)
		}
		for _, l := range levels {
			if !sender.Level().Valid() {
				t.Error("sender should not validate")
			}
			if l.Valid() {
				t.Error("level is validate")
			}
			if err := sender.SetLevel(l); err == nil {
				t.Error("setting invalid level should error")
			}
			if !sender.Level().Valid() {
				t.Error("level should be valid")
			}
			if l == sender.Level() {
				t.Error("values should NOT be equal")
			}
		}

	}
}

func TestCloserShouldUsuallyNoop(t *testing.T) {
	for _, sender := range senderFixture(t) {
		if err := sender.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBasicNoopSendTest(t *testing.T) {
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	for _, sender := range functionalMockSenders(t, senderFixture(t)) {
		for i := -10; i <= 110; i += 5 {
			m := message.NewString(level.Priority(i), "hello world! "+randomString(10, rand))
			sender.Send(m)
		}
	}
}

func TestBaseConstructor(t *testing.T) {
	sink, err := NewInternalLogger("sink", LevelInfo{level.Debug, level.Debug})
	if err != nil {
		t.Error(err)
	}
	handler := ErrorHandlerFromSender(sink)
	if sink.Len() != 0 {
		t.Error("elements should be equal")
	}
	if sink.HasMessage() {
		t.Error("should be false")
	}

	for _, n := range []string{"logger", "grip", "sender"} {
		made := MakeBase(n, func() {}, func() error { return nil })
		newed := NewBase(n)
		if newed.name != made.name {
			t.Error("elements should be equal")
		}
		if newed.level != made.level {
			t.Error("elements should be equal")
		}
		if newed.closer() != made.closer() {
			t.Error("elements should be equal")
		}

		for _, s := range []*Base{made, newed} {
			s.SetFormatter(nil)
			s.SetErrorHandler(nil)
			s.SetErrorHandler(handler)
			s.ErrorHandler()(errors.New("failed"), message.MakeString("fated"))
		}
	}

	if sink.Len() != 6 {
		t.Error("elements should be equal")
	}
	if !sink.HasMessage() {
		t.Error("should be true")
	}
}
