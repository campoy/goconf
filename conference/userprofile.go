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
	http.Handle("/userprofile", authHandler(userProfileHandler))
	http.Handle("/saveprofile", authHandler(saveProfileHandler))
}

const UserKind = "RegisteredUser"

type UserProfile struct {
	Name       string
	Topics     []string
	MainEmail  string
	NotifEmail string
	Tickets    []Ticket
}

func (u UserProfile) SelectedTopic(topic string) bool {
	for _, t := range u.Topics {
		if t == topic {
			return true
		}
	}
	return false
}

func userProfileHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	var data UserProfile

	heading := "Edit your"

	k := datastore.NewKey(ctx, UserKind, u.Email, 0, nil)
	if err := datastore.Get(ctx, k, &data); err != nil {
		if err != datastore.ErrNoSuchEntity {
			return fmt.Errorf("get userprofile: %v", err)
		}
		heading = "Create a "
		data.MainEmail = u.Email
	}

	p, err := NewPage(ctx, "userprofile", data)
	if err != nil {
		return fmt.Errorf("create userprofile page: %v", err)
	}
	p.Heading = heading
	return p.Render(w)
}

func saveProfileHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	if r.Method != "POST" {
		return RedirectTo("/userprofile")
	}
	up := UserProfile{
		MainEmail:  u.Email,
		Name:       r.FormValue("person_name"),
		NotifEmail: r.FormValue("notification_email"),
		Topics:     r.Form["topics"],
	}

	k := datastore.NewKey(ctx, UserKind, u.Email, 0, nil)
	if _, err := datastore.Put(ctx, k, &up); err != nil {
		return fmt.Errorf("save userprofile: %v", err)
	}
	return RedirectTo("/userprofile")
}
