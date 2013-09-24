package conf

import (
	"text/template"

	"appengine/datastore"

	"github.com/campoy/goconf/pkg/conf"
)

var topicList = []string{
	"Medical Innovations",
	"Programming Languages",
	"Web Technologies",
	"Movie Making",
}

var cityList = []string{
	"London",
	"Chicago",
	"San Francisco",
	"Paris",
}

// List of all conference lists to display
var conferenceLists = [...]struct {
	title string
	query *datastore.Query
}{
	{
		"All Conferences",
		datastore.NewQuery(conf.ConferenceKind),
	},
	{
		"All Conferences Sorted Alphabetically",
		datastore.NewQuery(conf.ConferenceKind).
			Order("Name"),
	},
	{
		"All Conferences In London sorted Alphabetically",
		datastore.NewQuery(conf.ConferenceKind).
			Filter("City =", "London").
			Order("Name"),
	},
	{
		"All Conferences About Medical Innovations In London",
		datastore.NewQuery(conf.ConferenceKind).
			Filter("City =", "London").
			Filter("Topic =", "Medical Innovations"),
	},
	{
		"All conferences With 50 or more attendees",
		datastore.NewQuery(conf.ConferenceKind).
			Filter("MaxAttendees >", 50),
	},
}

// Notification email data
const emailSender = "campoy@golang.org"

var emailSubjectTmpl = template.Must(template.New("subject").Parse("Conference you might be interested in: {{.Name}}"))
var emailBodyTmpl = template.Must(template.ParseFiles("templates/email.tmpl"))
