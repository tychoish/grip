package twitter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type twitterClientMock struct {
	VerifyError error
	SendError   error
	Content     string
}

func (tm *twitterClientMock) Verify() error        { return tm.VerifyError }
func (tm *twitterClientMock) Send(in string) error { tm.Content = in; return tm.SendError }
func (tm *twitterClientMock) reset()               { tm.VerifyError = nil; tm.SendError = nil; tm.Content = "" }

func newMockedTwitterSender(client *twitterClientMock) *twitterLogger {
	s := &twitterLogger{
		twitter: client,
	}
	s.SetName("mock-twitter")
	s.SetPriority(level.Info)
	return s
}

func TestTwitter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Constructors", func(t *testing.T) {
		t.Run("NilCredentialsPanic", func(t *testing.T) {
			func() {
				defer func() {
					if p := recover(); p == nil {
						t.Error("should have panic'd")
					}

				}()

				_, _ = MakeSender(ctx, nil)
			}()
		})
		t.Run("EmptyCredentialsError", func(t *testing.T) {
			s, err := MakeSender(ctx, &Options{})
			if err == nil {
				t.Error("expected error condition")
			}
			if s != nil {
				t.Fatal("constructor should not make an object in an error condition")
			}
		})
	})
	t.Run("MockSending", func(t *testing.T) {
		mock := &twitterClientMock{}
		t.Run("Flush", func(t *testing.T) {
			s := newMockedTwitterSender(mock)
			if err := s.Flush(ctx); err != nil {
				t.Error(err)
			}
		})
		mock.reset()
		t.Run("SendValidCase", func(t *testing.T) {
			msg := message.MakeString("hi")
			msg.SetPriority(level.Info)
			s := newMockedTwitterSender(mock)
			s.Send(msg)
			if mock.Content != msg.String() {
				t.Error("elements should be equal")
			}
		})
		mock.reset()
		t.Run("WithError", func(t *testing.T) {
			errsender := send.MakeInternal()
			errsender.SetName("errr")
			errsender.SetPriority(level.Info)

			s := newMockedTwitterSender(mock)
			s.SetErrorHandler(send.ErrorHandlerFromSender(errsender))
			mock.SendError = errors.New("sendERROR")

			msg := message.MakeString("hi")
			msg.SetPriority(level.Notice)
			s.Send(msg)
			if !errsender.HasMessage() {
				t.Error("should be true")
			}
			if !strings.Contains(errsender.GetMessage().Message.String(), "sendERROR") {
				t.Error("malformed string")
			}
		})
		mock.reset()
	})
	t.Run("WithError", func(t *testing.T) {
		errsender := send.NewInternal(2)
		errsender.SetName("errr")
		errsender.SetPriority(level.Info)

		s := &twitterLogger{
			twitter: &twitterClientImpl{twitter: (&Options{}).resolve(ctx)},
		}
		s.SetName("fake")
		s.SetErrorHandler(send.ErrorHandlerFromSender(errsender))

		msg := message.MakeString("hi")
		msg.SetPriority(level.Info)
		s.Send(msg)
		if !errsender.HasMessage() {
			t.Fatal("should have messages")
		}
		if !strings.Contains(errsender.GetMessage().Message.String(), "Bad Authentication data.") {
			t.Error("malformed string")
		}
	})
}
