package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/PuerkitoBio/goquery"
	"github.com/avelino/awesome-go/pkg/slug"
)

type Link struct {
	Title       string
	Url         string
	Description string
}

type Object struct {
	Title       string
	Slug        string
	Description string
	Items       []Link
}

func main() {
	err := GenerateHTML()
	if err != nil {
		panic(err)
	}
	input, err := os.ReadFile("./tmpl/index.html")
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(input)
	query, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		panic(err)
	}

	objs := make(map[string]*Object)
	query.Find("body #content ul ul").First().Each(func(_ int, s *goquery.Selection) {
		s.Find("li a").Each(func(_ int, s *goquery.Selection) {
			selector, exists := s.Attr("href")
			if !exists {
				return
			}
			obj := makeObjById(selector, query.Find("body"))
			if obj == nil {
				return
			}
			objs[selector] = obj
		})
	})

	makeCategoryPages(objs)
	linkCategoryPagesInIndex(string(input), query, objs)

	makeSitemap(objs)
}

func makeCategoryPages(objs map[string]*Object) {
	for _, obj := range objs {
		folder := fmt.Sprintf("tmpl/%s", obj.Slug)
		err := os.Mkdir(folder, 0755)
		if err != nil {
			log.Println(err)
		}

		t := template.Must(template.ParseFiles("tmpl/cat-tmpl.html"))
		f, _ := os.Create(fmt.Sprintf("%s/index.html", folder))
		t.Execute(f, obj)
	}
}

func makeSitemap(objs map[string]*Object) {
	t := template.Must(template.ParseFiles("tmpl/sitemap-tmpl.xml"))
	f, _ := os.Create("tmpl/sitemap.xml")
	t.Execute(f, objs)
}

func makeObjById(selector string, s *goquery.Selection) (obj *Object) {
	s.Find(selector).Each(func(_ int, s *goquery.Selection) {
		desc := s.NextFiltered("p")
		ul := s.NextFilteredUntil("ul", "h2")

		links := []Link{}
		ul.Find("li").Each(func(_ int, s *goquery.Selection) {
			url, _ := s.Find("a").Attr("href")
			link := Link{
				Title:       s.Find("a").Text(),
				Description: s.Text(),
				Url:         url,
			}
			links = append(links, link)
		})
		obj = &Object{
			Slug:        slug.Generate(s.Text()),
			Title:       s.Text(),
			Description: desc.Text(),
			Items:       links,
		}
	})
	return
}

func linkCategoryPagesInIndex(html string, query *goquery.Document, objs map[string]*Object) {
	query.Find("body #content ul li ul li a").Each(func(_ int, s *goquery.Selection) {
		href, hrefExists := s.Attr("href")
		if !hrefExists {
			return
		}

		// do not replace links if no page has been created for it
		_, objExists := objs[href]
		if !objExists {
			return
		}

		uri := strings.SplitAfter(href, "#")
		if len(uri) >= 2 && uri[1] != "contents" {
			html = strings.ReplaceAll(
				html, fmt.Sprintf(`href="%s"`, href), fmt.Sprintf(`href="%s"`, uri[1]))
		}
	})

	os.WriteFile("./tmpl/index.html", []byte(html), 0644)
}
