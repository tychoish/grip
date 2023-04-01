package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestMockSenderWithNewConstructor(t *testing.T) {
	opts := setupFixture()
	sender, err := MakeIssueSender(context.Background(), opts)
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestConstructorMustCreate(t *testing.T) {
	opts := setupFixture()
	opts.client = &jiraClientMock{failCreate: true}
	sender, err := MakeIssueSender(context.Background(), opts)

	if sender != nil {
		t.Fatal("expected nil, but got value", sender)
	}
	if err == nil {
		t.Error("expected an error but got nil")
	}
}

func TestConstructorMustPassAuthTest(t *testing.T) {
	opts := setupFixture()
	opts.client = &jiraClientMock{failAuth: true}
	sender, err := MakeIssueSender(context.Background(), opts)

	if sender != nil {
		t.Fatal("expected nil, but got value", sender)
	}
	if err == nil {
		t.Error("expected an error but got nil")
	}
}

func TestConstructorErrorsWithInvalidConfigs(t *testing.T) {
	sender, err := MakeIssueSender(context.Background(), nil)

	if sender != nil {
		t.Fatal("expected nil, but got value", sender)
	}
	if err == nil {
		t.Error("expected an error but got nil")
	}

	sender, err = MakeIssueSender(context.Background(), &Options{})
	if sender != nil {
		t.Fatal("expected nil, but got value", sender)
	}
	if err == nil {
		t.Error("expected an error but got nil")
	}

	opts := &Options{
		BasicAuthOpts: BasicAuth{
			Username: "foo",
			Password: "bar",
		},
		Oauth1Opts: Oauth1{
			AccessToken: "12345",
		},
	}

	sender, err = MakeIssueSender(context.Background(), opts)
	if err == nil {
		t.Fatal("error must not be nil")
	}
	if err.Error() != "must specify exactly 1 method of authentication" {
		t.Fatal("error was not expected:", err)
	}
	if sender != nil {
		t.Fatal("expected nil, but got value", sender)
	}
}

func TestSendMethod(t *testing.T) {
	opts := setupFixture()
	sender, err := MakeIssueSender(context.Background(), opts)
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	sender.SetPriority(level.Info)

	mock, ok := opts.client.(*jiraClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}

	m := message.MakeString("hello")
	m.SetPriority(level.Debug)
	sender.Send(m)
	if mock.numSent != 0 {
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}

	m = message.MakeString("")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numSent != 0 {
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}

	m = message.MakeString("world")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%v should be equal to %v", mock.numSent, 1)
	}
}

func TestSendMethodWithError(t *testing.T) {
	opts := setupFixture()
	sender, err := MakeIssueSender(context.Background(), opts)
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
		t.Fatal("messsage should have failed to send")
	}

	m := message.MakeString("world")
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

// Test get fields
func TestGetFieldsWithJiraIssue(t *testing.T) {
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

	if fields.Project.Key != project {
		t.Errorf("%v should be equal to %v", fields.Project.Key, project)
	}
	if fields.Summary != summary {
		t.Errorf("%v should be equal to %v", fields.Summary, summary)
	}
	if fields.Reporter != nil {
		t.Fatal("expected nil, but got value", fields.Reporter)
	}
	if fields.Assignee != nil {
		t.Fatal("expected nil, but got value", fields.Assignee)
	}
	if fields.Type.Name != "Task" {
		t.Errorf("%v should be equal to %v", fields.Type.Name, "Task")
	}
	if fields.Labels != nil {
		t.Fatal("expected nil, but got value", fields.Labels)
	}
	if fields.Unknowns != nil {
		t.Fatal("expected nil, but got value", fields.Unknowns)
	}

	// Test Two: with reporter, assignee and type
	m2 := NewIssue(project, summary, reporterField, assigneeField,
		typeField, labelsField)
	fields = getFields(m2)

	if fields.Reporter.Name != "Annie" {
		t.Errorf("%v should be equal to %v", fields.Reporter.Name, "Annie")
	}
	if fields.Assignee.Name != "Sejin" {
		t.Errorf("%v should be equal to %v", fields.Assignee.Name, "Sejin")
	}
	if fields.Type.Name != "Bug" {
		t.Errorf("%v should be equal to %v", fields.Type.Name, "Bug")
	}
	expected := []string{"Soul", "Pop"}
	for idx := range fields.Labels {
		if fields.Labels[idx] != expected[idx] {
			t.Error("inequality at index", idx)
		}
	}
	if fields.Unknowns != nil {
		t.Error("expected unknowns to be nil")
	}

	// Test Three: everything plus Unknown fields
	m3 := NewIssue(project, summary, reporterField, assigneeField,
		typeField, unknownField)
	fields = getFields(m3)
	if fields.Unknowns["Artist"] != "Adele" {
		t.Errorf("%v should be equal to %v", fields.Unknowns["Artist"], "Adele")
	}
}

func TestGetFieldsWithFields(t *testing.T) {
	msg := "Get the message"
	testFields := message.Fields{"key0": 12, "key1": 42, "message": msg}
	m := message.MakeFields(testFields)

	fields := getFields(m)
	if fields.Summary != msg {
		t.Errorf("%v should be equal to %v", fields.Summary, msg)
	}
	if fields.Description == "" {
		t.Error("fields.Description should be nil")
	}
}

func TestTruncate(t *testing.T) {
	opts := setupFixture()
	sender, err := MakeIssueSender(context.Background(), opts)
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

	m := message.MakeString("aaa")
	m.SetPriority(level.Info)
	if !ok {
		t.Error("shoud not have been false")
	}
	sender.Send(m)
	if len(mock.lastSummary) != 3 {
		t.Error("expected value was not of the correct length:", mock.lastSummary)
	}

	var longString bytes.Buffer
	for i := 0; i < 1000; i++ {
		longString.WriteString("a")
	}
	m = message.MakeString(longString.String())
	m.SetPriority(level.Info)
	if !ok {
		t.Error("shoud not have been false")
	}
	sender.Send(m)
	if len(mock.lastSummary) != 254 {
		t.Error("expected value was not of the correct length:", mock.lastSummary)
	}

	buffer := bytes.NewBufferString("")
	buffer.Grow(40000)
	for i := 0; i < 40000; i++ {
		buffer.WriteString("a")
	}

	m = message.MakeString(buffer.String())
	m.SetPriority(level.Info)
	if !ok {
		t.Error("shoud not have been false")
	}
	sender.Send(m)
	if len(mock.lastDescription) != 32767 {
		t.Error("expected value was not of the correct length:", mock.lastDescription)
	}

}

func TestCustomFields(t *testing.T) {
	opts := setupFixture()
	sender, err := MakeIssueSender(context.Background(), opts)
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
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}

	jiraIssue := &Issue{
		Summary: "test",
		Type:    "type",
		Fields: map[string]any{
			"customfield_12345": []string{"hi", "bye"},
		},
	}

	m := MakeIssue(jiraIssue)
	m.SetPriority(level.Warning)

	sender.Send(m)

	expected := []string{"hi", "bye"}
	values := mock.lastFields.Unknowns["customfield_12345"].([]string)
	for idx := range values {
		if values[idx] != expected[idx] {
			t.Error("inequality at index", idx)
		}
	}

	if "test" != mock.lastFields.Summary {
		t.Errorf("%v should be equal to %v", "test", mock.lastFields.Summary)
	}

	bytes, err := json.Marshal(&mock.lastFields)
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes) != 79 {
		t.Error("marshaled value has unexpected length")
	}
	if `{"customfield_12345":["hi","bye"],"issuetype":{"name":"type"},"summary":"test"}` != string(bytes) {
		t.Errorf("%v should be equal to %v", `{"customfield_12345":["hi","bye"],"issuetype":{"name":"type"},"summary":"test"}`, string(bytes))
	}
}

func TestPopulateKey(t *testing.T) {
	opts := setupFixture()
	sender, err := MakeIssueSender(context.Background(), opts)
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
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}

	count := 0
	jiraIssue := &Issue{
		Summary: "foo",
		Type:    "bug",
		Callback: func(_ string) {
			count++
		},
	}

	if 0 != count {
		t.Errorf("%v should be equal to %v", 0, count)
	}
	m := MakeIssue(jiraIssue)
	m.SetPriority(level.Alert)
	sender.Send(m)
	if 1 != count {
		t.Errorf("%v should be equal to %v", 1, count)
	}
	issue := m.Raw().(*Issue)
	if mock.issueKey != issue.IssueKey {
		t.Errorf("%v should be equal to %v", mock.issueKey, issue.IssueKey)
	}

	messageFields := message.MakeFields(message.Fields{
		"msg":     "something",
		"message": "foo",
	})

	messageFields.SetPriority(level.Info)
	sender.Send(messageFields)
	messageIssue := messageFields.Raw().(message.Fields)
	if mock.issueKey != messageIssue[jiraIssueKey] {
		t.Errorf("%v should be equal to %v", mock.issueKey, messageIssue[jiraIssueKey])
	}
}

func TestWhenCallbackNil(t *testing.T) {
	opts := setupFixture()
	sender, err := MakeIssueSender(context.Background(), opts)
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
		t.Errorf("%v should be equal to %v", mock.numSent, 0)
	}

	jiraIssue := &Issue{
		Summary: "foo",
		Type:    "bug",
	}

	m := MakeIssue(jiraIssue)
	m.SetPriority(level.Alert)
	func() {
		// should not panic
		sender.Send(m)
	}()
}
