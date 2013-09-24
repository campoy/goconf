package conf

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/taskqueue"
	"appengine/urlfetch"
	"appengine/user"

	"code.google.com/p/google-api-go-client/calendar/v3"
	"github.com/campoy/goconf/pkg/auth"
	"github.com/campoy/goconf/pkg/conf"
	"github.com/campoy/goconf/pkg/tmpl"
)

var calendarConfig = config(calendar.CalendarScope)

func init() {
	tmpl.ParseTemplates("templates/*.tmpl")

	// home
	http.Handle("/", handler(homeHandler))

	// conferences
	http.Handle("/scheduleconference", handler(scheduleConfHandler))
	http.Handle("/saveconference", authHandler(saveConfHandler))
	http.Handle("/listconferences", authHandler(listConfsHandler))
	http.Handle("/notifyinterestedusers", handler(notifyInterestedUsersHandler))

	// admin page
	http.Handle("/developer", handler(developerHandler))

	// tickets
	http.Handle("/showtickets", handler(showTicketsHandler))
	http.Handle("/buyticket", authHandler(buyTicketHandler))

	// user profile
	http.Handle("/userprofile", authHandler(userProfileHandler))
	http.Handle("/saveprofile", authHandler(saveProfileHandler))
	auth.Handle("/calendarinfo", handler(calendarInfoHandler), calendarConfig)
}

// home

func homeHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	p, err := NewPage(ctx, "home", nil)
	if err != nil {
		fmt.Errorf("create home page: %v", err)
	}
	return p.Render(w)
}

// conferences

func scheduleConfHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	p, err := NewPage(ctx, "scheduleconf", nil)
	if err != nil {
		return fmt.Errorf("create scheduleconf page: %v", err)
	}
	return p.Render(w)
}

func confFromRequest(r *http.Request) (*conf.Conference, error) {
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

	return &conf.Conference{
		Name:         confName,
		Description:  r.FormValue("conf_desc"),
		City:         r.FormValue("city"),
		Topic:        r.FormValue("topic"),
		MaxAttendees: int(nAtt),
		TixAvailable: int(nAtt),
		StartDate:    start,
		EndDate:      end,
		Organizer:    email,
	}, nil
}

func saveConfHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	c, err := confFromRequest(r)
	if err != nil {
		return fmt.Errorf("conf from request: %v", err)
	}

	err = datastore.RunInTransaction(ctx, func(ctx appengine.Context) error {
		// Save the conference and generate the tickets
		if err := c.Save(ctx); err != nil {
			return fmt.Errorf("save conference: %v", err)
		}
		if err := c.CreateAndSaveTickets(ctx); err != nil {
			return fmt.Errorf("generate tickets: %v", err)
		}

		// Announce the conference
		a := conf.NewAnnouncement(fmt.Sprintf(
			"A new conference has just been scheduled! %s in %s. Don't wait; book now!",
			c.Name, c.City))
		if err := a.Save(ctx); err != nil {
			return fmt.Errorf("announce conference: %v", err)
		}

		// Queue a task to email interested users.
		task := taskqueue.NewPOSTTask(
			"/notifyinterestedusers",
			url.Values{"conf_id": []string{c.ID()}},
		)
		if _, err = taskqueue.Add(ctx, task, ""); err != nil {
			return fmt.Errorf("add task to default queue: %v", err)
		}

		// Queue a task to review the conference.
		task = &taskqueue.Task{
			Method:  "PULL",
			Payload: []byte(c.ID()),
		}
		task, err := taskqueue.Add(ctx, task, "review-conference-queue")
		if err != nil {
			return fmt.Errorf("add task to review queue: %v", err)
		}
		return nil
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		return err
	}

	return RedirectTo("/showtickets?conf_id=" + url.QueryEscape(c.ID()))
}

func listConfsHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	data := []*conf.ConfList{}

	for _, c := range conferenceLists {
		l, err := conf.LoadConfList(ctx, c.title, c.query)
		if err != nil {
			return err
		}
		data = append(data, l)
	}

	l, err := conf.LoadConfList(ctx,
		"All conferences About Medical Innovations in London with "+u.Email,
		datastore.NewQuery(conf.ConferenceKind).
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

func notifyInterestedUsersHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	conf, err := conf.LoadConference(ctx, r.FormValue("conf_id"))
	if err != nil {
		return fmt.Errorf("load conf: %v", err)
	}

	var body, subject bytes.Buffer
	if err := emailSubjectTmpl.Execute(&subject, conf); err != nil {
		return err
	}
	if err := emailBodyTmpl.Execute(&body, conf); err != nil {
		return err
	}

	return conf.MailNotifications(ctx, emailSender, subject.String(), body.String())
}

// admin page

func deleteAll(ctx appengine.Context, kind string) error {
	keys, err := datastore.NewQuery(kind).KeysOnly().GetAll(ctx, nil)
	if err != nil {
		return err
	}
	return datastore.DeleteMulti(ctx, keys)
}

func developerHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)

	if r.Method == "GET" {
		p, err := NewPage(ctx, "developer", nil)
		if err != nil {
			return fmt.Errorf("create developer page: %v", err)
		}
		return p.Render(w)
	}
	if r.Method != "POST" {
		return fmt.Errorf("unsupported method %v", r.Method)
	}

	switch {
	case r.FormValue("deleteall") == "yes":
		kind := r.FormValue("kind")
		keys, err := datastore.NewQuery(kind).KeysOnly().GetAll(ctx, nil)
		if err != nil {
			return fmt.Errorf("get keys to delete: %v", err)
		}
		if err := datastore.DeleteMulti(ctx, keys); err != nil {
			return fmt.Errorf("delete keys: %v", err)
		}
	case len(r.FormValue("announcement")) > 0:
		a := conf.NewAnnouncement(r.FormValue("announcement"))
		if err := a.Save(ctx); err != nil {
			return err
		}
	}
	return RedirectTo("/developer")
}

// tickets

func showTicketsHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	c, err := conf.LoadConference(ctx, r.FormValue("conf_id"))
	if err != nil {
		return fmt.Errorf("load conference: %v", err)
	}

	ts, err := c.AvailableTickets(ctx)
	if err != nil {
		return fmt.Errorf("available tickets: %v", err)
	}

	data := struct {
		ConfName string
		Tickets  []conf.Ticket
	}{c.Name, ts}

	p, err := NewPage(ctx, "tickets", data)
	if err != nil {
		return fmt.Errorf("create tickets page: %v", err)
	}
	return p.Render(w)
}

func buyTicketHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	t, err := conf.LoadTicket(ctx, r.FormValue("ticket_key_str"))
	if err != nil {
		return fmt.Errorf("load ticket: %v", err)
	}

	if err := t.SellTo(ctx, u.Email); err != nil {
		return fmt.Errorf("sell ticket: %v", err)
	}

	return RedirectTo("/userprofile")
}

// user profile

func userProfileHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	up, err := conf.LoadUserProfile(ctx, u.Email)
	if err != nil {
		return fmt.Errorf("load user profile: %v", err)
	}

	p, err := NewPage(ctx, "userprofile", up)
	if err != nil {
		return fmt.Errorf("create userprofile page: %v", err)
	}
	return p.Render(w)
}

func saveProfileHandler(w io.Writer, r *http.Request, ctx appengine.Context, u *user.User) error {
	if r.Method != "POST" {
		return RedirectTo("/userprofile")
	}
	up := conf.UserProfile{
		MainEmail:  u.Email,
		Name:       r.FormValue("person_name"),
		NotifEmail: r.FormValue("notification_email"),
		Topics:     r.Form["topics"],
	}

	if err := up.Save(ctx); err != nil {
		return fmt.Errorf("save user profile: %v", err)
	}
	return RedirectTo("/userprofile")
}

func calendarInfoHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	client, err := auth.Client(r, &urlfetch.Transport{Context: ctx}, calendarConfig)
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

	p, err := NewPage(ctx, "showcalendar", evts)
	if err != nil {
		fmt.Errorf("create showcalendar page: %v", err)
	}
	return p.Render(w)
}

// Helper types and function

type RedirectTo string

func (r RedirectTo) Error() string { return string(r) }

type handler func(io.Writer, *http.Request) error

func (f handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := &bytes.Buffer{}
	err := f(b, r)
	if err != nil {
		if red, ok := err.(RedirectTo); ok {
			http.Redirect(w, r, string(red), http.StatusMovedPermanently)
			return
		}
		msg := fmt.Sprintf("%q: request failed: %v", r.URL.Path, err)
		appengine.NewContext(r).Errorf(msg)
		http.Error(w, msg, 500)
		return
	}
	w.Write(b.Bytes())
}

type authHandler func(io.Writer, *http.Request, appengine.Context, *user.User) error

func (f authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		http.Error(w, r.URL.Path+" requires to be logged in", http.StatusForbidden)
		return
	}
	handler(func(w io.Writer, r *http.Request) error {
		return f(w, r, c, u)
	}).ServeHTTP(w, r)
}
