// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"fmt"
	"io"
	"net/http"

	"appengine"
)

func init() {
	http.Handle("/", handler(homeHandler))
}

func homeHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	p, err := NewPage(ctx, "home", nil)
	if err != nil {
		fmt.Errorf("create home page: %v", err)
	}
	return p.Render(w)
}
