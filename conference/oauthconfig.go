// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package conference

import (
	"appengine"

	"code.google.com/p/goauth2/oauth"
)

func config(scope string) *oauth.Config {
	redirect := "https://go-conf.appspot.com/oauth2callback"
	if appengine.IsDevAppServer() {
		redirect = "http://localhost:8080/oauth2callback"
	}

	return &oauth.Config{
		ClientId:     "client-id: you should change this",
		ClientSecret: "client-secret: change this too",
		Scope:        scope,
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		RedirectURL:  redirect,
	}
}
