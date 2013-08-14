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
	"sync"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/taskqueue"
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

func ConfFromRequest(r *http.Request) (*Conference, error) {
	nAtt, err := strconv.ParseInt(r.FormValue("max_attendees"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("bad max_attendees value: %q", r.FormValue("max_attendees"))
	}
	start, err := time.Parse("2006-01-02", r.FormValue("start_date"))
	if err != nil {
		return nil, fmt.Errorf("bad start_date value: %q", r.FormValue("start_date"))
	}
	end, err := time.Parse("2006-01-02", r.FormValue("end_date"))
	if err != nil {
		return nil, fmt.Errorf("bad end_date value: %q", r.FormValue("end_date"))
	}
	confName := r.FormValue("conf_name")
	email := ""
	if u := user.Current(appengine.NewContext(r)); u != nil {
		email = u.Email
	}

	return &Conference{
		Name:            confName,
		Description:     r.FormValue("conf_desc"),
		City:            r.FormValue("city"),
		Topic:           r.FormValue("topic"),
		MaxAttendees:    nAtt,
		NumTixAvailable: nAtt,
		StartDate:       start,
		EndDate:         end,
		Organizer:       email,
	}, nil
}

func saveConfHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	conf, err := ConfFromRequest(r)
	if err != nil {
		return fmt.Errorf("conf from request: %v", err)
	}

	k := datastore.NewKey(ctx, ConferenceKind, "", 0, nil)
	k, err = datastore.Put(ctx, k, conf)
	if err != nil {
		return fmt.Errorf("save conference: %v", err)
	}

	// Create tickets for the conference.
	errc := make(chan error, int(conf.MaxAttendees))
	var wg sync.WaitGroup
	wg.Add(int(conf.MaxAttendees))
	for i := int64(0); i < conf.MaxAttendees; i++ {
		go func(ticketNum int) {
			defer wg.Done()
			ticket := Ticket{
				Number:   ticketNum,
				Status:   TicketAvailable,
				ConfName: conf.Name,
				ConfKey:  k.Encode(),
			}
			tk := datastore.NewKey(ctx, TicketKind, "", 0, k)
			_, err := datastore.Put(ctx, tk, &ticket)
			if err != nil {
				errc <- fmt.Errorf("save ticket %v for conference %v: %v", ticketNum, conf.Name, err)
			}
		}(i)
	}
	wg.Wait()
	for err := range errc {
		if err != nil {
			return err
		}
	}

	task := taskqueue.NewPOSTTask(
		"/processconference",
		url.Values{"conf_key": []string{k.Encode()}},
	)
	_, err = taskqueue.Add(ctx, task, "")
	if err != nil {
		return fmt.Errorf("add process conf task: %v", err)
	}

	return RedirectTo(
		fmt.Sprintf("/showtickets?conf_key_str=%v&conf_name=%v",
			url.QueryEscape(k.Encode()),
			url.QueryEscape(conf.Name)))
}
