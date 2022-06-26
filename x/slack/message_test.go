package slack

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bluele/slack"
)

func TestSlackAttachmentFieldConvert(t *testing.T) {
	gripField := AttachmentField{
		Title: "1",
		Value: "2",
		Short: true,
	}
	slackField := gripField.convert()

	if slackField.Title != "1" {
		t.Error("elements should be equal")
	}
	if slackField.Value != "2" {
		t.Error("elements should be equal")
	}
	if !slackField.Short {
		t.Error("should be true")
	}
}

func TestSlackAttachmentConvert(t *testing.T) {
	af := AttachmentField{
		Title: "1",
		Value: "2",
		Short: true,
	}

	at := Attachment{
		Color:      "1",
		Fallback:   "2",
		AuthorName: "3",
		AuthorIcon: "6",
		Title:      "7",
		TitleLink:  "8",
		Text:       "10",
		Fields:     []*AttachmentField{&af},
		MarkdownIn: []string{"15", "16"},
	}
	slackAttachment := at.convert()

	if slackAttachment.Color != "1" {
		t.Error("elements should be equal")
	}
	if slackAttachment.Fallback != "2" {
		t.Error("elements should be equal")
	}
	if slackAttachment.AuthorName != "3" {
		t.Error("elements should be equal")
	}
	if slackAttachment.AuthorIcon != "6" {
		t.Error("elements should be equal")
	}
	if slackAttachment.Title != "7" {
		t.Error("elements should be equal")
	}
	if slackAttachment.TitleLink != "8" {
		t.Error("elements should be equal")
	}
	if slackAttachment.Text != "10" {
		t.Error("elements should be equal")
	}
	if strings.Join(slackAttachment.MarkdownIn, "+") != "15+16" {
		t.Error("elements should be equal")
	}
	if f := slackAttachment.Fields; len(f) != 1 {
		t.Errorf("%v should have a length of 1", f)
	}
	if slackAttachment.Fields[0].Title != "1" {
		t.Error("elements should be equal")
	}
	if slackAttachment.Fields[0].Value != "2" {
		t.Error("elements should be equal")
	}
	if !slackAttachment.Fields[0].Short {
		t.Error("should be true")
	}
}

func TestSlackAttachmentIsSame(t *testing.T) {
	grip := Attachment{}
	slack := slack.Attachment{}

	vGrip := reflect.TypeOf(grip)
	vSlack := reflect.TypeOf(slack)

	for i := 0; i < vSlack.NumField(); i++ {
		slackField := vSlack.Field(i)
		gripField, found := vGrip.FieldByName(slackField.Name)
		if !found {
			continue
		}

		referenceTag := slackField.Tag.Get("json")
		jsonTag := gripField.Tag.Get("json")
		if referenceTag != jsonTag {
			t.Errorf("SlackAttachment.%s should have json tag with value: \"%s\"", jsonTag, gripField.Name)
		}
		bsonTag := gripField.Tag.Get("bson")
		if referenceTag != bsonTag {
			t.Errorf("SlackAttachment.%s should have bson tag with value: \"%s\"", bsonTag, gripField.Name)
		}
		yamlTag := gripField.Tag.Get("yaml")
		if referenceTag != yamlTag {
			t.Errorf("SlackAttachment.%s should have yaml tag with value: \"%s\"", yamlTag, gripField.Name)
		}
	}

}

func TestSlackAttachmentFieldIsSame(t *testing.T) {
	gripStruct := AttachmentField{}
	slackStruct := slack.AttachmentField{}

	vGrip := reflect.TypeOf(gripStruct)
	vSlack := reflect.TypeOf(slackStruct)

	if vGrip.NumField() != vSlack.NumField() {
		t.Error("elements should be equal")
	}
	for i := 0; i < vSlack.NumField(); i++ {
		slackField := vSlack.Field(i)
		gripField, found := vGrip.FieldByName(slackField.Name)
		if !found {
			t.Errorf("field %s found in slack.AttachmentField, but not in message.SlackAttachmentField", slackField.Name)
			continue
		}

		referenceTag := slackField.Tag.Get("json")
		jsonTag := gripField.Tag.Get("json")
		if referenceTag != jsonTag {
			t.Errorf("SlackAttachmentField.%s should have json tag with value: \"%s\"", jsonTag, gripField.Name)
		}
		bsonTag := gripField.Tag.Get("bson")
		if referenceTag != bsonTag {
			t.Errorf("SlackAttachmentField.%s should have bson tag with value: \"%s\"", bsonTag, gripField.Name)
		}
		yamlTag := gripField.Tag.Get("yaml")
		if referenceTag != yamlTag {
			t.Errorf("SlackAttachmentField.%s should have yaml tag with value: \"%s\"", yamlTag, gripField.Name)
		}

		if gripField.Type.Kind() != slackField.Type.Kind() {
			t.Error("elements should be equal")
		}
	}
}
