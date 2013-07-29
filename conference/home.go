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

type handler func(io.Writer, *http.Request) error

func (f handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := &bytes.Buffer{}
	err := f(b, r)
	if err != nil {
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
