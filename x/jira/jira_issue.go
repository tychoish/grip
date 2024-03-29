package jira

import (
	"fmt"

	"github.com/tychoish/grip/message"
)

type jiraMessage struct {
	issue *Issue
	message.Base
}

// Issue requires project and summary to create a real jira issue.
// Other fields depend on permissions given to the specific project, and
// all fields must be legitimate custom fields defined for the project.
// To see whether you have the right permissions to create an issue with certain
// fields, check your JIRA interface on the web.
type Issue struct {
	IssueKey    string   `bson:"issue_key" json:"issue_key" yaml:"issue_key"`
	Project     string   `bson:"project" json:"project" yaml:"project"`
	Summary     string   `bson:"summary" json:"summary" yaml:"summary"`
	Description string   `bson:"description" json:"description" yaml:"description"`
	Reporter    string   `bson:"reporter" json:"reporter" yaml:"reporter"`
	Assignee    string   `bson:"assignee" json:"assignee" yaml:"assignee"`
	Type        string   `bson:"type" json:"type" yaml:"type"`
	Components  []string `bson:"components" json:"components" yaml:"components"`
	Labels      []string `bson:"labels" json:"labels" yaml:"labels"`
	FixVersions []string `bson:"versions" json:"versions" yaml:"versions"`
	// ... other fields
	Fields   map[string]any `bson:"fields" json:"fields" yaml:"fields"`
	Callback func(string)   `bson:"-" json:"-" yaml:"-"`
}

// Field is a struct composed of a key-value pair.
type Field struct {
	Key   string
	Value any
}

// MakeIssue creates a jiraMessage instance with the given JiraIssue.
func MakeIssue(issue *Issue) message.Composer {
	return &jiraMessage{
		issue: issue,
	}
}

// NewIssue creates and returns a fully formed jiraMessage, which implements
// message.Composer. project string and summary string are required, and any
// number of additional fields may be included. Fields with keys Reporter, Assignee,
// Type, and Labels will be specifically assigned to respective fields in the new
// jiraIssue included in the jiraMessage, (e.g. JiraIssue.Reporter, etc), and
// all other fields will be included in jiraIssue.Fields.
func NewIssue(project, summary string, fields ...Field) message.Composer {
	issue := &Issue{
		Project: project,
		Summary: summary,
		Fields:  map[string]any{},
	}

	// Assign given fields to jira issue fields
	for _, f := range fields {
		switch f.Key {
		case "reporter", "Reporter":
			issue.Reporter = f.Value.(string)
		case "assignee", "Assignee":
			issue.Assignee = f.Value.(string)
		case "type", "Type":
			issue.Type = f.Value.(string)
		case "labels", "Labels":
			issue.Labels = f.Value.([]string)
		case "component", "Component":
			issue.Components = f.Value.([]string)
		default:
			issue.Fields[f.Key] = f.Value
		}
	}

	// Setting "Task" as the default value for IssueType
	if issue.Type == "" {
		issue.Type = "Task"
	}

	return MakeIssue(issue)
}

func (m *jiraMessage) String() string { return m.issue.Summary }
func (m *jiraMessage) Raw() any       { return m.issue }
func (*jiraMessage) Structured() bool { return true }
func (m *jiraMessage) Loggable() bool { return m.issue.Summary != "" && m.issue.Type != "" }
func (m *jiraMessage) Annotate(k string, v any) {
	if m.issue.Fields == nil {
		m.issue.Fields = map[string]any{}
	}

	if value, ok := v.(string); ok {
		m.issue.Fields[k] = value
		return
	}

	m.issue.Fields[k] = fmt.Sprint(v)
}
