package jira

import (
	"context"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
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
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
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
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
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
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender != nil {
		t.Fatal("expected nil sender")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJiraCommentConstructorErrorsWithInvalidConfigs(t *testing.T) {
	sender, err := NewCommentSender(context.Background(), "1234", nil, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender != nil {
		t.Fatal("expected nil sender")
	}
	if err == nil {
		t.Fatal("expected error")
	}

	sender, err = NewIssueSender(context.Background(), &Options{}, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
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
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("expected sender to be not nil")
	}

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Fatal("expected true")
	}
	if mock.numSent != 0 {
		t.Fatal("expected values to be equal")
	}

	m := message.NewString(level.Debug, "sending debug level comment")
	sender.Send(m)
	if mock.numSent != numShouldHaveSent {
		t.Fatal("expected values to be equal")
	}

	m = message.NewString(level.Alert, "sending alert level comment")
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Fatal("expected values to be equal")
	}

	m = message.NewString(level.Emergency, "sending emergency level comment")
	sender.Send(m)
	numShouldHaveSent++
	if mock.numSent != numShouldHaveSent {
		t.Fatal("expected values to be equal")
	}
}

func TestJiraCommentSendMethodWithError(t *testing.T) {
	opts := setupOptionsFixture()
	sender, err := NewCommentSender(context.Background(), "1234", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
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

	m := message.NewString(level.Alert, "test")
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
	c := NewComment(level.Notice, "ABC-123", "Hi")

	sender, err := NewCommentSender(context.Background(), "XYZ-123", opts, send.LevelInfo{Default: level.Trace, Threshold: level.Info})
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
		t.Fatal("expected values to be equal")
	}
	if "ABC-123" != mock.lastIssue {
		t.Fatal("expected values to be equal")
	}
}
