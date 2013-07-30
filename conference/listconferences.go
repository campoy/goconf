// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"fmt"
	"io"
	"net/http"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

func init() {
	http.Handle("/listconferences", authHandler(listConfsHandler))
}

type ConfList struct {
	Title       string
	Conferences []Conference
}

// List of all conference lists to display
var conferenceLists = [...]struct {
	title string
	query *datastore.Query
}{
	{
		"All Conferences",
		datastore.NewQuery(ConferenceKind),
	},
	{
		"All Conferences Sorted Alphabetically",
		datastore.NewQuery(ConferenceKind).
			Order("Name"),
	},
	{
		"All Conferences In London sorted Alphabetically",
		datastore.NewQuery(ConferenceKind).
			Filter("City =", "London").
			Order("Name"),
	},
	{
		"All Conferences About Medical Innovations In London",
		datastore.NewQuery(ConferenceKind).
			Filter("City =", "London").
			Filter("Topic =", "Medical Innovations"),
	},
	{
		"All conferences With 50 or more attendees",
		datastore.NewQuery(ConferenceKind).
			Filter("MaxAttendees >", 50),
	},
}

func NewConfList(ctx appengine.Context, title string, q *datastore.Query) (ConfList, error) {
	list := ConfList{Title: title}

	ks, err := q.GetAll(ctx, &list.Conferences)
	if err != nil {
		return list, fmt.Errorf("get %q: %v", list.Title, err)
	}
	for i, k := range ks {
		list.Conferences[i].Key = k.Encode()
	}
	return list, nil
}

func listConfsHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	data := []ConfList{}

	for _, c := range conferenceLists {
		l, err := NewConfList(ctx, c.title, c.query)
		if err != nil {
			return err
		}
		data = append(data, l)
	}

	l, err := NewConfList(ctx,
		"All conferences About Medical Innovations in London with "+u.Email,
		datastore.NewQuery(ConferenceKind).
			Filter("City =", "London").
			Filter("Topic =", "Medical Innovations").
			Filter("Organizer =", u.Email),
	)
	if err != nil {
		return err
	}
	data = append(data, l)

	p, err := NewPage(ctx, "listconfs", data)
	if err != nil {
		return fmt.Errorf("create listconfs page: %v", err)
	}
	return p.Render(w)
}
