package main

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed static
var embededFiles embed.FS

//go:embed templates
var templateFS embed.FS

func mustTemplate(useOs bool, name string) (tpl *template.Template) {
	var currFS fs.FS
	var err error
	templateDir := "templates"

	if useOs {
		currFS = os.DirFS(".")
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		currFS = templateFS
	}

	trimI := strings.Index(name, templateDir) + len(templateDir) + 1
	tpl, err = template.New(name[trimI:]).Funcs(template.FuncMap{
		"prettyDate": func(dt string) template.HTML {
			parse, err := time.Parse(time.RFC3339, dt)
			if err != nil {
				return template.HTML("Unknown Date (" + dt + ")")
			}
			datestring := fmt.Sprintf("<time datetime='%s' > %s </time>", dt, parse.Format(time.RFC822))
			return template.HTML(datestring)
		},
	}).ParseFS(currFS, name)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println(tpl.DefinedTemplates(), "attempted lookup:", name[trimI:])

	return tpl
}

func getFileSystem(useOS bool, embedfs embed.FS) http.FileSystem {
	if useOS {
		log.Println("Using live fs:")
		return http.FS(os.DirFS("static"))
	}
	fsys, err := fs.Sub(embedfs, "static")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

func connectDb() *sql.DB {
	p := os.Getenv("DB")
	if p == "" {
		p = "./data.sqlite"
	}
	_, err := os.Open(p)
	db, err := sql.Open("sqlite3", p)
	if err != nil {
		panic(err)
	}
	return db
}

func printrouter(nr *mux.Router) {
	_ = nr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			fmt.Println("ROUTE:", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			fmt.Println("Path regexp:", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			fmt.Println("Queries templates:", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			fmt.Println("Queries regexps:", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			fmt.Println("Methods:", strings.Join(methods, ","))
		}
		fmt.Println()
		return nil
	})
}
