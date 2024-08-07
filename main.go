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

// save function
func (p *Page) save() error {
	// save page title into dataDir
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

// redirects to front page if user tries to view nonexistent page
// doesn't actually redirect right now because of infinite redirect loop
func frontpageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view.html", http.StatusFound)
}

// 	// search for all pages available
// 	files, err := os.ReadDir("/Users/saxon/vscode/web_app_personal/data/")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	for _, file := range files {
// 		fmt.Println(file.Name())
// 	}

// 	// search for index template and redirect to that page
// 	tmpl, err := os.ReadDir("/Users/saxon/vscode/web_app_personal/")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	for _, file := range tmpl {
// 		fmt.Println(file.Name())
// 		if file.Name() == "index.html" {
// 			//endless loop occuring here
// 			http.Redirect(w, r, "index.html", http.StatusFound)
// 			return
// 		}
// 	}
// }

// creation handler for new pages
func creationHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	log.Println(w, r, title, "creation handler")
	log.Println(p, "creation handler")
	renderTemplate(w, "create", p)
}

// subdirectory for viewing of pages
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	// p, err := loadIndex(title)
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
	http.Redirect(w, r, "/data/"+title, http.StatusFound)
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

		log.Println(m[:])
	}
}

func main() {

	http.HandleFunc("/", frontpageHandler)
	// http.HandleFunc("/index/", makeHandler(index))
	// http.HandleFunc("/", makeHandler(indexHandler))
	http.HandleFunc("/create/", makeHandler(creationHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	// starts path in web_app folder
	// gets redirected to index.html
	log.Fatal(http.ListenAndServe(":8080", http.FileServer(http.Dir("/Users/saxon/vscode/web_app_personal"))))
}
