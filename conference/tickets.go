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

const TicketKind = "Ticket"

type Ticket struct {
	Number   int64
	Status   TicketStatus
	ConfName string
	ConfKey  string
	Owner    string
	Key      string
}

type TicketStatus string

const (
	TicketAvailable TicketStatus = "available"
	TicketSold      TicketStatus = "sold"
)

func init() {
	http.Handle("/showtickets", handler(showTicketsHandler))
	http.Handle("/buyticket", authHandler(buyTicketHandler))
}

func showTicketsHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	confKey := r.FormValue("conf_key_str")
	confName := r.FormValue("conf_name")

	data := struct {
		ConfName string
		Tickets  []Ticket
	}{ConfName: confName}

	k, err := datastore.DecodeKey(confKey)
	if err != nil {
		return fmt.Errorf("decode key: %v", err)
	}

	ks, err := datastore.NewQuery(TicketKind).
		Ancestor(k).
		Filter("Status =", TicketAvailable).
		Order("Number").
		GetAll(ctx, &data.Tickets)
	if err != nil {
		return fmt.Errorf("get tickets: %v", err)
	}

	for i, k := range ks {
		data.Tickets[i].Key = k.Encode()
	}

	p, err := NewPage(ctx, "tickets", data)
	if err != nil {
		return fmt.Errorf("create tickets page: %v", err)
	}
	return p.Render(w)
}

func buyTicketHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	tk, err := datastore.DecodeKey(r.FormValue("ticket_key_str"))
	if err != nil {
		return fmt.Errorf("bad ticket key: %v", err)
	}

	// Get ticket
	var t Ticket
	if err := datastore.Get(ctx, tk, &t); err != nil {
		return fmt.Errorf("get ticket: %v", err)
	}
	if t.Status != TicketAvailable {
		return fmt.Errorf("ticket is %v", t.Status)
	}

	// Get conference
	var conf Conference
	ck := tk.Parent()
	if err := datastore.Get(ctx, ck, &conf); err != nil {
		return fmt.Errorf("get conference: %v", err)
	}

	// Get userprofile
	var up UserProfile
	upk := datastore.NewKey(ctx, UserKind, u.Email, 0, nil)
	if err := datastore.Get(ctx, upk, &up); err != nil {
		return fmt.Errorf("get user profile: %v", err)
	}

	err = datastore.RunInTransaction(ctx, func(c appengine.Context) error {
		t.Status = TicketSold
		conf.NumTixAvailable--
		up.Tickets = append(up.Tickets, t)

		if _, err = datastore.Put(c, tk, &t); err != nil {
			return fmt.Errorf("update ticket: %v", err)
		}
		if _, err = datastore.Put(c, ck, &conf); err != nil {
			return fmt.Errorf("update conference: %v", err)
		}
		if _, err = datastore.Put(c, upk, &up); err != nil {
			return fmt.Errorf("update user profile: %v", err)
		}

		return nil
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		return fmt.Errorf("transactionf failed: %v", err)
	}

	return RedirectTo("/userprofile")
}
