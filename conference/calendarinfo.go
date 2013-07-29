package conference

import (
	"fmt"
	"io"
	"net/http"

	"appengine"
	"appengine/urlfetch"

	"code.google.com/p/google-api-go-client/calendar/v3"
	"github.com/campoy/goconf/oauth2"
)

var calCfg = config(calendar.CalendarScope)

func init() {
	handler := oauth2.HandlerFunc("calendar", handler(calendarInfoHandler), calCfg)
	http.HandleFunc("/calendarinfo", handler)
}

func calendarInfoHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	client, err := oauth2.Client(r, &urlfetch.Transport{Context: ctx}, calCfg)
	if err != nil {
		return nil
	}

	cal, err := calendar.New(client)
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

	return renderPage(ctx, w, "showcalendar", evts)
}
