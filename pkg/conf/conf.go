// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD style
// license that can be found in the LICENSE file.

// Package conf provides the data models for the conference management
// business.
//
// This package works on top of Google App Engine, using datastore as the
// main storage system.
package conf

import (
	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"appengine/memcache"
)

const (
	LatestAnnouncementKey = "LatestAnnouncement" // Memcache key for the latest announcement

	// Datastore kinds
	AnnouncementKind = "Announcement"
	ConferenceKind   = "Conference"
	TicketKind       = "Ticket"
	UserKind         = "RegisteredUser"
)

// A ConfList contains a list of conferences with a title for the whole list.
// For instance the title could be "All the conferences in London", and the
// list of conferences would contain all the conferences matching that description.
type ConfList struct {
	Title       string
	Conferences []Conference
}

// LoadConfList executes the given query and returns a new ConfList with the given title and
// containing the conferences obtained from the query result.
func LoadConfList(ctx appengine.Context, title string, q *datastore.Query) (*ConfList, error) {
	list := &ConfList{Title: title}

	ks, err := q.GetAll(ctx, &list.Conferences)
	if err != nil {
		return nil, fmt.Errorf("get %q: %v", list.Title, err)
	}
	for i, k := range ks {
		list.Conferences[i].key = k
	}
	return list, nil
}

// Conference contains all the information for a conference.
type Conference struct {
	Name         string
	Description  string
	City         string
	Topic        string
	MaxAttendees int
	TixAvailable int
	StartDate    time.Time
	EndDate      time.Time
	Organizer    string

	key *datastore.Key
}

// ID returns a unique identifier for any Conference that has already
// been saved in the datastore.
func (c *Conference) ID() string { return c.key.Encode() }

// LoadConference loads a conference from the datastore given its unique id.
func LoadConference(ctx appengine.Context, id string) (*Conference, error) {
	k, err := datastore.DecodeKey(id)
	if err != nil {
		return nil, fmt.Errorf("wrong key %q: %v", id, err)
	}
	return loadConference(ctx, k)
}

// loadConference loads a conference from the datastore given a datastore key.
func loadConference(ctx appengine.Context, k *datastore.Key) (*Conference, error) {
	var conf Conference
	if err := datastore.Get(ctx, k, &conf); err != nil {
		return nil, err
	}
	conf.key = k
	return &conf, nil
}

// Save saves a conference into datastore.
// This doesn't save any of the tickets of the conference.
func (conf *Conference) Save(ctx appengine.Context) error {
	k := conf.key
	if k == nil {
		k = datastore.NewKey(ctx, ConferenceKind, "", 0, nil)
	}

	k, err := datastore.Put(ctx, k, conf)
	if err != nil {
		return fmt.Errorf("save conference: %v", err)
	}
	conf.key = k
	return nil
}

// CreateAndSaveTickets creates as many tickets as the max number of attendees
// for the conference and saves them into the datastore.
func (conf *Conference) CreateAndSaveTickets(ctx appengine.Context) error {
	for i := 1; i <= conf.MaxAttendees; i++ {
		t := Ticket{
			Number:   i,
			State:    TicketAvailable,
			ConfName: conf.Name,
		}
		if err := t.save(ctx, conf.key); err != nil {
			return fmt.Errorf("save ticket %v for conference %v: %v", i, conf.Name, err)
		}
	}
	return nil
}

// MailNotifications finds all the users interested in the topic of the conference
// and sends them an email notifying the conference.
//
// This operation can be slow and shouldn't be performed in the critical path of the
// application.
func (conf *Conference) MailNotifications(ctx appengine.Context, sender, subject, body string) error {
	ks, err := datastore.NewQuery(UserKind).
		Filter("Topics =", conf.Topic).
		KeysOnly().
		GetAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("get interested users: %v", err)
	}

	to := make([]string, len(ks))
	for i, k := range ks {
		to[i] = k.StringID()
	}

	msg := &mail.Message{
		Sender:  sender,
		To:      to,
		Subject: subject,
		Body:    body,
	}
	if err := mail.Send(ctx, msg); err != nil {
		return fmt.Errorf("send mail: %v", err)
	}
	return nil
}

// Ticket is a single ticket for a conference.
type Ticket struct {
	Number   int
	State    TicketState
	ConfName string
	Owner    string

	key *datastore.Key
}

// TicketState represents the state of a conference ticket.
type TicketState string

const (
	TicketAvailable TicketState = "available"
	TicketSold      TicketState = "sold"
)

// ID returns a unique identifier for any Ticket that has already
// been saved in the datastore.
func (t *Ticket) ID() string { return t.key.Encode() }

// LoadTicket loads a Ticket from the datastore given its unique id.
func LoadTicket(ctx appengine.Context, id string) (*Ticket, error) {
	k, err := datastore.DecodeKey(id)
	if err != nil {
		return nil, fmt.Errorf("wrong key: %v", err)
	}

	var t Ticket
	if err := datastore.Get(ctx, k, &t); err != nil {
		return nil, err
	}
	t.key = k
	return &t, nil
}

func (t *Ticket) save(ctx appengine.Context, confKey *datastore.Key) error {
	k := datastore.NewKey(ctx, TicketKind, "", int64(t.Number), confKey)
	k, err := datastore.Put(ctx, k, t)
	if err != nil {
		return err
	}
	t.key = k
	return nil
}

// SellTo marks a ticket as sold to the given email updating the corresponding
// conference and saving all the modified elements to the datastore.
func (t *Ticket) SellTo(ctx appengine.Context, email string) error {
	// All done in a single transaction, to avoid data races.
	return datastore.RunInTransaction(ctx, func(c appengine.Context) error {
		if t.State != TicketAvailable {
			return fmt.Errorf("cannot sell a %v ticket", t.State)
		}

		conf, err := loadConference(ctx, t.key.Parent())
		if err != nil {
			return fmt.Errorf("load ticket's conference: %v", err)
		}

		up, err := LoadUserProfile(ctx, email)
		if err != nil {
			return fmt.Errorf("load user profile: %v", err)
		}

		conf.TixAvailable--
		t.State = TicketSold
		t.Owner = up.MainEmail

		if t.save(ctx, conf.key); err != nil {
			return fmt.Errorf("save ticket: %v", err)
		}
		if conf.Save(ctx); err != nil {
			return fmt.Errorf("save conference: %v", err)
		}
		return nil
	}, nil)
}

// AvailableTickets loads all the tickets that have State TicketAvailable for the
// conference.
func (conf *Conference) AvailableTickets(ctx appengine.Context) ([]Ticket, error) {
	var ts []Ticket
	ks, err := datastore.NewQuery(TicketKind).
		Ancestor(conf.key).
		Filter("State =", TicketAvailable).
		Order("Number").
		GetAll(ctx, &ts)
	if err != nil {
		return nil, fmt.Errorf("load tickets: %v", err)
	}

	for i, k := range ks {
		ts[i].key = k
	}
	return ts, nil
}

// An Announcement is a message to be displayed to all the users of the
// application. Only the newest Announcement is normally displayed.
type Announcement struct {
	Message string
	Time    time.Time
}

// NewAnnouncement returns a new Announcement with the given text and the current
// time.
func NewAnnouncement(msg string) *Announcement {
	return &Announcement{
		Message: msg,
		Time:    time.Now(),
	}
}

// Save saves the Announcement to both datastore and memcache.
// Memcache errors are logged and ignored.
func (a *Announcement) Save(ctx appengine.Context) error {
	k := datastore.NewKey(ctx, AnnouncementKind, "", 0, nil)
	if _, err := datastore.Put(ctx, k, a); err != nil {
		return err
	}
	if err := a.memcacheSet(ctx); err != nil {
		ctx.Errorf("memcache set: %v", err)
	}
	return nil
}

// memcacheSet sets a as the latests announcement in memcache.
func (a *Announcement) memcacheSet(ctx appengine.Context) error {
	item := &memcache.Item{
		Key:        LatestAnnouncementKey,
		Object:     a,
		Expiration: 1 * time.Hour,
	}
	return memcache.JSON.Set(ctx, item)
}

// LatestAnnouncement returns the latest announcement from either memcache
// or the datastore.
// If no announcement is found LatestAnnouncement returns nil and no error.
func LatestAnnouncement(ctx appengine.Context) (*Announcement, error) {
	var a Announcement
	_, err := memcache.JSON.Get(ctx, LatestAnnouncementKey, &a)
	if err == nil {
		return &a, nil
	}

	_, err = datastore.NewQuery(AnnouncementKind).
		Order("-Time"). // Order from newer to older
		Limit(1).       // Get only one result at most
		Run(ctx).
		Next(&a)
	if err == datastore.Done {
		// There's no announcement
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get last announcement from datastore: %v", err)
	}
	a.memcacheSet(ctx)
	return &a, nil
}

// UserProfile contains the information for a registered user.
type UserProfile struct {
	Name       string
	Topics     []string
	MainEmail  string
	NotifEmail string

	tickets []Ticket
}

// InterestedIn returns true if the user has declared an interest on the given topic.
func (u *UserProfile) InterestedIn(topic string) bool {
	for _, t := range u.Topics {
		if t == topic {
			return true
		}
	}
	return false
}

// Attending returns the name of all the conferences for which the user has acquired a ticket.
func (u *UserProfile) Attending() []string {
	set := make(map[string]bool)
	for _, t := range u.tickets {
		set[t.ConfName] = true
	}
	list := make([]string, 0, len(set))
	for name := range set {
		list = append(list, name)
	}
	return list
}

// LoadUserProfile loads a user profile from the datastore given an email.
// If the user profile is not found a new one is created and stored in the datastore.
func LoadUserProfile(ctx appengine.Context, email string) (*UserProfile, error) {
	var up UserProfile
	k := datastore.NewKey(ctx, UserKind, email, 0, nil)
	err := datastore.Get(ctx, k, &up)
	if err == datastore.ErrNoSuchEntity {
		up.MainEmail = email
		return &up, up.Save(ctx)
	}

	ks, err := datastore.NewQuery(TicketKind).Filter("Owner =", email).GetAll(ctx, &up.tickets)
	if err != nil {
		return nil, err
	}
	for i, k := range ks {
		up.tickets[i].key = k
	}

	return &up, nil
}

// Save save a UserProfile to the datastore.
func (up *UserProfile) Save(ctx appengine.Context) error {
	if len(up.MainEmail) == 0 {
		return fmt.Errorf("cannot save user profile without email")
	}
	k := datastore.NewKey(ctx, UserKind, up.MainEmail, 0, nil)
	_, err := datastore.Put(ctx, k, up)
	return err
}
