package render

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"golang.org/x/tools/blog/atom"
)

var (
	defaultFeed = atom.Feed{
		Title: "author's web log",
		ID:    "tag:seankhliao.com,2020:seankhliao.com",
		Link: []atom.Link{
			{
				Rel:  "self",
				Href: "https://seankhliao.com/feed.atom",
				Type: "application/atom+xml",
			}, {
				Rel:  "alternate",
				Href: "https://seankhliao.com/blog/",
				Type: "text/html",
			},
		},
		Updated: atom.Time(time.Now()),
		Author: &atom.Person{
			Name:  "Sean Liao",
			URI:   "https://seankhliao.com/",
			Email: "blog-atom@seankhliao.com",
		},
	}
)

type Options struct {
	In              string
	Out             string
	Template        *template.Template
	GoogleAnalytics string
	URLBase         string
	URLLogger       string
}

func (o *Options) InitFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.In, "in", "src", "input directory")
	fs.StringVar(&o.Out, "out", "public", "output directory")
	fs.StringVar(&o.GoogleAnalytics, "ga", "UA-114337586-1", "google analytics id")
	fs.StringVar(&o.URLBase, "base", "https://seankhliao.com", "base url")
	fs.StringVar(&o.URLLogger, "logger", "https://statslogger.seankhliao.com/form", "statslogger url")
}

func Process(o Options) error {
	pages, err := processInput(o)
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}
	if len(pages) > 1 {
		pages, err = processFill(pages, o.Out)
		if err != nil {
			return fmt.Errorf("process: %w", err)
		}
	}
	err = processOutput(o, pages)
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}
	return nil
}

func processInput(o Options) ([]*Page, error) {
	info, err := os.Stat(o.In)
	if err != nil {
		return nil, fmt.Errorf("stat file %s: %w", o.In, err)
	}
	var pages []*Page
	if !info.IsDir() {
		page, err := NewPageFromFile(o.In)
		if err != nil {
			return nil, fmt.Errorf("process input file %s: %w", o.In, err)
		}
		pages = []*Page{page}
	} else {
		err = filepath.Walk(o.In, walker(o.In, &pages))
		if err != nil {
			return nil, fmt.Errorf("process input dir %s: %w", o.In, err)
		}
	}
	for i := range pages {
		pages[i].GoogleAnalytics = o.GoogleAnalytics
		pages[i].URLBase = o.URLBase
		pages[i].URLLogger = o.URLLogger
		pages[i].URLAbsolute = canonical(strings.TrimPrefix(pages[i].name, o.In))
		pages[i].URLCanonical = o.URLBase + pages[i].URLAbsolute
		if pages[i].name != o.In {
			r, err := filepath.Rel(o.In, pages[i].name)
			if err == nil {
				pages[i].name = filepath.Join(o.Out, r)
			}
		}
	}
	return pages, nil
}

func processFill(pages []*Page, out string) ([]*Page, error) {
	sort.Slice(pages, func(i, j int) bool { return pages[i].name > pages[j].name })

	feed, blogindex, buf := defaultFeed, 0, strings.Builder{}
	buf.WriteString("<ul>\n")
	for i, p := range pages {
		if strings.Contains(p.name, "/blog/") {
			if filepath.Base(p.name) != "index.html" {
				pages[i].Date = filepath.Base(p.name)[:10]
				pages[i].Header = blogHeader(pages[i].Date)
				buf.WriteString(blogLink(p.Date, p.URLAbsolute, p.Title))
				feed.Entry = append(feed.Entry, blogEntry(p, feed.Author))
			} else {
				blogindex = i
			}
		}
		if filepath.Ext(p.name) == ".html" {
			pages[i].Main = imgHack(p.Main)
		}
	}

	buf.WriteString("</ul>\n")
	pages[blogindex].Main = buf.String()
	pages[blogindex].Header = blogIndexHeader()

	// create atom
	p, err := atomPage(feed, out)
	if err != nil {
		return nil, fmt.Errorf("create atom: %w", err)
	}
	pages = append(pages, p)

	// create sitemap
	all := make([][]byte, len(pages))
	for i := range all {
		all[i] = []byte(pages[i].URLCanonical)
	}
	p, err = NewPage(filepath.Join(out, "sitemap.txt"), bytes.Join(all, []byte("\n")), true)
	if err != nil {
		return nil, fmt.Errorf("fill sitemap.txt: %w", err)
	}
	pages = append(pages, p)

	return pages, nil
}

func processOutput(o Options, pages []*Page) error {
	for _, p := range pages {
		os.MkdirAll(filepath.Dir(p.name), 0o755)
		f, err := os.Create(p.name)
		if err != nil {
			return fmt.Errorf("create file %s: %w", p.name, err)
		}
		if p.pass {
			_, err = f.Write(p.data)
		} else {
			err = o.Template.ExecuteTemplate(f, "LayoutGohtml", p)
		}
		f.Close()
		if err != nil {
			return fmt.Errorf("write file %s: %w", p.name, err)
		}
	}
	return nil
}

func walker(base string, pt *[]*Page) func(p string, i os.FileInfo, err error) error {
	pages := *pt
	return func(p string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if i.IsDir() {
			return nil
		}
		page, err := NewPageFromFile(p)
		if err != nil {
			return fmt.Errorf("walk %s: %w", p, err)
		}
		pages = append(pages, page)
		*pt = pages
		return nil
	}
}

func canonical(p string) string {
	if p[0] != '/' {
		p = "/" + p
	}
	if strings.HasSuffix(p, ".html") {
		p = strings.TrimSuffix(strings.TrimSuffix(p, ".html"), "index")
		if p == "" {
			p = "/"
		}
		if p[len(p)-1] != '/' {
			p = p + "/"
		}
	}
	return p
}

func atomPage(feed atom.Feed, out string) (*Page, error) {
	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	e.Indent("", "\t")
	err := e.Encode(feed)
	if err != nil {
		return nil, fmt.Errorf("fill encode atom: %w", err)
	}
	p, err := NewPage(filepath.Join(out, "feed.atom"), buf.Bytes(), true)
	if err != nil {
		return nil, fmt.Errorf("fill feed.atom: %w", err)
	}
	return p, nil
}

func blogIndexHeader() string {
	return `<h2><a href="/blog/">b<em>log</em></a></h2>
<p>Artisanal, <em>hand-crafted</em> blog posts imbued with delayed <em>regrets</em></p>`
}

func blogHeader(date string) string {
	return fmt.Sprintf(`<h2><a href="/blog/">b<em>log</em></a></h2>
<p><time datetime="%s">%s</time></p>`, date, date)
}

func blogLink(date, urlabsolute, title string) string {
	return fmt.Sprintf(`<li><time datetime="%s">%s</time> | <a href="%s">%s</a></li>`+"\n",
		date, date, urlabsolute, title)
}

func blogEntry(p *Page, author *atom.Person) *atom.Entry {
	return &atom.Entry{
		Title: p.Title,
		Link: []atom.Link{
			{Rel: "alternate", Href: p.URLCanonical, Type: "text/html"},
		},
		ID:        p.URLCanonical,
		Published: atom.TimeStr(p.Date + "T00:00:00Z"),
		Updated:   atom.TimeStr(p.Date + "T00:00:00Z"),
		Author:    author,
		Summary:   &atom.Text{Type: "text", Body: p.Title},
	}
}

func imgHack(html string) string {
	r := regexp.MustCompile(`<h4><img src="(.*?).webp" alt="(.*?)" /></h4>`)
	return r.ReplaceAllString(html, `
<picture>
        <source type="image/webp" srcset="$1.webp">
        <source type="image/jpeg" srcset="$1.jpg">
        <img src="$1.png" alt="$2">
</picture>
`)
}
