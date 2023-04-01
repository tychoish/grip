package twitter

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type twitterLogger struct {
	twitter twitterClient
	send.Base
}

// Options describes the credentials required to connect to the
// twitter API. While the name is used for internal reporting, the
// other values should be populated with credentials obtained from the
// Twitter API.
type Options struct {
	Name           string
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
}

func (opts *Options) resolve(ctx context.Context) *twitter.Client {
	return twitter.NewClient(oauth1.NewConfig(opts.ConsumerKey, opts.ConsumerSecret).
		Client(ctx, oauth1.NewToken(opts.AccessToken, opts.AccessSecret)))
}

// MakeSender constructs a default sender implementation that
// posts messages to a twitter account. The implementation does not
// rate limit outgoing messages, which should be the responsibility of
// the caller.
func MakeSender(ctx context.Context, opts *Options) (send.Sender, error) {
	s := &twitterLogger{
		twitter: newTwitterClient(ctx, opts),
	}
	fallback := log.New(os.Stdout, "", log.LstdFlags)
	s.SetErrorHandler(send.ErrorHandlerFromLogger(fallback))
	s.SetResetHook(func() {
		fallback.SetPrefix(fmt.Sprintf("[%s] ", s.Name()))
	})

	s.SetName(opts.Name)

	if err := s.twitter.Verify(); err != nil {
		return nil, fmt.Errorf("problem connecting to twitter: %w", err)
	}

	return s, nil
}

func (s *twitterLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
		if err := s.twitter.Send(m.String()); err != nil {
			s.ErrorHandler()(err, m)
		}
	}
}

type twitterClient interface {
	Verify() error
	Send(string) error
}

type twitterClientImpl struct {
	twitter *twitter.Client
}

func newTwitterClient(ctx context.Context, opts *Options) twitterClient {
	return &twitterClientImpl{twitter: opts.resolve(ctx)}
}

func (tc *twitterClientImpl) Verify() error {
	_, _, err := tc.twitter.Accounts.VerifyCredentials(&twitter.AccountVerifyParams{})
	return fmt.Errorf("could not verify account: %w", err)
}

func (tc *twitterClientImpl) Send(in string) error {
	_, _, err := tc.twitter.Statuses.Update(in, nil)
	return err
}
