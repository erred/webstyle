// +build generate

package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func main() {
	t := template.Must(template.New("").Parse(tmpl))

	var files []string
	for _, g := range os.Args[2:] {
		f, err := filepath.Glob(g)
		if err != nil {
			log.Fatal("glob: ", g, ": ", err)
		}
		files = append(files, f...)
	}

	r := strings.NewReplacer("-", "", " ", "", ".", "")
	fs := make([]File, 0, len(files))
	for _, f := range files {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal("readfile: ", f, ": ", err)
		}
		b = bytes.ReplaceAll(b, []byte("`"), []byte("`+\"`\"+`"))
		fs = append(fs, File{
			Name:    r.Replace(strings.Title(filepath.Base(f))),
			Content: string(b),
		})
	}

	f, err := os.Create("webstyle.go")
	if err != nil {
		log.Fatal("create webstyle.go: ", err)
	}
	defer f.Close()
	err = t.Execute(f, Data{fs})
	if err != nil {
		log.Fatal("execute: ", err)
	}
}

type Data struct {
	Files []File
}
type File struct {
	Name    string
	Content string
}

var tmpl = `// Code generated by generate.go DO NOT EDIT.
package webstyle

const (
        {{ range .Files }}
        {{ .Name }} = ` + "`" + `{{"{{"}} define "{{ .Name }}" {{"}}"}}
{{ .Content }}
{{"{{"}} end {{"}}"}}` + "`" + `
        {{ end }}
)
`
