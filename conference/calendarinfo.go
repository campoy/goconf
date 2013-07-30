// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"fmt"
	"io"
	"net/http"

	"appengine"
	"appengine/urlfetch"

	"code.google.com/p/google-api-go-client/calendar/v3"
	"github.com/campoy/oauth2util"
)

var calCfg = config(calendar.CalendarScope)

func init() {
	oauth2util.Handle("/calendarinfo", handler(calendarInfoHandler), calCfg)
}

func calendarInfoHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	client, err := oauth2util.Client(r, &urlfetch.Transport{Context: ctx}, calCfg)
	if err != nil {
		return fmt.Errorf("oauth2 client: %v", err)
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
