// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"bytes"
	"io"
	"net/http"

	"appengine"
	"appengine/user"
)

var (
	topicList = []string{
		"Medical Innovations",
		"Programming Languages",
		"Web Technologies",
		"Movie Making",
	}
	cityList = []string{
		"London",
		"Chicago",
		"San Francisco",
		"Paris",
	}
)

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
		appengine.NewContext(r).Errorf("request failed: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(b.Bytes())
}

type authHandler func(io.Writer, *http.Request, appengine.Context, *user.User) error

func (f authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := &bytes.Buffer{}
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		http.Error(w, r.URL.Path+" requires to be logged in", http.StatusForbidden)
		return
	}
	err := f(b, r, c, u)
	if err != nil {
		if red, ok := err.(RedirectTo); ok {
			http.Redirect(w, r, string(red), http.StatusMovedPermanently)
			return
		}
		appengine.NewContext(r).Errorf("request failed: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(b.Bytes())
}
