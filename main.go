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
	// filepath.Join(tmplDir, "index.html"),
))

type Page struct {
	Title string
	Body  []byte
}

// func indexServer() {

// }

// save function.
// grabs filename and data from Page struct, places it into /data/ and adds .txt
func (p *Page) save() error {
	filename := filepath.Join(dataDir, p.Title+".txt")
	// if dataDir DNE then create w/ 0600 permissions
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.Mkdir(dataDir, 0600)
	}
	return os.WriteFile(filename, p.Body, 0600)
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

// check if page exists, if it doesn't, render create template
func pageCheck(title string) (*Page, error) {
	p, err := loadPage(title)
	if err != nil {
		return p, err
	}
	return p, err

}

func pageRedirect(w http.ResponseWriter, r *http.Request) {
	log.Println("problem loading page", http.StatusInternalServerError)
	http.Redirect(w, r, "/", http.StatusFound)
	return
}

// redirects to front page if user tries to view nonexistent page
// doesn't actually redirect right now because of infinite redirect loop
func frontpageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}
	p, err := loadPage("index")
	if err != nil {
		p = &Page{Title: "index"}
	}
	renderTemplate(w, "index", p)
}

// creation handler for new pages
// this can override an existing page
// /create/page1 will change title/body of existing page1.txt -- not wanted
func creationHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := pageCheck(title)
	if err != nil {
		p = &Page{Title: title}
		fmt.Println("creation handler worked")
		renderTemplate(w, "create", p)
		return
	}
	log.Println("creation handler failed, page exists already")
	pageRedirect(w, r)
	return
}

// subdirectory for viewing of pages
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
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
		log.Println("error saving page")
		return
	}
	// when a file is saved, this is where it goes.
	// writes it & redirects user to that new/updated page
	http.Redirect(w, r, "../data/"+title+".txt", http.StatusFound)
}

// calls the correct template based on URL
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view|index|create|test)/([a-zA-Z0-9]+)$")

// makes and runs the handler (view, edit, save, etc.), checks to see if the path is valid
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			fmt.Printf("%v", m)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/create/", makeHandler(creationHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
