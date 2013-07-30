package conference

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

func init() {
	http.Handle("/scheduleconference", handler(scheduleConfHandler))
	http.Handle("/saveconference", handler(saveConfHandler))
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

type Ticket struct {
	Number   int64
	Status   TicketStatus
	ConfName string
	ConfKey  string
	Owner    string
}

type TicketStatus string

const (
	TicketAvailable TicketStatus = "available"
	TicketSoldOut   TicketStatus = "soldout"
)

func saveConfHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		return fmt.Errorf("required login for saveconf")
	}

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

	k := datastore.NewKey(ctx, "Conference", "", 0, nil)
	k, err = datastore.Put(ctx, k, &conf)
	if err != nil {
		return fmt.Errorf("save conference: %v", err)
	}
	conf.Key = k.Encode()
	_, err = datastore.Put(ctx, k, &conf)
	if err != nil {
		return fmt.Errorf("save conference with key: %v", err)
	}
	/*
		for i := 0; i < nAtt; i++ {
			ticket := Ticket{
				TicketNumber:   int64(i),
				Status:         TicketAvailable,
				ConferenceName: confName,
				ConferenceKey:  k.Encode(),
			}
			tk := datastore.NewKey(ctx, "Ticket", "", 0, k)
			_, err := datastore.Put(ctx, tk)
			if err != nil {
				return fmt.Errorf("save ticket %v for conference %v: %v", i, confName, err)
			}
		}
	*/

	return RedirectTo("/listconferences")
}
