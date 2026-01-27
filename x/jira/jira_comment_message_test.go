package jira

import (
	"context"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
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

	sender, err := MakeCommentSender(context.Background(), "1234", opts)
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
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
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
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
	if sender != nil {
		t.Fatal("sender should not have been nil")
	}
	if err == nil {
		t.Fatal("error should not have been nil")
	}
}

func TestCommentConstructorErrorsWithInvalidConfigs(t *testing.T) {
	sender, err := MakeCommentSender(context.Background(), "1234", nil)
	if sender != nil {
		t.Fatal("sender should not have been nil")
	}
	if err == nil {
		t.Fatal("error should not have been nil")
	}

	sender, err = MakeIssueSender(context.Background(), &Options{})
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
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
	if err != nil {
		t.Fatal(err)
	}
	sender.SetPriority(level.Info)
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}

	m := message.MakeString("sending debug level comment")
	m.SetPriority(level.Debug)
	sender.Send(m)
	if mock.numSent != numShouldHaveSent {
		t.Errorf("%v should be equal to %v", mock.numSent, numShouldHaveSent)
	}

	m = message.MakeString("sending alert level comment")
	m.SetPriority(level.Alert)
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Errorf("%v should be equal to %v", mock.numSent, numShouldHaveSent)
	}

	m = message.MakeString("sending emergency level comment")
	m.SetPriority(level.Emergency)
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Errorf("%v should be equal to %v", mock.numSent, numShouldHaveSent)
	}
}

func TestCommentSendMethodWithError(t *testing.T) {
	opts := setupFixture()

	sender, err := MakeCommentSender(context.Background(), "1234", opts)
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
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}
	if mock.failSend {
		t.Errorf("should have failed")
	}

	m := message.MakeString("test")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%v should be equal to %v", mock.numSent, 1)
	}

	mock.failSend = true
	sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%v should be equal to %v", mock.numSent, 1)
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

	c := MakeComment("ABC-123", "Hi")
	c.SetPriority(level.Warning)
	sender, err := MakeCommentSender(context.Background(), "XYZ-123", opts)
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
		t.Errorf("%v should be equal to %v", 1, mock.numSent)
	}
	if "ABC-123" != mock.lastIssue {
		t.Errorf("%v should be equal to %v", "ABC-123", mock.lastIssue)
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

	m.Annotate("k", 1)
	m.Annotate("k", true)
	m.Annotate("k", nil)
	if m.issue.Fields == nil {
		t.Fatal("message context should be non-nil")
	}
	if len(m.issue.Fields) != 1 {
		t.Error(m.Base.Context.Len())
	}
}
