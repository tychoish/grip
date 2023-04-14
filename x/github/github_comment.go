package github

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/github"
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

	fallback := log.New(os.Stdout, "", log.LstdFlags)
	s.SetName(name)
	s.SetErrorHandler(send.ErrorHandlerFromLogger(fallback))
	s.SetFormatter(send.MakeDefaultFormatter())
	s.SetResetHook(func() {
		fallback.SetPrefix(fmt.Sprintf("[%s] [%s/%s#%d] ", s.Name(), opts.Account, opts.Repo, issueID))
	})

	return s, nil
}

func (s *githubCommentLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
		text, err := s.Formatter()(m)
		if err != nil {
			s.ErrorHandler()(err, m)
			return
		}

		comment := &github.IssueComment{Body: &text}

		ctx := context.TODO()
		_, _, err = s.gh.CreateComment(ctx, s.opts.Account, s.opts.Repo, s.issue, comment)
		if err != nil {
			s.ErrorHandler()(err, m)
		}
	}
}
