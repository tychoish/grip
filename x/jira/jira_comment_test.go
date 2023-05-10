package jira

import (
	"context"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func setupOptionsFixture() *Options {
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

func TestJiraCommentMockSenderWithNewConstructor(t *testing.T) {
	opts := setupOptionsFixture()
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
	if sender == nil {
		t.Fatal("expected sender to be not nil")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestJiraCommentConstructorMustCreate(t *testing.T) {
	opts := setupOptionsFixture()
	opts.client = &jiraClientMock{failCreate: true}
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
	if sender != nil {
		t.Fatal("expected nil sender")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJiraCommentConstructorMustPassAuthTest(t *testing.T) {
	opts := setupOptionsFixture()
	opts.client = &jiraClientMock{failAuth: true}
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
	if sender != nil {
		t.Fatal("expected nil sender")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJiraCommentConstructorErrorsWithInvalidConfigs(t *testing.T) {
	sender, err := MakeCommentSender(context.Background(), "1234", nil)
	if sender != nil {
		t.Fatal("expected nil sender")
	}
	if err == nil {
		t.Fatal("expected error")
	}

	sender, err = MakeIssueSender(context.Background(), &Options{})
	if sender != nil {
		t.Fatal("expected nil sender")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJiraCommentSendMethod(t *testing.T) {
	opts := setupOptionsFixture()
	numShouldHaveSent := 0
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("expected sender to be not nil")
	}
	sender.SetPriority(level.Info)

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Fatal("expected true")
	}
	if mock.numSent != 0 {
		t.Fatal("expected values to be equal")
	}

	m := message.MakeString("sending debug level comment")
	m.SetPriority(level.Debug)
	sender.Send(m)
	if mock.numSent != numShouldHaveSent {
		t.Fatal("expected values to be equal")
	}

	m = message.MakeString("sending alert level comment")
	m.SetPriority(level.Alert)
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Fatal("expected values to be equal")
	}

	m = message.MakeString("sending emergency level comment")
	m.SetPriority(level.Emergency)
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Fatal("expected values to be equal")
	}
}

func TestJiraCommentSendMethodWithError(t *testing.T) {
	opts := setupOptionsFixture()
	sender, err := MakeCommentSender(context.Background(), "1234", opts)
	if sender == nil {
		t.Fatal("expected sender to be not nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Fatal("expected true")
	}
	if mock.numSent != 0 {
		t.Fatal("expected values to be equal")
	}
	if mock.failSend {
		t.Fatal("expected false")
	}

	m := message.MakeString("test")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numSent != 1 {
		t.Fatal("expected values to be equal")
	}

	mock.failSend = true
	sender.Send(m)
	if mock.numSent != 1 {
		t.Fatal("expected values to be equal")
	}
}

func TestJiraCommentCreateMethodChangesClientState(t *testing.T) {
	base := &jiraClientImpl{}
	new := &jiraClientImpl{}

	if err := new.CreateClient(nil, "foo"); err != nil {
		t.Fatal(err)
	}
	if base.Client == new.Client {
		t.Fatal("clients should not be equal")
	}
}

func TestJiraCommentSendWithJiraIssueComposer(t *testing.T) {
	opts := setupOptionsFixture()
	c := MakeComment("ABC-123", "Hi")
	c.SetPriority(level.Warning)
	sender, err := MakeCommentSender(context.Background(), "XYZ-123", opts)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("expected sender to be not nil")
	}

	sender.Send(c)

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Fatal("expected true")
	}
	if 1 != mock.numSent {
		t.Error("expected values to be equal", mock.numSent)
	}
	if "ABC-123" != mock.lastIssue {
		t.Error("expected values to be equal", mock.lastIssue)
	}
}
