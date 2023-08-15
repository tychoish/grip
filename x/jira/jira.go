package jira

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

// jiraIssueKey is the key in a message.Fields that will hold the ID of the issue created
const jiraIssueKey = "jira-key"

type jiraJournal struct {
	opts *Options
	send.Base
}

// Options include configurations for the JIRA client
type Options struct {
	Name          string // Name of the journaler
	BaseURL       string // URL of the JIRA instance
	BasicAuthOpts BasicAuth
	Oauth1Opts    Oauth1
	HTTPClient    *http.Client
	client        jiraClient
}

type BasicAuth struct {
	UseBasicAuth bool
	Username     string
	Password     string
}

type Oauth1 struct {
	PrivateKey  []byte
	AccessToken string
	TokenSecret string
	ConsumerKey string
}

// MakeIssueSender is the same as NewJiraLogger but uses a warning
// level of Trace
func MakeIssueSender(ctx context.Context, opts *Options) (send.Sender, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	j := &jiraJournal{
		opts: opts,
	}

	if err := j.opts.client.CreateClient(opts.HTTPClient, opts.BaseURL); err != nil {
		return nil, err
	}

	authOpts := jiraAuthOpts{
		username:           opts.BasicAuthOpts.Username,
		password:           opts.BasicAuthOpts.Password,
		addBasicAuthHeader: opts.BasicAuthOpts.UseBasicAuth,
		accessToken:        opts.Oauth1Opts.AccessToken,
		tokenSecret:        opts.Oauth1Opts.TokenSecret,
		privateKey:         opts.Oauth1Opts.PrivateKey,
		consumerKey:        opts.Oauth1Opts.ConsumerKey,
	}
	if err := j.opts.client.Authenticate(ctx, authOpts); err != nil {
		return nil, fmt.Errorf("jira authentication error: %v", err)
	}

	j.SetName(opts.Name)
	j.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))

	return j, nil
}

// Send post issues via jiraJournal with information in the message.Composer
func (j *jiraJournal) Send(m message.Composer) {
	if send.ShouldLog(j, m) {
		issueFields := getFields(m)
		if len(issueFields.Summary) > 254 {
			issueFields.Summary = issueFields.Summary[:254]
		}
		if len(issueFields.Description) > 32767 {
			issueFields.Description = issueFields.Description[:32767]
		}

		issueKey, err := j.opts.client.PostIssue(issueFields)
		if !j.HandleErrorOK(send.WrapError(err, m)) {
			return
		}
		populateKey(m, issueKey)
	}
}

// Validate inspects the contents of JiraOptions struct and returns an error in case of
// missing any required fields.
func (o *Options) Validate() error {
	if o == nil {
		return errors.New("jira options cannot be nil")
	}

	errs := []string{}

	if o.Name == "" {
		errs = append(errs, "no name specified")
	}

	if o.BaseURL == "" {
		errs = append(errs, "no baseURL specified")
	}

	if (o.BasicAuthOpts.Username == "") == (o.Oauth1Opts.AccessToken == "") {
		return errors.New("must specify exactly 1 method of authentication")
	}

	if o.client == nil {
		o.client = &jiraClientImpl{}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func getFields(m message.Composer) *jira.IssueFields {
	var issueFields *jira.IssueFields

	switch msg := m.Raw().(type) {
	case *Issue:
		issueFields = &jira.IssueFields{
			Project:     jira.Project{Key: msg.Project},
			Summary:     msg.Summary,
			Description: msg.Description,
		}
		if len(msg.Fields) != 0 {
			issueFields.Unknowns = msg.Fields
		}
		if msg.Reporter != "" {
			issueFields.Reporter = &jira.User{Name: msg.Reporter}
		}
		if msg.Assignee != "" {
			issueFields.Assignee = &jira.User{Name: msg.Assignee}
		}
		if msg.Type != "" {
			issueFields.Type = jira.IssueType{Name: msg.Type}
		}
		if len(msg.Labels) > 0 {
			issueFields.Labels = msg.Labels
		}
		if len(msg.Components) > 0 {
			issueFields.Components = make([]*jira.Component, 0, len(msg.Components))
			for _, component := range msg.Components {
				issueFields.Components = append(issueFields.Components,
					&jira.Component{
						Name: component,
					})
			}
		}
		if len(msg.FixVersions) > 0 {
			issueFields.FixVersions = make([]*jira.FixVersion, 0, len(msg.FixVersions))
			for _, version := range msg.FixVersions {
				issueFields.FixVersions = append(issueFields.FixVersions,
					&jira.FixVersion{
						Name: version,
					})
			}
		}

	case message.Fields:
		issueFields = &jira.IssueFields{
			Summary: fmt.Sprintf("%s", msg[message.FieldsMsgName]),
		}
		for k, v := range msg {
			if k == message.FieldsMsgName {
				continue
			}

			issueFields.Description += fmt.Sprintf("*%s*: %s\n", k, v)
		}

	default:
		issueFields = &jira.IssueFields{
			Summary:     m.String(),
			Description: fmt.Sprintf("%+v", msg),
		}
	}
	return issueFields
}

func populateKey(m message.Composer, issueKey string) {
	switch msg := m.Raw().(type) {
	case *Issue:
		msg.IssueKey = issueKey
		if msg.Callback != nil {
			msg.Callback(issueKey)
		}
	case message.Fields:
		msg[jiraIssueKey] = issueKey
	}
}

////////////////////////////////////////////////////////////////////////
//
// interface wrapper for the slack client so that we can mock things out
//
////////////////////////////////////////////////////////////////////////

type jiraClient interface {
	CreateClient(*http.Client, string) error
	Authenticate(context.Context, jiraAuthOpts) error
	PostIssue(*jira.IssueFields) (string, error)
	PostComment(string, string) error
}

type jiraAuthOpts struct {
	// basic or password auth
	username           string
	password           string
	addBasicAuthHeader bool

	// oauth 1.0
	privateKey  []byte
	accessToken string
	tokenSecret string
	consumerKey string
}

type jiraClientImpl struct {
	*jira.Client
	baseURL string
}

func (c *jiraClientImpl) CreateClient(client *http.Client, baseURL string) error {
	var err error
	c.baseURL = baseURL
	c.Client, err = jira.NewClient(client, baseURL)
	return err
}

func (c *jiraClientImpl) Authenticate(ctx context.Context, opts jiraAuthOpts) error {
	if opts.username != "" {
		if opts.addBasicAuthHeader {
			c.Client.Authentication.SetBasicAuth(opts.username, opts.password) //nolint

		} else {
			authed, err := c.Client.Authentication.AcquireSessionCookie(opts.username, opts.password) //nolint
			if err != nil {
				return fmt.Errorf("problem authenticating to jira as '%s' [%s]", opts.username, err.Error())
			}

			if !authed {
				return fmt.Errorf("problem authenticating to jira as '%s'", opts.username)
			}
		}
		return nil
	} else if opts.accessToken != "" {
		credentials := JiraOauthCredentials{
			PrivateKey:  opts.privateKey,
			AccessToken: opts.accessToken,
			TokenSecret: opts.tokenSecret,
			ConsumerKey: opts.consumerKey,
		}
		httpClient, err := Oauth1Client(ctx, credentials)
		if err != nil {
			return err
		}
		return c.CreateClient(httpClient, c.baseURL)
	}

	return errors.New("no authentication method specified")
}

func (c *jiraClientImpl) PostIssue(issueFields *jira.IssueFields) (string, error) {
	i := jira.Issue{Fields: issueFields}
	issue, resp, err := c.Client.Issue.Create(&i)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("encountered error logging to jira: %s [%s]",
				err.Error(), string(data))
		}

		return "", err
	}
	if issue == nil {
		return "", errors.New("no issue returned from Jira")
	}

	return issue.Key, nil
}

// todo: allow more parameters than just body?
func (c *jiraClientImpl) PostComment(issueID string, commentToPost string) error {
	_, _, err := c.Client.Issue.AddComment(issueID, &jira.Comment{Body: commentToPost})
	return err
}

type JiraOauthCredentials struct {
	PrivateKey  []byte
	AccessToken string
	TokenSecret string
	ConsumerKey string
}

// Oauth1Client is used to generate a http.Client that supports OAuth 1.0, to be used as the
// HTTP client in the Jira client implementation above
func Oauth1Client(ctx context.Context, credentials JiraOauthCredentials) (*http.Client, error) {
	keyDERBlock, _ := pem.Decode(credentials.PrivateKey)
	if keyDERBlock == nil {
		return nil, errors.New("unable to decode jira private key")
	}
	if !(keyDERBlock.Type == "PRIVATE KEY" || strings.HasSuffix(keyDERBlock.Type, " PRIVATE KEY")) {
		return nil, fmt.Errorf("malformed key block type: %s", keyDERBlock.Type)
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(keyDERBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse jira private key: %w", err)
	}
	oauthConfig := oauth1.Config{
		ConsumerKey: credentials.ConsumerKey,
		CallbackURL: "oob",
		Signer: &oauth1.RSASigner{
			PrivateKey: privateKey,
		},
	}
	oauthToken := oauth1.NewToken(credentials.AccessToken, credentials.TokenSecret)
	return oauthConfig.Client(ctx, oauthToken), nil
}
