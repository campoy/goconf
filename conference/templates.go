package conference

import (
	"bytes"
	"html/template"
	"io"

	"appengine"
	"appengine/user"
)

var (
	tmpl = template.Must(
		template.New("base").
			Funcs(template.FuncMap{"exec": execTemplate}).
			ParseGlob("templates/*.tmpl"))
)

// execTemplate is a helper to execute a template and return the output as a
// template.HTML value.
func execTemplate(t *template.Template, name string, data interface{}) (template.HTML, error) {
	b := new(bytes.Buffer)
	err := t.ExecuteTemplate(b, name, data)
	if err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

type User struct {
	*user.User
	Name string
}

type Page struct {
	Base      *template.Template
	Content   string
	User      *User
	LogoutURL string
	LoginURL  string
	Data      interface{}
}

func newPage(ctx appengine.Context, name string, data interface{}) (p *Page, err error) {
	p = &Page{
		Base:    tmpl,
		Content: name,
		Data:    data,
	}
	if u := user.Current(ctx); u != nil {
		p.User = &User{User: u}
		p.LogoutURL, err = user.LogoutURL(ctx, "/")
	} else {
		p.LoginURL, err = user.LoginURL(ctx, "/")
	}
	return p, err
}

func renderPage(ctx appengine.Context, w io.Writer, name string, data interface{}) error {
	p, err := newPage(ctx, name, data)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, p)
}
