// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"fmt"
	"io"
	"net/http"

	"appengine"
	"appengine/datastore"
)

func init() {
	http.Handle("/developer", handler(developerHandler))
}

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
	if r.FormValue("deleteall") == "yes" {
		err := deleteAll(ctx, r.FormValue("kind"))
		if err != nil {
			return err
		}
		return RedirectTo("/developer")
	}
	if ann := r.FormValue("announcement"); len(ann) > 0 {
		err := SetLatestAnnouncement(ctx, ann)
		if err != nil {
			return err
		}
	}
	return RedirectTo("/developer")
}
