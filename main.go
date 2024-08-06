package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const dataDir string = "data"
const tmplDir string = "tmpl"

// list of templates
var templates = template.Must(template.ParseFiles(
	filepath.Join(tmplDir, "edit.html"),
	filepath.Join(tmplDir, "view.html"),
	filepath.Join(tmplDir, "create.html"),
	filepath.Join(tmplDir, "index.html"),
))

type Page struct {
	Title string
	Body  []byte
}

type Template struct {
	Title string
	Body  []byte
}

//const dirPerms int = 0700

// save function
func (p *Page) save() error {
	// save page title into dataDir
	filename := filepath.Join(dataDir, p.Title+".txt")
	// if dataDir DNE then create w/ 0600 permissions
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.Mkdir(dataDir, 0700)
	}
	return os.WriteFile(filename, p.Body, 0700)
}

// loads pages from data dir
func loadPage(title string) (*Page, error) {
	filename := filepath.Join(dataDir, title+".txt")
	body, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		log.Printf("error loading page %q: %s", filename, err)
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

// main page is the index of website
// can't get /.../ to work without /.../...
// "page doesn't exist"
// maybe find a way to load from tmpl folder
// func index(title string) (*Template, error) {
// 	filename := filepath.Join(tmplDir, "index.html")
// 	body, err := os.ReadFile(filepath.Clean(filename))
// 	if err != nil {
// 		log.Printf("error loading index %q: %s", filename, err)
// 	}
// 	return &Template{Title: title, Body: body}, nil

// p, err := loadPage(title)
// if err != nil {
// 	// os.Create("dir")
// 	// http.Redirect(w, r, "/index/", http.StatusFound)
// 	return
// }
// renderTemplate(w, "index", p)
// }

// redirects to front page if user tries to view nonexistent page
func frontpageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "", http.StatusFound)
}

// creation handler for new pages
func creationHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "create", p)
}

// subdirectory for viewing of pages
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		// redirects a page that does not exist to edit & create it
		// don't want this for anyone.
		// http.Redirect(w, r, "/create/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

// allows for editing of a page, not creation
func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// allows for saving input of a page when editing
func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// calls the correct template based on URL
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view|index|create)/([a-zA-Z0-9]+)$")

// makes and runs the handler (view, edit, save, etc.), checks to see if the path is valid
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])

		fmt.Println(m[:])
	}
}

func main() {
	// http.HandleFunc("/", frontpageHandler)
	// http.HandleFunc("/index/", makeHandler(index))
	http.HandleFunc("/create/", makeHandler(creationHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Fatal(http.ListenAndServe(":8080", http.FileServer(http.Dir("/Users/saxon/vscode/web_app_personal/tmpl"))))
}
