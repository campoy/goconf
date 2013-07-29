package conference

import (
	"fmt"
	"io"
	"net/http"

	"appengine"
	"appengine/urlfetch"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/calendar/v3"
)

var calendarConfig = config(calendar.CalendarScope)

func init() {
	http.HandleFunc("/calendarinfo",
		func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, calendarConfig.AuthCodeURL("foo"), http.StatusFound)
		},
	)
	http.Handle("/oauth2callback", handler(calendarInfoHandler))
}

func calendarInfoHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)

	t := &oauth.Transport{
		Config:    calendarConfig,
		Transport: &urlfetch.Transport{Context: ctx},
	}
	t.Exchange(r.FormValue("code"))

	cal, err := calendar.New(t.Client())
	if err != nil {
		return fmt.Errorf("create calendar service: %v", err)
	}
	evts, err := cal.Events.List("primary").
		MaxResults(10).
		TimeMin("2013-05-28T00:00:00-08:00").
		Do()

	if err != nil {
		return fmt.Errorf("get calendar events: %v", err)
	}

	ctx.Infof("%#v", evts)

	return renderPage(ctx, w, "showcalendar", evts)
}
