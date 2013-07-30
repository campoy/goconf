// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"bytes"
	"html/template"
	"io"
	"time"

	"appengine"
	"appengine/user"
)

var (
	tmpl = template.Must(
		template.New("base").
			Funcs(template.FuncMap{
			"exec": execTemplate,
			"date": func(d time.Time) string { return d.Format("2006 Jan 2") },
		}).ParseGlob("templates/*.tmpl"))
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
	Base         *template.Template
	Content      string
	User         *User
	LogoutURL    string
	LoginURL     string
	Heading      string
	Topics       []string
	Cities       []string
	Announcement string
	Data         interface{}
}

func NewPage(ctx appengine.Context, name string, data interface{}) (*Page, error) {
	ann, err := GetLatestAnnouncement(ctx)
	if err != nil {
		ctx.Errorf("latest announcement: %v", err)
	}

	p := &Page{
		Base:         tmpl,
		Content:      name,
		Data:         data,
		Topics:       topicList,
		Cities:       cityList,
		Announcement: ann,
	}

	if u := user.Current(ctx); u != nil {
		p.User = &User{User: u}
		p.LogoutURL, err = user.LogoutURL(ctx, "/")
	} else {
		p.LoginURL, err = user.LoginURL(ctx, "/")
	}

	return p, err
}

func (p *Page) Render(w io.Writer) error {
	return tmpl.Execute(w, p)
}
