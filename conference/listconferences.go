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
	http.Handle("/listconferences", handler(listConfsHandler))
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
		datastore.NewQuery("Conference"),
	},
	{
		"All Conferences Sorted Alphabetically",
		datastore.NewQuery("Conference").
			Order("Name"),
	},
	{
		"All Conferences In London sorted Alphabetically",
		datastore.NewQuery("Conference").
			Filter("City =", "London").
			Order("Name"),
	},
	{
		"All Conferences About Medical Innovations In London",
		datastore.NewQuery("Conference").
			Filter("City =", "London").
			Filter("Topic =", "Medical Innovations"),
	},
	{
		"All conferences With 50 or more attendees",
		datastore.NewQuery("Conference").
			Filter("MaxAttendees >", 50),
	},
}

func listConfsHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	data := []ConfList{}

	for _, c := range conferenceLists {
		list := ConfList{Title: c.title}
		_, err := c.query.GetAll(ctx, &list.Conferences)
		if err != nil {
			return fmt.Errorf("get %q: %v", list.Title, err)
		}
		data = append(data, list)
	}

	u := user.Current(ctx)
	if u == nil {
		return fmt.Errorf("required signed user for listconfs")
	}

	list := ConfList{
		Title: "All conferences About Medical Innovations in London with " + u.Email,
	}
	_, err := datastore.NewQuery("Conference").
		Filter("City =", "London").
		Filter("Topic =", "Medical Innovations").
		Filter("Organizer =", u.Email).
		GetAll(ctx, &list.Conferences)
	if err != nil {
		return fmt.Errorf("get %q: %v", list.Title, err)
	}
	data = append(data, list)

	p, err := NewPage(ctx, "listconfs", data)
	if err != nil {
		return fmt.Errorf("create listconfs page: %v", err)
	}
	return p.Render(w)
}
