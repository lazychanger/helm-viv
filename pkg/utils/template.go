package utils

import (
	"io"
	"text/template"
)

// see https://github.com/spf13/cobra/blob/main/command.go#L864
func Tmpl(w io.Writer, text string, data interface{}) error {
	t := template.New("top")

	t.Funcs(funcMap())

	return template.Must(t.Parse(text)).Execute(w, data)
}
