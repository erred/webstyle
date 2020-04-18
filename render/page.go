package render

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday/v2"
	"sigs.k8s.io/yaml"
)

type Page struct {
	// passthrough, don't process
	pass bool
	name string
	data []byte

	// Optional yaml front matter,
	// also supplied to ExecuteTemplate
	Date        string
	Description string
	Header      string
	Style       string
	Title       string

	// filled, supplied to ExecuteTemplate
	GoogleAnalytics string // GA ID
	Main            string // html content
	URLAbsolute     string // start from /
	URLBase         string // https://... without trailing /
	URLCanonical    string // URLBase + URLAbsolute
	URLLogger       string
}

// NewPage reates a new page from filename and file contents
// if data starts with `---` it is assumed to be followed by
// a yaml front matter then markdown
func NewPage(name string, data []byte, pass bool) (*Page, error) {
	p := Page{
		pass: pass,
		data: data,
		name: name,
	}
	if !pass && bytes.HasPrefix(data, []byte(`---`)) {
		parts := bytes.SplitN(p.data, []byte(`---`), 3)
		err := yaml.Unmarshal(parts[1], &p)
		if err != nil {
			return nil, err
		}
		p.data = parts[2]
	}
	if filepath.Ext(p.name) == ".md" {
		p.name = strings.TrimSuffix(p.name, "md") + "html"
		p.Main = string(blackfriday.Run(
			p.data,
			blackfriday.WithRenderer(
				blackfriday.NewHTMLRenderer(
					blackfriday.HTMLRendererParameters{
						Flags: blackfriday.CommonHTMLFlags,
					},
				),
			),
		))
	}
	return &p, nil
}
