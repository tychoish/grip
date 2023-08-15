package jira

import (
	"context"
	"fmt"

	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type jiraCommentJournal struct {
	issueID string
	opts    *Options
	send.Base
}

// MakeCommentSender is the same as NewJiraCommentLogger but uses a warning
// level of Trace
func MakeCommentSender(ctx context.Context, id string, opts *Options) (send.Sender, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	j := &jiraCommentJournal{
		opts:    opts,
		issueID: id,
	}
	if err := j.opts.client.CreateClient(opts.HTTPClient, opts.BaseURL); err != nil {
		return nil, err
	}

	authOpts := jiraAuthOpts{
		username:           opts.BasicAuthOpts.Username,
		password:           opts.BasicAuthOpts.Password,
		addBasicAuthHeader: opts.BasicAuthOpts.UseBasicAuth,
		accessToken:        opts.Oauth1Opts.AccessToken,
		tokenSecret:        opts.Oauth1Opts.TokenSecret,
		privateKey:         opts.Oauth1Opts.PrivateKey,
		consumerKey:        opts.Oauth1Opts.ConsumerKey,
	}
	if err := j.opts.client.Authenticate(ctx, authOpts); err != nil {
		return nil, fmt.Errorf("jira authentication error: %v", err)
	}

	j.SetName(fmt.Sprint(opts.Name, id))
	j.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))

	return j, nil
}

// Send post issues via jiraCommentJournal with information in the message.Composer
func (j *jiraCommentJournal) Send(m message.Composer) {
	if send.ShouldLog(j, m) {
		issue := j.issueID
		if c, ok := m.Raw().(*Comment); ok {
			issue = c.IssueID
		}
		if err := j.opts.client.PostComment(issue, m.String()); err != nil {
			j.HandleError(send.WrapError(err, m))
		}
	}
}
