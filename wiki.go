package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
	"regexp"
)

var templates = template.Must(template.ParseFiles("templates/edit.html", "templates/view.html"))

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

type Page struct {
	Id    int64
	Title string
	Body  string
}

func (p *Page) save() error {
	db, err := sqlx.Connect("postgres", "postgres://postgres:Welcome1@192.168.29.21/wiki")
	checkErr(err, "Connection failed")
	defer db.Close()

	tx := db.MustBegin()

	tx.MustExec("insert into pages(title, body) values ($1, $2)", p.Title, p.Body)

	tx.Commit()
	return nil
}

func loadPage(title string) (*Page, error) {
	db, err := sqlx.Connect("postgres", "postgres://postgres:Welcome1@192.168.29.21/wiki")
	checkErr(err, "Connection failed")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	row := db.QueryRowx("select title, body from pages where title = $1", title)
	var ptitle string
	var pbody string
	err = row.Scan(&ptitle, &pbody)
	if err != nil {
		return nil, err
	}
	return &Page{Title: ptitle, Body: pbody}, nil

}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view.html", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit.html", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: body}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func frontPageRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func initializeDb() {
	db, err := sqlx.Connect("postgres", "postgres://postgres:Welcome1@192.168.29.21:5432/wiki")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schema := `
	CREATE TABLE IF NOT EXISTS pages (id serial primary key, title text, body text);
	`
	db.MustExec(schema)
}

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func main() {
	initializeDb()
	http.HandleFunc("/", frontPageRedirect)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.ListenAndServe(":8080", nil)
}
