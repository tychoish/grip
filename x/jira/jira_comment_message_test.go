package jira

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type JiraCommentSuite struct {
	opts *Options
	suite.Suite
}

func TestJiraCommentSuite(t *testing.T) {
	suite.Run(t, new(JiraCommentSuite))
}

func (j *JiraCommentSuite) SetupTest() {
	j.opts = &Options{
		BaseURL: "url",
		BasicAuthOpts: BasicAuth{
			Username: "username",
			Password: "password",
		},
		client: &jiraClientMock{},
		Name:   "1234",
	}
}

func (j *JiraCommentSuite) TestMockSenderWithNewConstructor() {
	sender, err := NewCommentSender(context.Background(), "1234", j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)
}

func (j *JiraCommentSuite) TestConstructorMustCreate() {
	j.opts.client = &jiraClientMock{failCreate: true}
	sender, err := NewCommentSender(context.Background(), "1234", j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.Nil(sender)
	j.Error(err)
}

func (j *JiraCommentSuite) TestConstructorMustPassAuthTest() {
	j.opts.client = &jiraClientMock{failAuth: true}
	sender, err := NewCommentSender(context.Background(), "1234", j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.Nil(sender)
	j.Error(err)
}

func (j *JiraCommentSuite) TestConstructorErrorsWithInvalidConfigs() {
	sender, err := NewCommentSender(context.Background(), "1234", nil, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.Nil(sender)
	j.Error(err)

	sender, err = NewIssueSender(context.Background(), &Options{}, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.Nil(sender)
	j.Error(err)
}

func (j *JiraCommentSuite) TestSendMethod() {
	numShouldHaveSent := 0
	sender, err := NewCommentSender(context.Background(), "1234", j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NoError(err)
	j.Require().NotNil(sender)

	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)

	m := message.NewDefaultMessage(level.Debug, "sending debug level comment")
	sender.Send(m)
	j.Equal(mock.numSent, numShouldHaveSent)

	m = message.NewDefaultMessage(level.Alert, "sending alert level comment")
	sender.Send(m)
	numShouldHaveSent++
	j.Equal(mock.numSent, numShouldHaveSent)

	m = message.NewDefaultMessage(level.Emergency, "sending emergency level comment")
	sender.Send(m)
	numShouldHaveSent++
	j.Equal(mock.numSent, numShouldHaveSent)
}

func (j *JiraCommentSuite) TestSendMethodWithError() {
	sender, err := NewCommentSender(context.Background(), "1234", j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)

	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)
	j.False(mock.failSend)

	m := message.NewDefaultMessage(level.Alert, "test")
	sender.Send(m)
	j.Equal(mock.numSent, 1)

	mock.failSend = true
	sender.Send(m)
	j.Equal(mock.numSent, 1)
}

func (j *JiraCommentSuite) TestCreateMethodChangesClientState() {
	base := &jiraClientImpl{}
	new := &jiraClientImpl{}

	j.Equal(base, new)
	j.NoError(new.CreateClient(nil, "foo"))
	j.NotEqual(base, new)
}

func (j *JiraCommentSuite) TestSendWithJiraIssueComposer() {
	c := NewComment(level.Notice, "ABC-123", "Hi")

	sender, err := NewCommentSender(context.Background(), "XYZ-123", j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NoError(err)
	j.Require().NotNil(sender)

	sender.Send(c)

	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(1, mock.numSent)
	j.Equal("ABC-123", mock.lastIssue)
}

func TestJiraMessageComposerConstructor(t *testing.T) {
	const testMsg = "hello"
	assert := assert.New(t) // nolint
	reporterField := Field{Key: "Reporter", Value: "Annie"}
	assigneeField := Field{Key: "Assignee", Value: "Sejin"}
	typeField := Field{Key: "Type", Value: "Bug"}
	labelsField := Field{Key: "Labels", Value: []string{"Soul", "Pop"}}
	unknownField := Field{Key: "Artist", Value: "Adele"}
	msg := NewIssue("project", testMsg, reporterField, assigneeField, typeField, labelsField, unknownField)
	issue := msg.Raw().(*Issue)

	assert.Equal(issue.Project, "project")
	assert.Equal(issue.Summary, testMsg)
	assert.Equal(issue.Reporter, reporterField.Value)
	assert.Equal(issue.Assignee, assigneeField.Value)
	assert.Equal(issue.Type, typeField.Value)
	assert.Equal(issue.Labels, labelsField.Value)
	assert.Equal(issue.Fields[unknownField.Key], unknownField.Value)
}

func TestJiraIssueAnnotationOnlySupportsStrings(t *testing.T) {
	assert := assert.New(t) // nolint

	m := &jiraMessage{
		issue: &Issue{},
	}

	assert.Error(m.Annotate("k", 1))
	assert.Error(m.Annotate("k", true))
	assert.Error(m.Annotate("k", nil))
}
