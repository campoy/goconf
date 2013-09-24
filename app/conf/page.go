package conf

import (
	"io"

	"appengine"
	"appengine/user"

	"github.com/campoy/goconf/pkg/conf"
	"github.com/campoy/goconf/pkg/tmpl"
)

// A Page contains the information displayed by the home template.
type Page struct {
	Content string      // Name of the embedded template
	Data    interface{} // Data for the embedded template

	User         *user.User
	LogoutURL    string
	LoginURL     string
	Topics       []string
	Cities       []string
	Announcement string
}

// NewPage returns a new Page initialized embedding the template with the
// given name and data, the current user for the given context, and the
// latest announcement.
func NewPage(ctx appengine.Context, name string, data interface{}) (*Page, error) {
	p := &Page{
		Content: name,
		Data:    data,
		Topics:  topicList,
		Cities:  cityList,
	}

	a, err := conf.LatestAnnouncement(ctx)
	if err != nil {
		ctx.Errorf("latest announcement: %v", err)
	}
	if a != nil {
		p.Announcement = a.Message
	}

	if u := user.Current(ctx); u != nil {
		p.User = u
		p.LogoutURL, err = user.LogoutURL(ctx, "/")
	} else {
		p.LoginURL, err = user.LoginURL(ctx, "/")
	}

	return p, err
}

// Render executes the home template passing the data on the page.
func (p *Page) Render(w io.Writer) error {
	return tmpl.Execute(w, p)
}
