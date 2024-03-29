package github

import (
	"context"

	"github.com/google/go-github/github"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type githubCommentLogger struct {
	issue int
	opts  *GithubOptions
	gh    githubClient
	send.Base
}

// NewCommentSender creates a new Sender implementation that
// adds a comment to a github issue (or pull request) for every log
// message sent.
//
// Specify the credentials to use the GitHub via the GithubOptions
// structure, and the issue number as an argument to the constructor.
func NewCommentSender(name string, issueID int, opts *GithubOptions) (send.Sender, error) {
	s := &githubCommentLogger{
		opts:  opts,
		issue: issueID,
		gh:    &githubClientImpl{},
	}

	ctx := context.TODO()

	s.gh.Init(ctx, opts.Token)

	s.SetName(name)
	s.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))
	s.SetFormatter(send.MakeDefaultFormatter())

	return s, nil
}

func (s *githubCommentLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
		text, err := s.Format(m)
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}

		comment := &github.IssueComment{Body: &text}

		ctx := context.TODO()
		_, _, err = s.gh.CreateComment(ctx, s.opts.Account, s.opts.Repo, s.issue, comment)
		if err != nil {
			s.HandleError(send.WrapError(err, m))
		}
	}
}
