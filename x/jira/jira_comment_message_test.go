package jira

import (
	"context"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func setupFixture() *Options {
	return &Options{
		BaseURL: "url",
		BasicAuthOpts: BasicAuth{
			Username: "username",
			Password: "password",
		},
		client: &jiraClientMock{},
		Name:   "1234",
	}
}

func TestCommentMockSenderWithNewConstructor(t *testing.T) {
	opts := setupFixture()

	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}
}

func TestCommentConstructorMustCreate(t *testing.T) {
	opts := setupFixture()

	opts.client = &jiraClientMock{failCreate: true}
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender != nil {
		t.Fatal("sender should not have been nil")
	}
	if err == nil {
		t.Fatal("error should not have been nil")
	}
}

func TestCommentConstructorMustPassAuthTest(t *testing.T) {
	opts := setupFixture()

	opts.client = &jiraClientMock{failAuth: true}
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender != nil {
		t.Fatal("sender should not have been nil")
	}
	if err == nil {
		t.Fatal("error should not have been nil")
	}
}

func TestCommentConstructorErrorsWithInvalidConfigs(t *testing.T) {
	sender, err := NewCommentSender(context.Background(), "1234", nil, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender != nil {
		t.Fatal("sender should not have been nil")
	}
	if err == nil {
		t.Fatal("error should not have been nil")
	}

	sender, err = NewIssueSender(context.Background(), &Options{}, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender != nil {
		t.Fatal("sender should not have been nil")
	}
	if err == nil {
		t.Fatal("error should not have been nil")
	}
}

func TestCommentSendMethod(t *testing.T) {
	opts := setupFixture()

	numShouldHaveSent := 0
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}

	m := message.MakeString("sending debug level comment")
	m.SetPriority(level.Debug)
	sender.Send(m)
	if mock.numSent != numShouldHaveSent {
		t.Errorf("%q should be equal to %q", mock.numSent, numShouldHaveSent)
	}

	m = message.MakeString("sending alert level comment")
	m.SetPriority(level.Alert)
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Errorf("%q should be equal to %q", mock.numSent, numShouldHaveSent)
	}

	m = message.MakeString("sending emergency level comment")
	m.SetPriority(level.Emergency)
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Errorf("%q should be equal to %q", mock.numSent, numShouldHaveSent)
	}
}

func TestCommentSendMethodWithError(t *testing.T) {
	opts := setupFixture()

	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}
	if mock.failSend {
		t.Errorf("should have failed")
	}

	m := message.MakeString("test")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%q should be equal to %q", mock.numSent, 1)
	}

	mock.failSend = true
	sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%q should be equal to %q", mock.numSent, 1)
	}
}

func TestCommentCreateMethodChangesClientState(t *testing.T) {
	base := &jiraClientImpl{}
	new := &jiraClientImpl{}

	if base.Client != new.Client && base.Client != nil {
		t.Error("should not")
	}
	if err := new.CreateClient(nil, "foo"); err != nil {
		t.Fatal(err)
	}
	if base.Client == new.Client {
		t.Error("should not be equal")
	}
}

func TestCommentSendWithJiraIssueComposer(t *testing.T) {
	opts := setupFixture()

	c := NewComment(level.Notice, "ABC-123", "Hi")

	sender, err := NewCommentSender(context.Background(), "XYZ-123", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}

	sender.Send(c)

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if 1 != mock.numSent {
		t.Errorf("%q should be equal to %q", 1, mock.numSent)
	}
	if "ABC-123" != mock.lastIssue {
		t.Errorf("%q should be equal to %q", "ABC-123", mock.lastIssue)
	}
}

func TestCommentJiraMessageComposerConstructor(t *testing.T) {
	const testMsg = "hello"
	reporterField := Field{Key: "Reporter", Value: "Annie"}
	assigneeField := Field{Key: "Assignee", Value: "Sejin"}
	typeField := Field{Key: "Type", Value: "Bug"}
	labelsField := Field{Key: "Labels", Value: []string{"Soul", "Pop"}}
	unknownField := Field{Key: "Artist", Value: "Adele"}
	msg := NewIssue("project", testMsg, reporterField, assigneeField, typeField, labelsField, unknownField)
	issue := msg.Raw().(*Issue)

	if "project" != issue.Project {
		t.Error("elements should be equal")
	}
	if testMsg != issue.Summary {
		t.Error("elements should be equal")
	}
	if reporterField.Value != issue.Reporter {
		t.Error("elements should be equal")
	}
	if assigneeField.Value != issue.Assignee {
		t.Error("elements should be equal")
	}
	if typeField.Value != issue.Type {
		t.Error("elements should be equal")
	}
	for idx, elem := range issue.Labels {
		if labelsField.Value.([]string)[idx] != elem {
			t.Error("elements should be equal")
		}
	}
	if unknownField.Value != issue.Fields[unknownField.Key] {
		t.Error("elements should be equal")
	}
}

func TestCommentJiraIssueAnnotationOnlySupportsStrings(t *testing.T) {
	m := &jiraMessage{
		issue: &Issue{},
	}

	if err := m.Annotate("k", 1); err == nil {
		t.Error("error should not be nil")
	}
	if err := m.Annotate("k", true); err == nil {
		t.Error("error should not be nil")
	}
	if err := m.Annotate("k", nil); err == nil {
		t.Error("error should not be nil")
	}
}
