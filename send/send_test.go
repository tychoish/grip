package send

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

type SenderSuite struct {
	senders map[string]Sender
	rand    *rand.Rand
	tempDir string
	suite.Suite
}

func TestSenderSuite(t *testing.T) {
	suite.Run(t, new(SenderSuite))
}

func (s *SenderSuite) SetupSuite() {
	var err error
	s.rand = rand.New(rand.NewSource(time.Now().Unix()))
	s.tempDir, err = ioutil.TempDir("", "sender-test-")
	s.Require().NoError(err)
}

func (s *SenderSuite) SetupTest() {
	s.Require().NoError(os.MkdirAll(s.tempDir, 0766))

	l := LevelInfo{level.Info, level.Notice}
	s.senders = map[string]Sender{
		// "slack": &slackJournal{Base: NewBase("slack")},
		// "xmpp":  &xmppLogger{Base: NewBase("xmpp")},
	}

	internal := MakeInternalLogger()
	internal.name = "internal"
	internal.output = make(chan *InternalMessage)
	s.senders["internal"] = internal

	native, err := NewStdOutput("native", l)
	s.Require().NoError(err)
	s.senders["native"] = native

	s.senders["writer"] = MakeWriter(native)

	var plain, plainerr, plainfile Sender
	plain, err = NewPlainStdOutput("plain", l)
	s.Require().NoError(err)
	s.senders["plain"] = plain

	plainerr, err = NewPlainStdError("plain.err", l)
	s.Require().NoError(err)
	s.senders["plain.err"] = plainerr

	plainfile, err = NewPlainFile("plain.file", filepath.Join(s.tempDir, "plain.file"), l)
	s.Require().NoError(err)
	s.senders["plain.file"] = plainfile

	var asyncOne, asyncTwo Sender
	asyncOne, err = NewStdOutput("async-one", l)
	s.Require().NoError(err)
	asyncTwo, err = NewStdOutput("async-two", l)
	s.Require().NoError(err)
	s.senders["async"] = NewAsyncGroup(context.Background(), 16, asyncOne, asyncTwo)

	nativeErr, err := NewStdError("error", l)
	s.Require().NoError(err)
	s.senders["error"] = nativeErr

	nativeFile, err := NewFile("native-file", filepath.Join(s.tempDir, "file"), l)
	s.Require().NoError(err)
	s.senders["native-file"] = nativeFile

	callsite, err := NewCallSit("callsite", 1, l)
	s.Require().NoError(err)
	s.senders["callsite"] = callsite

	callsiteFile, err := NewCallSiteFile("callsite", filepath.Join(s.tempDir, "cs"), 1, l)
	s.Require().NoError(err)
	s.senders["callsite-file"] = callsiteFile

	jsons, err := NewJSON("json", LevelInfo{level.Info, level.Notice})
	s.Require().NoError(err)
	s.senders["json"] = jsons

	jsonf, err := NewJSONFile("json", filepath.Join(s.tempDir, "js"), l)
	s.Require().NoError(err)
	s.senders["json"] = jsonf

	var sender Sender
	multiSenders := []Sender{}
	for i := 0; i < 4; i++ {
		sender, err = NewStdOutput(fmt.Sprintf("native-%d", i), l)
		s.Require().NoError(err)
		multiSenders = append(multiSenders, sender)
	}

	multi, err := NewMulti("multi", l, multiSenders)
	s.Require().NoError(err)
	s.senders["multi"] = multi

	bufferedInternal, err := NewStdOutput("buffered", l)
	s.Require().NoError(err)
	s.senders["buffered"] = NewBuffered(bufferedInternal, minInterval, 1)

	annotatingBase, err := NewStdOutput("async-one", l)
	s.Require().NoError(err)
	s.senders["annotating"] = MakeAnnotating(annotatingBase, map[string]interface{}{
		"one":    1,
		"true":   true,
		"string": "string",
	})

	for _, size := range []int{1, 100, 10000, 1000000} {
		name := fmt.Sprintf("inmemory-%d", size)
		s.senders[name], err = NewInMemorySender(name, l, size)
		s.Require().NoError(err)
		s.senders[name].SetFormatter(MakeDefaultFormatter())
	}
}

func (s *SenderSuite) TearDownTest() {
	if runtime.GOOS == "windows" {
		_ = s.senders["native-file"].Close()
		_ = s.senders["callsite-file"].Close()
		_ = s.senders["json"].Close()
		_ = s.senders["plain.file"].Close()
	}
	s.Require().NoError(os.RemoveAll(s.tempDir))
}

func (s *SenderSuite) functionalMockSenders() map[string]Sender {
	out := map[string]Sender{}
	for t, sender := range s.senders {
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

func (s *SenderSuite) TearDownSuite() {
	s.NoError(s.senders["internal"].Close())
}

func (s *SenderSuite) TestSenderImplementsInterface() {
	// this actually won't catch the error; the compiler will in
	// the fixtures, but either way we need to make sure that the
	// tests actually enforce this.
	for name, sender := range s.senders {
		s.Implements((*Sender)(nil), sender, name)
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

func (s *SenderSuite) TestNameSetterRoundTrip() {
	for n, sender := range s.senders {
		for i := 0; i < 100; i++ {
			name := randomString(12, s.rand)
			s.NotEqual(sender.Name(), name, n)
			sender.SetName(name)
			s.Equal(sender.Name(), name, n)
		}
	}
}

func (s *SenderSuite) TestLevelSetterRejectsInvalidSettings() {
	levels := []LevelInfo{
		{level.Invalid, level.Invalid},
		{level.Priority(-10), level.Priority(-1)},
		{level.Debug, level.Priority(-1)},
		{level.Priority(800), level.Priority(-2)},
	}

	for n, sender := range s.senders {
		if n == "async" {
			// the async sender doesn't meaningfully have
			// its own level because it passes this down
			// to its constituent senders.
			continue
		}

		s.NoError(sender.SetLevel(LevelInfo{level.Debug, level.Alert}))
		for _, l := range levels {
			s.True(sender.Level().Valid(), n)
			s.False(l.Valid(), n)
			s.Error(sender.SetLevel(l), n)
			s.True(sender.Level().Valid(), n)
			s.NotEqual(sender.Level(), l, n)
		}

	}
}

func (s *SenderSuite) TestCloserShouldUsuallyNoop() {
	for t, sender := range s.senders {
		s.NoError(sender.Close(), t)
	}
}

func (s *SenderSuite) TestBasicNoopSendTest() {
	for _, sender := range s.functionalMockSenders() {
		for i := -10; i <= 110; i += 5 {
			m := message.NewString(level.Priority(i), "hello world! "+randomString(10, s.rand))
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
