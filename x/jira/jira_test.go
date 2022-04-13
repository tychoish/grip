package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type JiraSuite struct {
	opts *Options
	suite.Suite
}

func TestJiraSuite(t *testing.T) {
	suite.Run(t, new(JiraSuite))
}

func (j *JiraSuite) SetupSuite() {}

func (j *JiraSuite) SetupTest() {
	j.opts = &Options{
		Name:    "bot",
		BaseURL: "url",
		BasicAuthOpts: BasicAuth{
			Username: "username",
			Password: "password",
		},
		client: &jiraClientMock{},
	}
}

func (j *JiraSuite) TestMockSenderWithNewConstructor() {
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)
}

func (j *JiraSuite) TestConstructorMustCreate() {
	j.opts.client = &jiraClientMock{failCreate: true}
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})

	j.Nil(sender)
	j.Error(err)
}

func (j *JiraSuite) TestConstructorMustPassAuthTest() {
	j.opts.client = &jiraClientMock{failAuth: true}
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})

	j.Nil(sender)
	j.Error(err)
}

func (j *JiraSuite) TestConstructorErrorsWithInvalidConfigs() {
	sender, err := NewIssueSender(context.Background(), nil, send.LevelInfo{Default: level.Trace, Threshold: level.Info})

	j.Nil(sender)
	j.Error(err)

	sender, err = NewIssueSender(context.Background(), &Options{}, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.Nil(sender)
	j.Error(err)

	opts := &Options{
		BasicAuthOpts: BasicAuth{
			Username: "foo",
			Password: "bar",
		},
		Oauth1Opts: Oauth1{
			AccessToken: "12345",
		},
	}

	sender, err = NewIssueSender(context.Background(), opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.Nil(sender)
	j.EqualError(err, "must specify exactly 1 method of authentication")
}

func (j *JiraSuite) TestSendMethod() {
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)

	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)

	m := message.NewDefaultMessage(level.Debug, "hello")
	sender.Send(m)
	j.Equal(mock.numSent, 0)

	m = message.NewDefaultMessage(level.Alert, "")
	sender.Send(m)
	j.Equal(mock.numSent, 0)

	m = message.NewDefaultMessage(level.Alert, "world")
	sender.Send(m)
	j.Equal(mock.numSent, 1)
}

func (j *JiraSuite) TestSendMethodWithError() {
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)

	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)
	j.False(mock.failSend)

	m := message.NewDefaultMessage(level.Alert, "world")
	sender.Send(m)
	j.Equal(mock.numSent, 1)

	mock.failSend = true
	sender.Send(m)
	j.Equal(mock.numSent, 1)
}

func (j *JiraSuite) TestCreateMethodChangesClientState() {
	base := &jiraClientImpl{}
	new := &jiraClientImpl{}

	j.Equal(base, new)
	_ = new.CreateClient(nil, "foo")
	j.NotEqual(base, new)
}

// Test get fields
func (j *JiraSuite) TestGetFieldsWithJiraIssue() {
	project := "Hello"
	summary := "it's me"

	// Test fields
	reporterField := Field{Key: "Reporter", Value: "Annie"}
	assigneeField := Field{Key: "Assignee", Value: "Sejin"}
	typeField := Field{Key: "Type", Value: "Bug"}
	labelsField := Field{Key: "Labels", Value: []string{"Soul", "Pop"}}
	unknownField := Field{Key: "Artist", Value: "Adele"}

	// Test One: Only Summary and Project
	m1 := NewIssue(project, summary)
	fields := getFields(m1)

	j.Equal(fields.Project.Key, project)
	j.Equal(fields.Summary, summary)
	j.Nil(fields.Reporter)
	j.Nil(fields.Assignee)
	j.Equal(fields.Type.Name, "Task")
	j.Nil(fields.Labels)
	j.Nil(fields.Unknowns)

	// Test Two: with reporter, assignee and type
	m2 := NewIssue(project, summary, reporterField, assigneeField,
		typeField, labelsField)
	fields = getFields(m2)

	j.Equal(fields.Reporter.Name, "Annie")
	j.Equal(fields.Assignee.Name, "Sejin")
	j.Equal(fields.Type.Name, "Bug")
	j.Equal(fields.Labels, []string{"Soul", "Pop"})
	j.Nil(fields.Unknowns)

	// Test Three: everything plus Unknown fields
	m3 := NewIssue(project, summary, reporterField, assigneeField,
		typeField, unknownField)
	fields = getFields(m3)
	j.Equal(fields.Unknowns["Artist"], "Adele")
}

func (j *JiraSuite) TestGetFieldsWithFields() {
	testFields := message.Fields{"key0": 12, "key1": 42}
	msg := "Get the message"
	m := message.NewFieldsMessage(msg, testFields)

	fields := getFields(m)
	j.Equal(fields.Summary, msg)
	j.NotNil(fields.Description)
}

func (j *JiraSuite) TestTruncate() {
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)

	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)

	m := message.NewDefaultMessage(level.Info, "aaa")
	j.True(m.Loggable())
	sender.Send(m)
	j.Len(mock.lastSummary, 3)

	var longString bytes.Buffer
	for i := 0; i < 1000; i++ {
		longString.WriteString("a")
	}
	m = message.NewDefaultMessage(level.Info, longString.String())
	j.True(m.Loggable())
	sender.Send(m)
	j.Len(mock.lastSummary, 254)

	buffer := bytes.NewBufferString("")
	buffer.Grow(40000)
	for i := 0; i < 40000; i++ {
		buffer.WriteString("a")
	}

	m = message.NewDefaultMessage(level.Info, buffer.String())
	j.True(m.Loggable())
	sender.Send(m)
	j.Len(mock.lastDescription, 32767)
}

func (j *JiraSuite) TestCustomFields() {
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)

	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)

	jiraIssue := &Issue{
		Summary: "test",
		Type:    "type",
		Fields: map[string]interface{}{
			"customfield_12345": []string{"hi", "bye"},
		},
	}

	m := MakeIssue(jiraIssue)
	j.NoError(m.SetPriority(level.Warning))
	j.True(m.Loggable())
	sender.Send(m)

	j.Equal([]string{"hi", "bye"}, mock.lastFields.Unknowns["customfield_12345"])
	j.Equal("test", mock.lastFields.Summary)

	bytes, err := json.Marshal(&mock.lastFields)
	j.NoError(err)
	j.Len(bytes, 79)
	j.Equal(`{"customfield_12345":["hi","bye"],"issuetype":{"name":"type"},"summary":"test"}`, string(bytes))
}

func (j *JiraSuite) TestPopulateKey() {
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)
	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)

	count := 0
	jiraIssue := &Issue{
		Summary: "foo",
		Type:    "bug",
		Callback: func(_ string) {
			count++
		},
	}

	j.Equal(0, count)
	m := MakeIssue(jiraIssue)
	j.NoError(m.SetPriority(level.Alert))
	j.True(m.Loggable())
	sender.Send(m)
	j.Equal(1, count)
	issue := m.Raw().(*Issue)
	j.Equal(mock.issueKey, issue.IssueKey)

	messageFields := message.MakeFieldsMessage(level.Info, "something", message.Fields{
		"message": "foo",
	})
	j.True(messageFields.Loggable())
	sender.Send(messageFields)
	messageIssue := messageFields.Raw().(message.Fields)
	j.Equal(mock.issueKey, messageIssue[jiraIssueKey])
}

func (j *JiraSuite) TestWhenCallbackNil() {
	sender, err := NewIssueSender(context.Background(), j.opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	j.NotNil(sender)
	j.NoError(err)
	mock, ok := j.opts.client.(*jiraClientMock)
	j.True(ok)
	j.Equal(mock.numSent, 0)

	jiraIssue := &Issue{
		Summary: "foo",
		Type:    "bug",
	}

	m := MakeIssue(jiraIssue)
	j.NoError(m.SetPriority(level.Alert))
	j.True(m.Loggable())
	j.NotPanics(func() {
		sender.Send(m)
	})
}
