package email

import (
	"net/mail"
	"runtime"
	"strings"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func setupFixture(t *testing.T) *SMTPOptions {
	t.Helper()
	opts := &SMTPOptions{
		client:        &smtpClientMock{},
		Subject:       "test email from logger",
		NameAsSubject: true,
		Name:          "test smtp sender",
		toAddrs: []*mail.Address{
			{
				Name:    "one",
				Address: "two",
			},
		},
	}
	if opts.GetContents != nil {
		t.Fatal("contents not nil on init")
	}
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	if opts.GetContents == nil {
		t.Fatal("contents nil after validate")
	}
	return opts
}

func TestOptionsMustBeIValid(t *testing.T) {
	invalidOpts := []*SMTPOptions{
		{},
		{
			Subject:          "too many subject uses",
			NameAsSubject:    true,
			MessageAsSubject: true,
		},
		{
			Subject: "missing name",
			toAddrs: []*mail.Address{
				{
					Name:    "one",
					Address: "two",
				},
			},
		},
		{
			Subject: "empty",
			Name:    "sender",
			toAddrs: []*mail.Address{},
		},
	}

	for _, opts := range invalidOpts {
		if err := opts.Validate(); err == nil {
			t.Fatal("error shuld not be nil")
		}
	}
}

func TestDefaultGetContents(t *testing.T) {
	opts := setupFixture(t)

	m := message.MakeString("helllooooo!")
	sbj, msg := opts.GetContents(opts, m)

	if !opts.NameAsSubject {
		t.Fatal("'opts.NameAsSubject' should be true")
	}
	if sbj != opts.Name {
		t.Fatal("values should be equal")
	}
	if msg != m.String() {
		t.Fatal("values should be equal")
	}

	opts.NameAsSubject = false
	sbj, _ = opts.GetContents(opts, m)
	if sbj != opts.Subject {
		t.Fatal("values should be equal")
	}

	opts.MessageAsSubject = true
	sbj, msg = opts.GetContents(opts, m)
	if msg != "" {
		t.Fatal("values should be equal")
	}
	if sbj != m.String() {
		t.Fatal("values should be equal")
	}
	opts.MessageAsSubject = false

	opts.Subject = ""
	sbj, msg = opts.GetContents(opts, m)
	if sbj != "" {
		t.Fatal("values should be equal")
	}
	if msg != m.String() {
		t.Fatal("values should be equal")
	}
	opts.Subject = "test email subject"

	opts.TruncatedMessageSubjectLength = len(m.String()) * 2
	sbj, msg = opts.GetContents(opts, m)
	if msg != m.String() {
		t.Fatal("values should be equal")
	}
	if sbj != m.String() {
		t.Fatal("values should be equal")
	}

	opts.TruncatedMessageSubjectLength = len(m.String()) - 2
	sbj, msg = opts.GetContents(opts, m)
	if msg != m.String() {
		t.Fatal("values should be equal")
	}
	if msg == sbj {
		t.Fatal("values should not be equal")
	}
	if len(msg) < len(sbj) {
		t.Fatal("'len(msg) > len(sbj)' should be true")
	}
}

func TestResetRecips(t *testing.T) {
	opts := setupFixture(t)
	if len(opts.toAddrs) == 0 {
		t.Fatal("'len(opts.toAddrs) > 0' should be true")
	}
	opts.ResetRecipients()
	if l := len(opts.toAddrs); l != 0 {
		t.Fatalf("length of opts.toAddrs should be %d", 0)
	}
}

func TestAddRecipientsFailsWithNoArgs(t *testing.T) {
	opts := setupFixture(t)
	opts.ResetRecipients()
	if err := opts.AddRecipients(); err == nil {
		t.Fatal("error shuld not be nil")
	}
	if l := len(opts.toAddrs); l != 0 {
		t.Fatalf("length of opts.toAddrs should be %d", 0)
	}
}

func TestAddRecipientsErrorsWithInvalidAddresses(t *testing.T) {
	opts := setupFixture(t)
	opts.ResetRecipients()
	if err := opts.AddRecipients("foo", "bar", "baz"); err == nil {
		t.Fatal("error shuld not be nil")
	}
	if l := len(opts.toAddrs); l != 0 {
		t.Fatalf("length of opts.toAddrs should be %d", 0)
	}
}

func TestAddingMultipleRecipients(t *testing.T) {
	opts := setupFixture(t)
	opts.ResetRecipients()

	if err := opts.AddRecipients("test <one@example.net>"); err != nil {
		t.Fatal(err)
	}
	if l := len(opts.toAddrs); l != 1 {
		t.Fatalf("length of opts.toAddrs should be %d", 1)
	}
	if err := opts.AddRecipients("test <one@example.net>", "test2 <two@example.net>"); err != nil {
		t.Fatal(err)
	}
	if l := len(opts.toAddrs); l != 3 {
		t.Fatalf("length of opts.toAddrs should be %d", 3)
	}
}

func TestAddingSingleRecipientWithInvalidAddressErrors(t *testing.T) {
	opts := setupFixture(t)
	opts.ResetRecipients()
	if err := opts.AddRecipient("test", "address"); err == nil {
		t.Fatal("error shuld not be nil")
	}
	if l := len(opts.toAddrs); l != 0 {
		t.Fatalf("length of opts.toAddrs should be %d", 0)
	}

	if runtime.Compiler != "gccgo" {
		// this panics on gccgo1.4, but is generally an interesting test.
		// not worth digging into a standard library bug that
		// seems fixed on gcgo. and/or in a more recent version.
		if err := opts.AddRecipient("test", "address"); err == nil {
			t.Fatal("error shuld not be nil")
		}
		if l := len(opts.toAddrs); l != 0 {
			t.Fatalf("length of opts.toAddrs should be %d", 0)
		}
	}
}

func TestAddingSingleRecipient(t *testing.T) {
	opts := setupFixture(t)
	opts.ResetRecipients()
	if err := opts.AddRecipient("test", "one@example.net"); err != nil {
		t.Fatal(err)
	}
	if l := len(opts.toAddrs); l != 1 {
		t.Fatalf("length of opts.toAddrs should be %d", 1)
	}
}

func TestMakeConstructorFailureCases(t *testing.T) {
	sender, err := MakeSender(nil)
	if sender != nil {
		t.Fatal("'sender' is expected to be nil")
	}
	if err == nil {
		t.Fatal("error shold not be nil")
	}

	sender, err = MakeSender(&SMTPOptions{})
	if sender != nil {
		t.Fatal("'sender' is expected to be nil")
	}
	if err == nil {
		t.Fatal("error shold not be nil")
	}
}

func TestSendMailErrorsIfNoAddresses(t *testing.T) {
	opts := setupFixture(t)
	opts.ResetRecipients()
	if l := len(opts.toAddrs); l != 0 {
		t.Fatalf("length of opts.toAddrs should be %d", 0)
	}

	m := message.MakeString("hello world!")
	if err := opts.sendMail(m); err == nil {
		t.Fatal("error shuld not be nil")
	}
}

func TestSendMailErrorsIfMailCallFails(t *testing.T) {
	opts := setupFixture(t)
	opts.client = &smtpClientMock{
		failMail: true,
	}

	m := message.MakeString("hello world!")
	if err := opts.sendMail(m); err == nil {
		t.Fatal("error shuld not be nil")
	}
}

func TestSendMailErrorsIfRecptFails(t *testing.T) {
	opts := setupFixture(t)
	opts.client = &smtpClientMock{
		failRcpt: true,
	}

	m := message.MakeString("hello world!")
	if err := opts.sendMail(m); err == nil {
		t.Fatal("error shuld not be nil")
	}
}

func TestSendMailErrorsIfDataFails(t *testing.T) {
	opts := setupFixture(t)
	opts.client = &smtpClientMock{
		failData: true,
	}

	m := message.MakeString("hello world!")
	if err := opts.sendMail(m); err == nil {
		t.Fatal("error shuld not be nil")
	}
}

func TestSendMailErrorsIfCreateFails(t *testing.T) {
	opts := setupFixture(t)
	opts.client = &smtpClientMock{
		failCreate: true,
	}

	m := message.MakeString("hello world!")
	if err := opts.sendMail(m); err == nil {
		t.Fatal("error shuld not be nil")
	}
}

func TestSendMailRecordsMessage(t *testing.T) {
	opts := setupFixture(t)
	m := message.MakeString("hello world!")
	if err := opts.sendMail(m); err != nil {
		t.Fatal(err)
	}
	mock, ok := opts.client.(*smtpClientMock)
	if !ok {
		t.Fatal("bad fixture")
	}
	if !strings.Contains(mock.message.String(), opts.Name) {
		t.Fatal("should be true")
	}
	if !strings.Contains(mock.message.String(), "plain") {
		t.Fatal("should be true")
	}
	if strings.Contains(mock.message.String(), "html") {
		t.Fatal("should be false")
	}

	opts.PlainTextContents = false
	if err := opts.sendMail(m); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mock.message.String(), opts.Name) {
		t.Fatal("should be true")
	}
	if !strings.Contains(mock.message.String(), "html") {
		t.Fatal("should be true")
	}
	if strings.Contains(mock.message.String(), "plain") {
		t.Fatal("should be false")
	}
}

func TestNewConstructor(t *testing.T) {
	opts := setupFixture(t)
	sender, err := MakeSender(nil)
	if err == nil {
		t.Fatal("error shold not be nil")
	}
	if sender != nil {
		t.Fatal("'sender' is expected to be nil")
	}

	sender, err = MakeSender(opts)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("'sender' is not expected to be nil")
	}
}

func TestSendMethod(t *testing.T) {
	opts := setupFixture(t)
	sender, err := MakeSender(opts)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("'sender' is not expected to be nil")
	}
	sender.SetPriority(level.Info)

	mock, ok := opts.client.(*smtpClientMock)
	if !ok {
		t.Fatal("'ok' should be true")
	}
	if mock.numMsgs != 0 {
		t.Fatal("values should be equal")
	}

	var m message.Composer
	m = message.MakeString("hello")
	m.SetPriority(level.Debug)
	sender.Send(m)
	if mock.numMsgs != 0 {
		t.Fatal("values should be equal", mock.numMsgs)
	}

	m = message.MakeString("")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numMsgs != 0 {
		t.Fatal("values should be equal")
	}

	m = message.MakeString("world")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numMsgs != 1 {
		t.Fatal("values should be equal")
	}
}

func TestSendMethodWithError(t *testing.T) {
	opts := setupFixture(t)
	sender, err := MakeSender(opts)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("'sender' is not expected to be nil")
	}

	mock, ok := opts.client.(*smtpClientMock)
	if !ok {
		t.Fatal("'ok' should be true")
	}
	if mock.numMsgs != 0 {
		t.Fatal("values should be equal")
	}
	if mock.failData {
		t.Fatal("should be false")
	}

	m := message.MakeString("world")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if mock.numMsgs != 1 {
		t.Fatal("values should be equal")
	}

	mock.failData = true
	sender.Send(m)
	if mock.numMsgs != 1 {
		t.Fatal("values should be equal")
	}
}

func TestSendMethodWithEmailComposerOverridesSMTPOptions(t *testing.T) {
	opts := setupFixture(t)
	sender, err := MakeSender(opts)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("'sender' is not expected to be nil")
	}

	sender.SetErrorHandler(func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	})

	mock, ok := opts.client.(*smtpClientMock)
	if !ok {
		t.Fatal("'ok' should be true")
	}
	if mock.numMsgs != 0 {
		t.Fatal("values should be equal")
	}
	m := NewMessage(level.Notice, Message{
		From:              "Mr Super Powers <from@example.com>",
		Recipients:        []string{"to@example.com"},
		Subject:           "Test",
		Body:              "just a test",
		PlainTextContents: true,
		Headers: map[string][]string{
			"X-Custom-Header":           {"special"},
			"Content-Type":              {"something/proprietary"},
			"Content-Transfer-Encoding": {"somethingunexpected"},
		},
	})
	if !m.Loggable() {
		t.Fatal("'m.Loggable()' should be true")
	}

	sender.Send(m)
	if mock.numMsgs != 1 {
		t.Fatal("values should be equal")
	}

	contains := []string{
		"From: \"Mr Super Powers\" <from@example.com>\r\n",
		"To: <to@example.com>\r\n",
		"Subject: Test\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Type: something/proprietary\r\n",
		"X-Custom-Header: special\r\n",
		"Content-Transfer-Encoding: base64\r\n",
		"anVzdCBhIHRlc3Q=",
	}
	data := mock.message.String()
	for i := range contains {
		if !strings.Contains(data, contains[i]) {
			t.Fatalf("expected %q to contain %q", data, contains[i])
		}
	}
}
