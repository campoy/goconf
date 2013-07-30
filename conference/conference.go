// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

const ConferenceKind = "Conference"

func init() {
	http.Handle("/scheduleconference", handler(scheduleConfHandler))
	http.Handle("/saveconference", authHandler(saveConfHandler))
}

func scheduleConfHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	p, err := NewPage(ctx, "scheduleconf", nil)
	if err != nil {
		return fmt.Errorf("create scheduleconf page: %v", err)
	}
	return p.Render(w)
}

type Conference struct {
	Name            string
	Description     string
	City            string
	Topic           string
	MaxAttendees    int64
	NumTixAvailable int64
	StartDate       time.Time
	EndDate         time.Time
	Organizer       string
	Key             string
}

func saveConfHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	nAtt, err := strconv.ParseInt(r.FormValue("max_attendees"), 10, 32)
	if err != nil {
		return fmt.Errorf("bad max_attendees value: %q", r.FormValue("max_attendees"))
	}
	start, err := time.Parse("2006-01-02", r.FormValue("start_date"))
	if err != nil {
		return fmt.Errorf("bad start_date value: %q", r.FormValue("start_date"))
	}
	end, err := time.Parse("2006-01-02", r.FormValue("end_date"))
	if err != nil {
		return fmt.Errorf("bad end_date value: %q", r.FormValue("end_date"))
	}
	confName := r.FormValue("conf_name")
	conf := Conference{
		Name:            confName,
		Description:     r.FormValue("conf_desc"),
		City:            r.FormValue("city"),
		Topic:           r.FormValue("topic"),
		MaxAttendees:    nAtt,
		NumTixAvailable: nAtt,
		StartDate:       start,
		EndDate:         end,
		Organizer:       u.Email,
	}

	k := datastore.NewKey(ctx, ConferenceKind, "", 0, nil)
	k, err = datastore.Put(ctx, k, &conf)
	if err != nil {
		return fmt.Errorf("save conference: %v", err)
	}

	for i := int64(0); i < nAtt; i++ {
		ticket := Ticket{
			Number:   int64(i),
			Status:   TicketAvailable,
			ConfName: confName,
			ConfKey:  k.Encode(),
		}
		tk := datastore.NewKey(ctx, TicketKind, "", 0, k)
		tk, err := datastore.Put(ctx, tk, &ticket)
		if err != nil {
			return fmt.Errorf("save ticket %v for conference %v: %v", i, confName, err)
		}
	}

	ann := fmt.Sprintf("A new conference has just been scheduled! %s in %s. Don't wait; book now!",
		conf.Name, conf.City)
	SetLatestAnnouncement(ctx, ann)

	red := fmt.Sprintf("/showtickets?conf_key_str=%v&conf_name=%v",
		url.QueryEscape(k.Encode()),
		url.QueryEscape(conf.Name))
	return RedirectTo(red)
}
