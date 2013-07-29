package oauth2

import (
	"net/http"
	"sync"

	"code.google.com/p/goauth2/oauth"
)

var (
	mutex    sync.RWMutex
	handlers map[string]http.HandlerFunc
)

func init() {
	handlers = make(map[string]http.HandlerFunc)
	http.HandleFunc("/oauth2callback", callback)
}

func HandlerFunc(state string, h http.Handler, config *oauth.Config) http.HandlerFunc {
	mutex.Lock()
	defer mutex.Unlock()

	handlers[state] = h.ServeHTTP

	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	}
}

func Client(r *http.Request, transport http.RoundTripper, config *oauth.Config) (*http.Client, error) {
	t := &oauth.Transport{
		Config:    config,
		Transport: transport,
	}
	_, err := t.Exchange(r.FormValue("code"))
	if err != nil {
		return nil, err
	}
	return t.Client(), nil
}

func callback(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	defer mutex.RUnlock()

	state := r.FormValue("state")
	h, ok := handlers[state]
	if !ok {
		http.Error(w, "unexpected state "+state, http.StatusBadRequest)
		return
	}
	h(w, r)
}
