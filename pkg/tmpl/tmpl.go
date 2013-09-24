// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD style
// license that can be found in the LICENSE file.

// The tmpl package allows the user to use the include function in its templates,
// which executes a template given its name and some data.
// It also provides a date formatting function named date.
package tmpl

import (
	"bytes"
	"html/template"
	"io"
	"time"
)

var tmpl *template.Template

func init() {
	// not initialized at variable declaration to avoid initialization loop with
	// execTemplate.
	tmpl = template.New("base").
		Funcs(template.FuncMap{
		"include": execTemplate,
		"date":    dateFmt,
	})
}

// execTemplate is a helper to execute a template and return the output as a
// template.HTML value.
func execTemplate(name string, data interface{}) (template.HTML, error) {
	b := new(bytes.Buffer)
	err := tmpl.ExecuteTemplate(b, name, data)
	if err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

func dateFmt(d time.Time) string {
	return d.Format("2006 Jan 2")
}

// ParseTemplates parses all the templates matching the given file pattern.
// An error is returned if the parsing fails.
func ParseTemplates(pattern string) error {
	_, err := tmpl.ParseGlob(pattern)
	return err
}

// Execute executes the parsed templates on the given data and writes the output
// to the given writer.
func Execute(w io.Writer, data interface{}) error {
	return tmpl.Execute(w, data)
}
