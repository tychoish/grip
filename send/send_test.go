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

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/assert/check"
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

	senders["writer"] = MakeWriter(MakePlain())

	var err error
	var plain, plainerr, plainfile Sender
	plain = MakePlain()
	err = plain.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	plain.SetName("plain")

	senders["plain"] = plain

	plainerr = MakePlainStdError()
	plainerr.SetName("plain.err")
	err = plainerr.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	senders["plain.err"] = plainerr

	plainfile, err = MakePlainFile(filepath.Join(tempDir, "plain.file"))
	if err != nil {
		t.Fatal(err)
	}
	plainfile.SetName("plain.file")
	err = plainfile.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	senders["plain.file"] = plainfile

	var asyncOne, asyncTwo Sender
	asyncOne = MakeStdOutput()
	asyncOne.SetName("async-one")
	err = asyncOne.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	asyncTwo = MakeStdOutput()
	asyncTwo.SetName("async-two")
	err = asyncTwo.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	senders["async"] = MakeAsyncGroup(context.Background(), 16, asyncOne, asyncTwo)

	nativeErr := MakeStdError()
	nativeErr.SetName("error")
	err = nativeErr.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	senders["error"] = nativeErr

	nativeFile, err := MakeFile(filepath.Join(tempDir, "file"))
	if err != nil {
		t.Fatal(err)
	}
	nativeFile.SetName("native-file")
	err = nativeFile.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	senders["native-file"] = nativeFile

	callsite := MakeCallSite(1)
	callsite.SetName("callsite")
	if err = callsite.SetLevel(l); err != nil {
		t.Fatal(err)
	}
	senders["callsite"] = callsite

	callsiteFile, err := MakeCallSiteFile(filepath.Join(tempDir, "cs"), 1)
	callsiteFile.SetName("callsite")
	if err != nil {
		t.Fatal(err)
	}
	if err = callsiteFile.SetLevel(l); err != nil {
		t.Fatal(err)
	}
	senders["callsite-file"] = callsiteFile

	jsons := MakeJSON()
	jsons.SetName("json")
	if err = jsons.SetLevel(LevelInfo{level.Info, level.Notice}); err != nil {
		t.Fatal(err)
	}
	senders["json"] = jsons

	jsonf, err := MakeJSONFile(filepath.Join(tempDir, "js"))
	if err != nil {
		t.Fatal(err)
	}
	if err = jsonf.SetLevel(l); err != nil {
		t.Error(err)
	}
	jsonf.SetName("json")
	senders["json"] = jsonf

	var sender Sender
	multiSenders := []Sender{}
	for i := 0; i < 4; i++ {
		sender = MakeStdOutput()
		sender.SetName(fmt.Sprintf("native-%d", i))
		if err = sender.SetLevel(l); err != nil {
			t.Fatal(err)
		}
		multiSenders = append(multiSenders, sender)
	}

	multi, err := NewMulti("multi", multiSenders)
	if err != nil {
		t.Fatal(err)
	}
	if err = multi.SetLevel(l); err != nil {
		t.Fatal(err)
	}
	senders["multi"] = multi

	bufferedInternal := MakeStdOutput()
	bufferedInternal.SetName("buffered")
	err = bufferedInternal.SetLevel(l)
	if err != nil {
		t.Fatal(err)
	}
	senders["buffered"] = MakeBuffered(bufferedInternal, minInterval, 1)

	annotatingBase := MakeStdOutput()
	annotatingBase.SetName("async-one")
	err = annotatingBase.SetLevel(l)
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

	senders["testing"] = MakeTesting(t)

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
	t.Parallel()
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
	t.Parallel()
	levels := []LevelInfo{
		{level.Invalid, level.Invalid},
		{level.Priority(0), level.Priority(0)},
		{level.Debug, level.Priority(0)},
		{level.Priority(0), level.Info},
		{level.Priority(225), level.Priority(225)},
	}

	for n, sender := range senderFixture(t) {
		if n == "async" {
			// the async sender doesn't meaningfully have
			// its own level because it passes this down
			// to its constituent senders.
			continue
		}
		t.Run(n, func(t *testing.T) {
			if err := sender.SetLevel(LevelInfo{level.Debug, level.Alert}); err != nil {
				t.Fatal(err)
			}

			for idx, l := range levels {
				t.Run(fmt.Sprint(idx), func(t *testing.T) {
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
				})
			}
		})
	}
}

func TestCloserShouldUsuallyNoop(t *testing.T) {
	t.Parallel()
	for _, sender := range senderFixture(t) {
		if err := sender.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRejectBadSetLevels(t *testing.T) {
	t.Parallel()
	for name, sender := range senderFixture(t) {
		t.Run(name, func(t *testing.T) {
			if err := sender.SetLevel(LevelInfo{level.Info, level.Info}); err != nil {
				t.Error(err)
			}
			if err := sender.SetLevel(LevelInfo{}); err == nil {
				t.Error("should not be able to set invalid level")
			}
		})
	}
}

func TestBasicNoopSendTest(t *testing.T) {
	t.Parallel()
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	for name, sender := range functionalMockSenders(t, senderFixture(t)) {
		t.Run(name, func(t *testing.T) {
			for i := -10; i <= 110; i += 5 {
				m := NewString(level.Priority(i), "hello world! "+randomString(10, rand))
				sender.Send(m)
			}
		})

	}
}

func TestBaseConstructor(t *testing.T) {
	t.Parallel()

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

func TestSenderConstructorFails(t *testing.T) {
	var err error
	_, err = MakeJSONFile("/root/log")
	check.Error(t, err)
	check.ErrorIs(t, err, os.ErrPermission)

	_, err = MakeCallSiteFile("/root/log", 1)
	check.Error(t, err)
	check.ErrorIs(t, err, os.ErrPermission)

	_, err = MakePlainFile("/root/log")
	check.Error(t, err)
	check.ErrorIs(t, err, os.ErrPermission)

	_, err = MakeFile("/root/log")
	check.Error(t, err)
	check.ErrorIs(t, err, os.ErrPermission)
}

func TestWrapping(t *testing.T) {
	base := MakePlain()

	for name, sender := range map[string]Sender{
		"Annotating":  MakeAnnotating(base, map[string]any{"hello": 52}),
		"Buffered":    MakeBuffered(base, time.Millisecond, 10),
		"Interceptor": MakeFilter(base, func(c message.Composer) {}),
		"Writer":      MakeWriter(base),
	} {
		t.Run(name, func(t *testing.T) {
			us := fun.Unwrap(sender)
			check.True(t, fun.IsWrapped(sender))
			check.True(t, us != nil)
			check.True(t, us == base)
		})
	}
}
