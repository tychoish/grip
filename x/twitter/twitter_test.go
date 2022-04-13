package send

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	return &twitterLogger{
		Base:    send.NewBase("mock-twitter"),
		twitter: client,
	}
}

func TestTwitter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Constructors", func(t *testing.T) {
		t.Run("NilCredentialsPanic", func(t *testing.T) {
			assert.Panics(t, func() { _, _ = MakeSender(ctx, nil) })
		})
		t.Run("EmptyCredentialsError", func(t *testing.T) {
			s, err := MakeSender(ctx, &TwitterOptions{})
			assert.Error(t, err)
			assert.Nil(t, s)
		})
		t.Run("SetInvalidLevel", func(t *testing.T) {
			s, err := NewSender(ctx, &TwitterOptions{}, send.LevelInfo{Default: -1, Threshold: -1})
			assert.Error(t, err)
			assert.Nil(t, s)
		})
	})
	t.Run("MockSending", func(t *testing.T) {
		mock := &twitterClientMock{}
		t.Run("Flush", func(t *testing.T) {
			s := newMockedTwitterSender(mock)
			assert.NoError(t, s.Flush(ctx))
		})
		mock.reset()
		t.Run("SendValidCase", func(t *testing.T) {
			msg := message.NewSimpleStringMessage(level.Info, "hi")
			s := newMockedTwitterSender(mock)
			s.Send(msg)
			assert.Equal(t, msg.String(), mock.Content)
		})
		mock.reset()
		t.Run("WithError", func(t *testing.T) {
			errsender, err := send.NewInternalLogger("errr", send.LevelInfo{Default: level.Info, Threshold: level.Info})
			require.NoError(t, err)
			s := newMockedTwitterSender(mock)
			require.NoError(t, s.SetErrorHandler(send.ErrorHandlerFromSender(errsender)))
			mock.SendError = errors.New("sendERROR")

			msg := message.NewSimpleStringMessage(level.Info, "hi")
			s.Send(msg)
			assert.True(t, errsender.HasMessage())
			assert.Contains(t, errsender.GetMessage().Message.String(), "sendERROR")
		})
		mock.reset()
	})
	t.Run("WithError", func(t *testing.T) {
		errsender, err := send.NewInternalLogger("errr", send.LevelInfo{Default: level.Info, Threshold: level.Info})
		require.NoError(t, err)

		s := &twitterLogger{
			twitter: &twitterClientImpl{twitter: (&TwitterOptions{}).resolve(ctx)},
			Base:    send.NewBase("fake"),
		}

		require.NoError(t, s.SetErrorHandler(send.ErrorHandlerFromSender(errsender)))

		msg := message.NewSimpleStringMessage(level.Info, "hi")
		s.Send(msg)
		require.True(t, errsender.HasMessage())
		assert.Contains(t, errsender.GetMessage().Message.String(), "Bad Authentication data.")
	})

}
