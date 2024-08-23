package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const dataDir string = "data"
const tmplDir string = "tmpl"

var validPath = regexp.MustCompile("^/(edit|save|view|create|login|signup|acctcreate)/([a-zA-Z0-9]+)$")

// list of templates
var templates = template.Must(template.ParseFiles(
	filepath.Join(tmplDir, "edit.html"),
	filepath.Join(tmplDir, "view.html"),
	filepath.Join(tmplDir, "create.html"),
	filepath.Join(tmplDir, "template.html"),
	filepath.Join(tmplDir, "login.html"),
	filepath.Join(tmplDir, "signup.html"),
))

type Page struct {
	Title string
	Body  []byte
}

type Credentials struct {
	Username string
	Password string
}

// save function.
// grabs filename and data from Page struct, places it into /data/ and adds .txt
func (p *Page) save() error {
	// filename := filepath.Join(dataDir, p.Title+".html")
	// if dataDir DNE then create w/ 0600 permissions
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.Mkdir(dataDir, 0600)
	}
	// return os.WriteFile(filename, p.Body, 0600)
	return createPage(p.Title, string(p.Body))
}

func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "This is the admin dashboard")
}

// authenticates the admin login request.
// takes parameter of "next http.handler" -- "next" is placeholder
func AdminAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// middleware -- checks for cookie "admin_session" for authentication
		if _, err := r.Cookie("admin_session"); err != nil {
			http.Redirect(w, r, "/login/", http.StatusSeeOther)
			return
		}
		// if authentication is successful, it calls the next handler
		next.ServeHTTP(w, r)
	})
}

// checks to see if a file exists or not before creating or editing
func doesExist(pagename string) bool {
	pagename = filepath.Join(dataDir, pagename+".html")
	if _, err := os.Stat(pagename); err == nil {
		// log.Println("page already exists, returning true")
		return true
	}
	// log.Println("page does not exist, continuing creation")
	return false
}

func IsEmpty(data string) bool {
	if len(data) == 0 {
		return true
	} else {
		return false
	}
}

func pwHash(password string) (hash string) {
	// creates new hash object
	hasher := sha256.New()
	// converts the password data into a byte slice, writes it to the hasher object
	hasher.Write([]byte(password))

	// computes the hash, returning it as a byte slice into hashBytes
	hashBytes := hasher.Sum(nil)
	// converts the bytes into a hexidecimal string
	hash = hex.EncodeToString(hashBytes)
	return hash

}

func scanDirectory(path string) ([]string, error) {
	var files []string

	// "walks" through the directories searching for files
	err := filepath.WalkDir(path, func(fp string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// checks if a directory & file of type .html does not exist, and disregards it
		if info.IsDir() || filepath.Ext(info.Name()) != ".html" {
			return nil
		}
		relativePath, _ := filepath.Rel(path, fp)
		files = append(files, relativePath)
		return nil
	})

	return files, err
}

// connect to database
func dbConnect() {
	// data source name (dsn). connection info for db. name:password@protocol(ip/port)/dbname
	dsn := "root:root@tcp(127.0.0.1:3306)/creds"

	// opens "creds" db with the dsn credentials
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// log.Println("didn't work")
		log.Fatal(err)
	}
	// pings the database to make sure it's online, if not, connects again
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func createPage(title, body string) error {
	// reads tmpl file
	template, err := os.ReadFile("./tmpl/template.html")
	if err != nil {
		return fmt.Errorf("could not read template file: %v", err)
	}

	templateStr := string(template)

	content := strings.ReplaceAll(templateStr, "{{.Title}}", title)
	content = strings.ReplaceAll(content, "{{.Body}}", body)

	err = os.WriteFile("./data/"+title+".html", []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("could not write HTML file: %v", err)
	}

	return nil
}

// checks login info against the database info
func credentialsCheck(db *sql.DB, username, pwHash string) bool {
	var dbHash string

	// sql query. selects the password column from users table where the name == username
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&dbHash)
	// if err successfully queries
	if err != nil {
		// username not found within rows
		if err == sql.ErrNoRows {
			// Username not found
			return false
		}
	}
	return dbHash == pwHash

}

func login(w http.ResponseWriter, r *http.Request) {
	if !loginFormCheck(w, r) {
		// log.Println("form check FAILED in login")
		return
	}

}

// checks the data submitted into the HTML login forum
func loginFormCheck(w http.ResponseWriter, r *http.Request) bool {
	// log.Println("form check started")
	uName, pw := "", ""
	r.ParseForm()

	uName = r.FormValue("username")
	pw = r.FormValue("password")

	// empty checking, return bool
	uNameCheck := IsEmpty(uName)
	pwCheck := IsEmpty(pw)

	// checks to see if any bool check is true (empty)
	if uNameCheck || pwCheck {
		log.Println(w, "Empty data in an input")
		return false
	}

	pwHash := pwHash(pw)
	dsn := "root:root@tcp(127.0.0.1:3306)/creds"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// log.Println("failed to start db server")
		http.Error(w, "error opening db in loginformcheck", http.StatusInternalServerError)
	}
	log.Println("started db server")
	// closes db once singup is complete
	defer db.Close()

	if !credentialsCheck(db, uName, pwHash) {
		log.Println("login form, credentialcheck FAILED")
		return false
	}
	// log.Println("form check ran successfully")
	return true
}

// checks the data submitted into the HTML login forum
func signup(w http.ResponseWriter, r *http.Request) {
	// checks if credentials in forum are acceptable
	if !signupFormCheck(w, r) {
		// log.Println("form check FAILED in signup")
		return
	}
	// starts up the db
	dsn := "root:root@tcp(127.0.0.1:3306)/creds"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// log.Println("failed to start db server")
		return
	}
	log.Println("started db server")
	// closes db once singup is complete
	defer db.Close()

	// takes acceptable values from forum and inserts them into the db
	username := r.FormValue("username")
	password := r.FormValue("password")
	pwHash := pwHash(password)

	// userCheck := db.QueryRow("SELECT username FROM users WHERE username = ?", username)
	// if userCheck != nil {
	// 	// log.Println("username already exists")
	// 	http.Error(w, "Username already taken.", http.StatusInternalServerError)
	// }

	_, err = db.Exec("INSERT INTO users (username, password_hash) VALUE (?, ?)", username, pwHash)
	if err != nil {
		// log.Println("failed to insert data")
		http.Error(w, "Failed to insert data", http.StatusInternalServerError)
	}

}

func signupFormCheck(w http.ResponseWriter, r *http.Request) bool {
	// log.Println("form check started")
	uName, pw, pwConfirm := "", "", ""
	r.ParseForm()
	// username from the form
	uName = r.FormValue("username")
	// password from the form
	pw = r.FormValue("password")
	// confirm password, must be same as first password
	pwConfirm = r.FormValue("passwordConfirm")

	// empty checking, return bool
	uNameCheck := IsEmpty(uName)
	pwCheck := IsEmpty(pw)
	pwConfirmCheck := IsEmpty(pwConfirm)

	// checks to see if any bool check is true (empty)
	if uNameCheck || pwCheck || pwConfirmCheck {
		log.Println(w, "Empty data in an input")
		return false
	}
	// checks if pw is the same as pwConfirm or not.
	if pw == pwConfirm {
		// log.Println(w, "Passwords are the same.")
	} else {
		// log.Println(w, "Passwords must be the same")
		return false
	}
	// log.Println("form check ran successfully")
	return true
}

func loginFormHandler(w http.ResponseWriter, r *http.Request) {
	login(w, r)
	// log.Println("finished loginformhandler")
	// when there is place to go after logging in, replace /login/
	http.Redirect(w, r, "/login/", http.StatusFound)

}

func acctCreateHandler(w http.ResponseWriter, r *http.Request) {
	signup(w, r)
	// log.Println("finished acct handler")
	http.Redirect(w, r, "/login/", http.StatusFound)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "signup"}
	// fmt.Println("signup handler worked")
	renderTemplate(w, "signup", p)
}

// handle the user login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "login"}
	// fmt.Println("login handler worked")
	renderTemplate(w, "login", p)
}

// attempt at making indexHandler() work within the html file rather than here
func fileListHandler(w http.ResponseWriter, r *http.Request) {
	files, err := scanDirectory("./data")
	if err != nil {
		http.Error(w, "Unable to list files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Files []string `json:"files"`
	}{
		Files: files,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Unable to encode JSON", http.StatusInternalServerError)
	}

}

// creates new pages.
func creationHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "create"}
	// fmt.Println("creation handler worked")
	renderTemplate(w, "create", p)
}

// subdirectory for viewing of pages
func viewHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "view"}
	fmt.Println("view handler worked")
	renderTemplate(w, "view", p)
}

// allows for editing of a page, not creation
func editHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "edit"}
	fmt.Println("edit handler worked")
	renderTemplate(w, "edit", p)
}

// allows for saving input of a page when editing
func saveHandler(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.FormValue("newpage_name"))
	body := r.FormValue("newpage_body")
	// input is saved with create or edit
	// if create: checks if the page exists. if exists and tmpl == create, don't create. if not, create.
	if doesExist(title) {
		if checkTemplate(r) == "create" {
			// log.Println("page creation failed. page exists already.")
			http.Redirect(w, r, "/error/create_error.html", http.StatusFound)
			return
		}
	}
	// if edit: checks if page exists. if does not exist and tmpl == edit, don't edit.
	// if page doesn't exist, don't create
	// if page exists, and tmpl == edit, then edit.
	if !doesExist(title) {
		if checkTemplate(r) == "edit" {
			// log.Println("page edit failed. cannot edit a page that doesn't exist.")
			http.Redirect(w, r, "/error/edit_error.html", http.StatusFound)
			return
		}
	}
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// log.Println("error saving page")
		return
	}
	// when a file is saved, this is where it goes.
	// writes it & redirects user to that new/updated page
	http.Redirect(w, r, "../data/"+title+".html", http.StatusFound)
}

// calls the correct template based on URL
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func checkTemplate(r *http.Request) string {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m[2] == "create" {
		// log.Println(m[:], "create line")
		return "create"
	}
	if m[2] == "edit" {
		// log.Println(m[:], "edit line")
		return "edit"
	}
	return "view"
}

// makes and runs the handler (view, edit, save, etc.), checks to see if the path is valid
// func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		m := validPath.FindStringSubmatch(r.URL.Path)
// 		if m == nil {
// 			http.NotFound(w, r)
// 			fmt.Printf("%v", m)
// 			return
// 		}
// 		fn(w, r, m[2])
// 	}
// }

func main() {
	dbConnect()
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/files", fileListHandler)
	http.HandleFunc("/create/", (creationHandler))
	http.HandleFunc("/view/", (viewHandler))
	http.HandleFunc("/edit/", (editHandler))
	http.HandleFunc("/login/", (loginHandler))
	http.HandleFunc("/loginredirect/", (loginFormHandler))
	http.HandleFunc("/signup/", (signupHandler))
	http.HandleFunc("/acctcreate/", (acctCreateHandler))
	http.HandleFunc("/save/", (saveHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
