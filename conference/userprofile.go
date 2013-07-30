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
	http.Handle("/userprofile", handler(userProfileHandler))
	http.Handle("/saveprofile", handler(saveProfileHandler))
}

type UserProfile struct {
	RegisteredConfs []Conference
	Name            string
	Topics          []string
	MainEmail       string
	NotifEmail      string
}

func (u UserProfile) SelectedTopic(topic string) bool {
	for _, t := range u.Topics {
		if t == topic {
			return true
		}
	}
	return false
}

func userProfileHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)

	u := user.Current(ctx)
	if u == nil {
		return fmt.Errorf("required login for userprofile")
	}

	var data UserProfile

	heading := "Edit your"

	k := datastore.NewKey(ctx, "RegisteredUser", u.Email, 0, nil)
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

func saveProfileHandler(w io.Writer, r *http.Request) error {
	if r.Method != "POST" {
		return RedirectTo("/userprofile")
	}

	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		return fmt.Errorf("required login for userprofile")
	}

	up := UserProfile{
		MainEmail:  u.Email,
		Name:       r.FormValue("person_name"),
		NotifEmail: r.FormValue("notification_email"),
		Topics:     r.Form["topics"],
	}

	k := datastore.NewKey(ctx, "RegisteredUser", u.Email, 0, nil)
	if _, err := datastore.Put(ctx, k, &up); err != nil {
		return fmt.Errorf("save userprofile: %v", err)
	}
	return RedirectTo("/userprofile")
}
