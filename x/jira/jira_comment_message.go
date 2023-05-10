package jira

import (
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

type jiraComment struct {
	Payload Comment `bson:"payload" json:"payload" yaml:"payload"`

	message.Base `bson:"meta" json:"meta" yaml:"meta"`
}

// Comment represents a single comment to post to the given JIRA issue
type Comment struct {
	IssueID string `bson:"issue_id,omitempty" json:"issue_id,omitempty" yaml:"issue_id,omitempty"`
	Body    string `bson:"body" json:"body" yaml:"body"`
}

// NewComment returns a self-contained composer for posting a comment
// to a single JIRA issue. This composer will override the issue set in the
// JIRA sender
func NewComment(p level.Priority, issueID, body string) message.Composer {
	s := MakeComment(issueID, body)
	s.SetPriority(p)

	return s
}

// MakeComment returns a self-contained composer for posting a comment
// to a single JIRA issue. This composer will override the issue set in the
// JIRA sender. The composer will not have a priority set
func MakeComment(issueID, body string) message.Composer {
	return &jiraComment{
		Payload: Comment{
			IssueID: issueID,
			Body:    body,
		},
	}
}

func (c *jiraComment) Loggable() bool { return len(c.Payload.IssueID) > 0 && len(c.Payload.Body) > 0 }
func (*jiraComment) Structured() bool { return false }
func (c *jiraComment) String() string { return c.Payload.Body }
func (c *jiraComment) Raw() any       { return &c.Payload }
