package render

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"golang.org/x/tools/blog/atom"
)

type Options struct {
	In              string
	Out             string
	Template        *template.Template
	GoogleAnalytics string
	URLBase         string
	URLLogger       string

	SingleFile bool
}

func (o *Options) InitFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.In, "in", "src", "input directory")
	fs.StringVar(&o.Out, "out", "public", "output directory")
	fs.StringVar(&o.GoogleAnalytics, "ga", "UA-114337586-1", "google analytics id")
	fs.StringVar(&o.URLBase, "base", "https://seankhliao.com", "base url")
	fs.StringVar(&o.URLLogger, "logger", "https://statslogger.seankhliao.com/api", "statslogger url")
	fs.BoolVar(&o.SingleFile, "single", false, "single file mode")
}

func Process(o Options) error {
	pages, err := processInput(o)
	if err != nil {
		return fmt.Errorf("Process input: %w", err)
	}
	pages, err = processFill(o, pages)
	if err != nil {
		return fmt.Errorf("Process fill: %w", err)
	}
	err = processOutput(o, pages)
	if err != nil {
		return fmt.Errorf("Process output: %w", err)
	}
	return nil
}

func processFill(o Options, pages []*Page) ([]*Page, error) {
	for i := range pages {
		pages[i].GoogleAnalytics = o.GoogleAnalytics
		pages[i].URLBase = o.URLBase
		pages[i].URLLogger = o.URLLogger
		pages[i].URLAbsolute = canonical(pages[i].name)
		pages[i].URLCanonical = o.URLBase + pages[i].URLAbsolute
	}

	if o.SingleFile {
		return pages, nil
	}

	sort.Slice(pages, func(i, j int) bool { return pages[i].name > pages[j].name })

	// atom
	feed := atom.Feed{
		Title: "author's web log",
		ID:    "tag:seankhliao.com,2020:seankhliao.com",
		Link: []atom.Link{
			{
				Rel:  "self",
				Href: "https://seankhliao.com/feed.atom",
				Type: "application/atom+xml",
			}, {
				Rel:  "alternate",
				Href: "https://seankhliao.com/blog",
				Type: "text/html",
			},
		},
		Updated: atom.Time(time.Now()),
		Author: &atom.Person{
			Name:  "Sean Liao",
			URI:   "https://seankhliao.com",
			Email: "blog-atom@seankhliao.com",
		},
	}

	// blog
	var blogindex int
	var buf strings.Builder
	buf.WriteString("<ul>\n")
	for i, p := range pages {
		if strings.HasPrefix(p.name, "blog") {
			if p.name != "blog/index.html" {
				pages[i].Header = fmt.Sprintf(`<h2><a href="/blog/" ping="%s?trigger=ping&src=%s&dst=%s">b<em>log</em></a></h2>`, o.URLLogger, p.URLCanonical, "/blog/") +
					"\n" + fmt.Sprintf(`<p><time datetime="%s">%s</time></p>`, p.Date, p.Date)
				buf.WriteString(fmt.Sprintf(`<li><time datetime="%s">%s</time> | <a href="%s" ping="%s?trigger=ping&src=%s&dst=%s">%s</a></li>`+"\n", p.Date, p.Date, p.URLAbsolute, o.URLLogger, "/blog/", p.URLCanonical, p.Title))

				feed.Entry = append(feed.Entry, &atom.Entry{
					Title:     p.Title,
					Link:      []atom.Link{{Rel: "alternate", Href: p.URLCanonical, Type: "text/html"}},
					ID:        p.URLCanonical,
					Published: atom.TimeStr(p.Date + "T00:00:00Z"),
					Updated:   atom.TimeStr(p.Date + "T00:00:00Z"),
					Author:    feed.Author,
					Summary:   &atom.Text{Type: "text", Body: p.Title},
				})
			} else {
				blogindex = i
			}
		}
		if filepath.Ext(p.name) == ".html" {
			pages[i].Main = imgHack(linkHack(p.Main, o.URLLogger, p.URLCanonical))
		}
	}
	buf.WriteString("</ul>\n")
	pages[blogindex].Main = buf.String()
	pages[blogindex].Header = fmt.Sprintf(`<h2><a href="/blog/" ping="%s?trigger=ping&src=%s&dst=%s">b<em>log</em></a></h2>`, o.URLLogger, o.URLBase+"/blog/", "/blog/") +
		"\n" + `<p>Artisanal, <em>hand-crafted</em> blog posts imbued with delayed <em>regrets</em></p>`

	// create atom
	var bb bytes.Buffer
	xenc := xml.NewEncoder(&bb)
	xenc.Indent("", "\t")
	err := xenc.Encode(feed)
	if err != nil {
		return nil, fmt.Errorf("encode atom: %w", err)
	}
	p, err := NewPage("feed.atom", bb.Bytes(), true)
	if err != nil {
		return nil, fmt.Errorf("fill feed.atom: %w", err)
	}
	pages = append(pages, p)

	// create sitemap
	all := make([][]byte, len(pages))
	for i := range all {
		all[i] = []byte(pages[i].URLCanonical)
	}
	p, err = NewPage("sitemap.txt", bytes.Join(all, []byte("\n")), true)
	if err != nil {
		return nil, fmt.Errorf("fill sitemap.txt: %w", err)
	}
	pages = append(pages, p)

	return pages, nil
}

func processInput(o Options) ([]*Page, error) {
	var pages []*Page
	if o.SingleFile {
		i, err := os.Stat(o.In)
		if err != nil {
			return nil, fmt.Errorf("single file mode %s: %w", o.In, err)
		}
		if i.IsDir() {
			return nil, fmt.Errorf("single file mode unexpected directory")
		}
		err = walker(o.In, &pages)(o.In, i, nil)
		if err != nil {
			return nil, fmt.Errorf("single file mode %s: %w", o.In, err)
		}
		return pages, nil
	}

	err := filepath.Walk(o.In, walker(o.In, &pages))
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func processOutput(o Options, pages []*Page) error {
	var wg sync.WaitGroup
	errc := make(chan error, len(pages))
	wg.Add(len(pages))

	for _, p := range pages {
		if !o.SingleFile {
			os.MkdirAll(filepath.Dir(filepath.Join(o.Out, p.name)), 0755)
		}
		go func(p *Page) {
			defer wg.Done()
			fo, err := os.Create(filepath.Join(o.Out, p.name))
			if err != nil {
				errc <- fmt.Errorf("create %s: %w", p.name, err)
				return
			}
			defer fo.Close()

			if p.pass {
				_, err = fo.Write(p.data)
				if err != nil {
					errc <- fmt.Errorf("write %s: %w", p.name, err)
				}
			} else {
				err = o.Template.ExecuteTemplate(fo, "LayoutGohtml", p)
				if err != nil {
					errc <- fmt.Errorf("execute %s: %w", p.name, err)
					return
				}
			}
		}(p)
	}

	wg.Wait()
	close(errc)
	var errs []error
	for e := range errc {
		errs = append(errs, e)
	}
	if len(errs) != 0 {
		return fmt.Errorf("output: %v", errs)
	}
	return nil
}

func walker(in string, ppages *[]*Page) func(p string, i os.FileInfo, err error) error {
	pages := *ppages
	return func(p string, i os.FileInfo, err error) error {
		if i.IsDir() || err != nil {
			return err
		}
		sp, err := filepath.Rel(in, p)
		if err != nil {
			return fmt.Errorf("rel: %v", err)
		}
		var pass bool
		if filepath.Ext(p) != ".md" {
			pass = true
		}
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		page, err := NewPage(sp, b, pass)
		if err != nil {
			return fmt.Errorf("process %s: %w", p, err)
		}
		pages = append(pages, page)
		*ppages = pages
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

func linkHack(html, log, src string) string {
	r := regexp.MustCompile(`<a href="(.*?)">`)
	return r.ReplaceAllString(html, fmt.Sprintf(`<a href="$1" ping="%s?trigger=ping&src=%s&dst=$1">`, log, src))
}
