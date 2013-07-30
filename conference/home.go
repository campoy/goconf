// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"bytes"
	"io"
	"net/http"

	"appengine"
)

func init() {
	http.Handle("/", handler(homeHandler))
}

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

func homeHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	return renderPage(ctx, w, "home", nil)
}
