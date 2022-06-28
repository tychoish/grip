package slack

import (
	"os"
	"strings"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func setupFixture() *SlackOptions {
	return &SlackOptions{
		Channel:  "#test",
		Hostname: "testhost",
		Name:     "bot",
		client:   &slackClientMock{},
	}
}

func TestMakeSlackConstructorErrorsWithUnsetEnvVar(t *testing.T) {
	sender, err := MakeSender(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if sender != nil {
		t.Fatal("sender expected to be nil")
	}

	sender, err = MakeSender(&SlackOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if sender != nil {
		t.Fatal("sender expected to be nil")
	}

	sender, err = MakeSender(&SlackOptions{Channel: "#meta"})
	if err == nil {
		t.Fatal("expected error")
	}
	if sender != nil {
		t.Fatal("sender expected to be nil")
	}
}

func TestMakeSlackConstructorErrorsWithInvalidConfigs(t *testing.T) {
	defer os.Setenv(slackClientToken, os.Getenv(slackClientToken))
	if err := os.Setenv(slackClientToken, "foo"); err != nil {
		t.Fatal(err)
	}

	sender, err := MakeSender(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if sender != nil {
		t.Fatal("sender expected to be nil")
	}

	sender, err = MakeSender(&SlackOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if sender != nil {
		t.Fatal("sender expected to be nil")
	}
}

func TestValidateAndConstructoRequiresValidate(t *testing.T) {
	opts := &SlackOptions{}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error")
	}

	opts.Hostname = "testsystem.com"
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error")
	}

	opts.Name = "test"
	opts.Channel = "$chat"
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error")
	}
	opts.Channel = "@test"
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	opts.Channel = "#test"
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}

	defer os.Setenv(slackClientToken, os.Getenv(slackClientToken))
	if err := os.Setenv(slackClientToken, "foo"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRequiresOctothorpOrArobase(t *testing.T) {
	opts := &SlackOptions{Name: "test", Channel: "#chat", Hostname: "foo"}
	if opts.Channel != "#chat" {
		t.Fatalf("expected 'opts.Channel' to be '#chat' but was %s", opts.Channel)
	}
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	opts = &SlackOptions{Name: "test", Channel: "@chat", Hostname: "foo"}
	if opts.Channel != "@chat" {
		t.Fatalf("expected 'opts.Channel' to be '@chat' but was %s", opts.Channel)
	}
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFieldSetIncludeCheck(t *testing.T) {
	opts := &SlackOptions{}
	if opts.FieldsSet != nil {
		t.Fatal("expected field set to be nil")
	}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error")
	}
	if opts.FieldsSet == nil {
		t.Fatal("expected field set to be non-nil")
	}

	if opts.fieldSetShouldInclude("time") {
		t.Fatal("expected false")
	}
	opts.FieldsSet["time"] = true
	if opts.fieldSetShouldInclude("time") {
		t.Fatal("expected false")
	}

	if opts.fieldSetShouldInclude("msg") {
		t.Fatal("expected false")
	}
	opts.FieldsSet["time"] = true
	if opts.fieldSetShouldInclude("msg") {
		t.Fatal("expected false")
	}

	for _, f := range []string{"a", "b", "c"} {
		if opts.fieldSetShouldInclude(f) {
			t.Fatal("expected false")
		}
		opts.FieldsSet[f] = true
		if !opts.fieldSetShouldInclude(f) {
			t.Fatal("expected to be true")
		}
	}
}

func TestFieldShouldIncludIsAlwaysTrueWhenFieldSetIsNile(t *testing.T) {
	opts := &SlackOptions{}
	if opts.FieldsSet != nil {
		t.Fatal("expected field set to be nil")
	}
	if opts.fieldSetShouldInclude("time") {
		t.Fatal("expected false")
	}
	for _, f := range []string{"a", "b", "c"} {
		if !opts.fieldSetShouldInclude(f) {
			t.Fatal("expected to be true")
		}
	}
}

func TestGetParamsWithAttachementOptsDisabledLevelImpact(t *testing.T) {
	opts := &SlackOptions{}
	if opts.Fields {
		t.Fatal("expected false")
	}
	if opts.BasicMetadata {
		t.Fatal("expected false")
	}

	msg, params := opts.produceMessage(message.MakeString("foo"))
	if params.Attachments[0].Color != "good" {
		t.Fatalf("expected 'params.Attachments[0].Color' to be 'good' but was %s", params.Attachments[0].Color)
	}
	if msg != "foo" {
		t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
	}

	for _, l := range []level.Priority{level.Emergency, level.Alert, level.Critical} {
		msg, params = opts.produceMessage(message.NewString(l, "foo"))
		if params.Attachments[0].Color != "danger" {
			t.Fatalf("expected 'params.Attachments[0].Color' to be 'danger' but was %s", params.Attachments[0].Color)
		}
		if msg != "foo" {
			t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
		}
	}

	for _, l := range []level.Priority{level.Warning, level.Notice} {
		msg, params = opts.produceMessage(message.NewString(l, "foo"))
		if params.Attachments[0].Color != "warning" {
			t.Fatalf("expected 'params.Attachments[0].Color' to be 'warning' but was %s", params.Attachments[0].Color)
		}
		if msg != "foo" {
			t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
		}
	}

	for _, l := range []level.Priority{level.Debug, level.Info, level.Trace} {
		msg, params = opts.produceMessage(message.NewString(l, "foo"))
		if params.Attachments[0].Color != "good" {
			t.Fatalf("expected 'params.Attachments[0].Color' to be 'good' but was %s", params.Attachments[0].Color)
		}
		if msg != "foo" {
			t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
		}
	}
}

func TestProduceMessageWithBasicMetaDataEnabled(t *testing.T) {
	opts := &SlackOptions{BasicMetadata: true}
	if opts.Fields {
		t.Fatal("expected false")
	}
	if !opts.BasicMetadata {
		t.Fatal("expected to be true")
	}

	msg, params := opts.produceMessage(message.NewString(level.Alert, "foo"))
	if params.Attachments[0].Color != "danger" {
		t.Fatalf("expected 'params.Attachments[0].Color' to be 'danger' but was %s", params.Attachments[0].Color)
	}
	if msg != "foo" {
		t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 1 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 1 but was %d", len(params.Attachments[0].Fields))
	}
	if !strings.Contains(params.Attachments[0].Fallback, "priority=alert") {
		t.Fatal("expected to be true")
	}

	opts.Hostname = "!"
	msg, params = opts.produceMessage(message.NewString(level.Alert, "foo"))
	if msg != "foo" {
		t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 1 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 1 but was %d", len(params.Attachments[0].Fields))
	}
	if strings.Contains(params.Attachments[0].Fallback, "host") {
		t.Fatal("expected false")
	}

	opts.Hostname = "foo"
	msg, params = opts.produceMessage(message.NewString(level.Alert, "foo"))
	if msg != "foo" {
		t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 2 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 2 but was %d", len(params.Attachments[0].Fields))
	}
	if !strings.Contains(params.Attachments[0].Fallback, "host=foo") {
		t.Fatal("expected to be true")
	}

	opts.Name = "foo"
	msg, params = opts.produceMessage(message.NewString(level.Alert, "foo"))
	if msg != "foo" {
		t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 3 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 3 but was %d", len(params.Attachments[0].Fields))
	}
	if !strings.Contains(params.Attachments[0].Fallback, "journal=foo") {
		t.Fatal("expected to be true")
	}
}

func TestFieldsMessageTypeIntegration(t *testing.T) {
	opts := &SlackOptions{Fields: true}
	if !opts.Fields {
		t.Fatal("expected to be true")
	}
	if opts.BasicMetadata {
		t.Fatal("expected false")
	}
	opts.FieldsSet = map[string]bool{
		"message": true,
		"other":   true,
		"foo":     true,
	}

	msg, params := opts.produceMessage(message.NewString(level.Alert, "foo"))
	if msg != "foo" {
		t.Fatalf("expected 'msg' to be 'foo' but was %s", msg)
	}
	if params.Attachments[0].Color != "danger" {
		t.Fatalf("expected 'params.Attachments[0].Color' to be 'danger' but was %s", params.Attachments[0].Color)
	}
	if len(params.Attachments[0].Fields) != 0 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 0 but was %d", len(params.Attachments[0].Fields))
	}

	// if the fields are nil, then we end up ignoring things, except the message
	msg, params = opts.produceMessage(message.NewAnnotated(level.Alert, "foo", message.Fields{}))
	if msg != "" {
		t.Fatalf("expected 'msg' to be '' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 1 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 1 but was %d", len(params.Attachments[0].Fields))
	}

	// when msg and the message match we ignore
	msg, params = opts.produceMessage(message.NewAnnotated(level.Alert, "foo", message.Fields{"msg": "foo"}))
	if msg != "" {
		t.Fatalf("expected 'msg' to be '' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 1 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 1 but was %d", len(params.Attachments[0].Fields))
	}

	msg, params = opts.produceMessage(message.NewAnnotated(level.Alert, "foo", message.Fields{"foo": "bar"}))
	if msg != "" {
		t.Fatalf("expected 'msg' to be '' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 2 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 2 but was %d", len(params.Attachments[0].Fields))
	}

	msg, params = opts.produceMessage(message.NewAnnotated(level.Alert, "foo", message.Fields{"other": "baz"}))
	if msg != "" {
		t.Fatalf("expected 'msg' to be '' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 2 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 2 but was %d", len(params.Attachments[0].Fields))
	}

	msg, params = opts.produceMessage(message.NewAnnotated(level.Alert, "foo", message.Fields{"untracked": "false", "other": "bar"}))
	if msg != "" {
		t.Fatalf("expected 'msg' to be '' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 2 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 2 but was %d", len(params.Attachments[0].Fields))
	}

	msg, params = opts.produceMessage(message.NewAnnotated(level.Alert, "foo", message.Fields{"foo": "false", "other": "bass"}))
	if msg != "" {
		t.Fatalf("expected 'msg' to be '' but was %s", msg)
	}
	if len(params.Attachments[0].Fields) != 3 {
		t.Fatalf("expected length of 'params.Attachments[0].Fields' to be 3 but was %d", len(params.Attachments[0].Fields))
	}
}

func TestMockSenderWithMakeConstructor(t *testing.T) {
	opts := setupFixture()
	defer os.Setenv(slackClientToken, os.Getenv(slackClientToken))
	if err := os.Setenv(slackClientToken, "foo"); err != nil {
		t.Fatal(err)
	}

	sender, err := MakeSender(opts)
	if sender == nil {
		t.Fatal("sender not expected to be nil")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestMockSenderWithNewConstructor(t *testing.T) {
	opts := setupFixture()
	sender, err := NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender == nil {
		t.Fatal("sender not expected to be nil")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvaldLevelCausesConstructionErrors(t *testing.T) {
	opts := setupFixture()
	sender, err := NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Invalid})
	if sender != nil {
		t.Fatal("sender expected to be nil")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConstructorMustPassAuthTest(t *testing.T) {
	opts := setupFixture()
	opts.client = &slackClientMock{failAuthTest: true}
	sender, err := NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Info})

	if sender != nil {
		t.Fatal("sender expected to be nil")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendMethod(t *testing.T) {
	opts := setupFixture()
	sender, err := NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender == nil {
		t.Fatal("sender not expected to be nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	mock, ok := opts.client.(*slackClientMock)
	if !ok {
		t.Fatal("expected to be true")
	}
	if 0 != mock.numSent {
		t.Fatalf("expected '0' to be 'mock.numSent' but was %d", 0)
	}

	m := message.NewString(level.Debug, "hello")
	sender.Send(m)
	if 0 != mock.numSent {
		t.Fatalf("expected '0' to be 'mock.numSent' but was %d", 0)
	}

	m = message.NewString(level.Alert, "")
	sender.Send(m)
	if 0 != mock.numSent {
		t.Fatalf("expected '0' to be 'mock.numSent' but was %d", 0)
	}

	m = message.NewString(level.Alert, "world")
	sender.Send(m)
	if 1 != mock.numSent {
		t.Fatalf("expected '1' to be 'mock.numSent' but was %d", 1)
	}
	if mock.lastTarget != "#test" {
		t.Fatalf("expected 'mock.lastTarget' to be '#test' but was %s", mock.lastTarget)
	}

	m = NewMessage(level.Alert, "#somewhere", "Hi", nil)
	sender.Send(m)
	if 2 != mock.numSent {
		t.Fatalf("expected '2' to be 'mock.numSent' but was %d", 2)
	}
	if mock.lastTarget != "#somewhere" {
		t.Fatalf("expected 'mock.lastTarget' to be '#somewhere' but was %s", mock.lastTarget)
	}
}

func TestSendMethodWithError(t *testing.T) {
	opts := setupFixture()
	sender, err := NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender == nil {
		t.Fatal("sender not expected to be nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	mock, ok := opts.client.(*slackClientMock)
	if !ok {
		t.Fatal("expected to be true")
	}
	if 0 != mock.numSent {
		t.Fatalf("expected 'mock.numSent' to be 0, but was %d", mock.numSent)
	}
	if mock.failSendingMessage {
		t.Fatal("expected false")
	}

	m := message.NewString(level.Alert, "world")
	sender.Send(m)
	if 1 != mock.numSent {
		t.Fatalf("expected '1' to be 'mock.numSent' but was %d", mock.numSent)
	}

	mock.failSendingMessage = true
	sender.Send(m)
	if 1 != mock.numSent {
		t.Fatalf("expected '1' to be 'mock.numSent' but was %d", mock.numSent)
	}

	// sender should not panic with empty attachments
	func() {
		m = NewMessage(level.Alert, "#general", "I am a formatted slack message", nil)
		sender.Send(m)
		if 1 != mock.numSent {
			t.Fatalf("expected '1' to be 'mock.numSent' but was %d", mock.numSent)
		}
	}()
}

func TestCreateMethodChangesClientState(t *testing.T) {
	base := &slackClientImpl{}
	new := &slackClientImpl{}

	if new.Slack != base.Slack {
		t.Fatal("expected 'new' and 'base' to be equal")
	}
	new.Create("foo")
	if new.Slack == base.Slack {
		t.Fatal("expected new to not be equal to base")
	}
}

func TestSendMethodDoesIncorrectlyAllowTooLowMessages(t *testing.T) {
	opts := setupFixture()
	sender, err := NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if sender == nil {
		t.Fatal("sender not expected to be nil")
	}
	if err != nil {
		t.Fatal(err)
	}

	mock, ok := opts.client.(*slackClientMock)
	if !ok {
		t.Fatal("expected to be true")
	}
	if 0 != mock.numSent {
		t.Fatalf("expected '0' to be 'mock.numSent' but was %d", mock.numSent)
	}

	if err := sender.SetLevel(send.LevelInfo{Default: level.Critical, Threshold: level.Alert}); err != nil {
		t.Fatal(err)
	}
	if 0 != mock.numSent {
		t.Fatalf("expected '0' to be 'mock.numSent' but was %d", mock.numSent)
	}
	sender.Send(message.NewString(level.Info, "hello"))
	if 0 != mock.numSent {
		t.Fatalf("expected '0' to be 'mock.numSent' but was %d", mock.numSent)
	}
	sender.Send(message.NewString(level.Alert, "hello"))
	if 1 != mock.numSent {
		t.Fatalf("expected '1' to be 'mock.numSent' but was %d", mock.numSent)
	}
	sender.Send(message.NewString(level.Alert, "hello"))
	if 2 != mock.numSent {
		t.Fatalf("expected '2' to be 'mock.numSent' but was %d", mock.numSent)
	}
}

func TestSettingBotIdentity(t *testing.T) {
	opts := setupFixture()
	sender, err := NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("sender not expected to be nil")
	}

	mock, ok := opts.client.(*slackClientMock)
	if !ok {
		t.Fatal("expected to be true")
	}
	if 0 != mock.numSent {
		t.Fatalf("expected '0' to be 'mock.numSent' but was %d", mock.numSent)
	}
	if mock.failSendingMessage {
		t.Fatal("expected false")
	}

	m := message.NewString(level.Alert, "world")
	sender.Send(m)
	if mock.numSent != 1 {
		t.Fatalf("expected 'mock.numSent' to be '1' but was %d", mock.numSent)
	}

	if len(mock.lastMsg.Username) != 0 {
		t.Fatal("expected 'mock.lastMsg.Username' to be empty")
	}
	if len(mock.lastMsg.IconUrl) != 0 {
		t.Fatal("expected 'mock.lastMsg.IconUrl' to be empty")
	}

	opts.Username = "Grip"
	opts.IconURL = "https://example.com/icon.ico"
	sender, err = NewSender(opts, "foo", send.LevelInfo{Default: level.Trace, Threshold: level.Info})
	if err != nil {
		t.Fatal(err)
	}
	sender.Send(m)
	if mock.numSent != 2 {
		t.Fatalf("expected 'mock.numSent' to be '2' but was %d", mock.numSent)
	}
	if mock.lastMsg.Username != "Grip" {
		t.Fatalf("expected 'mock.lastMsg.Username' to be 'Grip' but was %s", mock.lastMsg.Username)
	}
	if mock.lastMsg.IconUrl != "https://example.com/icon.ico" {
		t.Fatalf("expected 'mock.lastMsg.IconUrl' to be 'https://example.com/icon.ico' but was %s", mock.lastMsg.IconUrl)
	}
	if mock.lastMsg.AsUser {
		t.Fatal("expected false")
	}
}
