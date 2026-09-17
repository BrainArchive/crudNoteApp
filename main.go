package main

import (
	"database/sql"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/brainarchive/crudNoteApp/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// "github.com/a-h/templ"
// "github.com/brainarchive/crudNoteApp/templates"

type apiConfig struct {
	db    *database.Queries
	pages map[string]*template.Template
}

//go:embed static
var staticFiles embed.FS

func main() {
	godotenv.Load()
	var err error
	err = mime.AddExtensionType(".js", "text/javascript")
	err = mime.AddExtensionType(".css", "text/css")
	if err != nil {
		log.Fatal("can't add extension type to javascript", err)
	}
	// the filesystem starts as static/.... because of how go:embed works.
	// have to strip static so the new root is at ./...
	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal("substituting staticFiles failed", err)
	}

	dbUrl := os.Getenv("DB_URL")
	dbConnection, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("cannot connect to database:", err)
	}
	dbQueries := database.New(dbConnection)
	pages, err := loadTemplates()
	apiCfg := apiConfig{
		db:    dbQueries,
		pages: pages,
	}
	if err != nil {
		log.Fatal("error loading template:", err)
	}

	slog.Info("running server", "port", 3000)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", apiCfg.homeHandler)
	r.Get("/notes", apiCfg.loadAllNotes)
	r.Post("/notes", apiCfg.createNoteFormHandler)
	r.Get("/notes/{noteID}/edit", apiCfg.editNoteFormHandler)
	r.Get("/notes/{noteID}/edit_note", apiCfg.editNoteFormHandler)
	r.Get("/notes/{noteID}/edit_note_star", apiCfg.editNoteStarHandler)
	r.Get("/notes/{noteID}/display_note_star", apiCfg.noteDisplayStarHandler)
	r.Post("/notes/{noteID}", apiCfg.updateNoteFormHandler)
	r.Delete("/notes/{noteID}", apiCfg.deleteNoteStarHandler)

	fileServer(r, "/static", http.FS(files))

	apiRouter := chi.NewRouter()
	apiRouter.Post("/v1/notes", apiCfg.createNoteHandler)
	apiRouter.Get("/v1/notes", apiCfg.getAllNoteHandler)
	apiRouter.Get("/v1/notes/{noteID}", apiCfg.getNoteHandler)
	apiRouter.Put("/v1/notes/{noteID}", apiCfg.updateNoteHandler)
	apiRouter.Delete("/v1/notes/{noteID}", apiCfg.deleteNoteHandler)
	r.Mount("/api/", apiRouter)
	log.Fatal(http.ListenAndServe(":3000", r))
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	// FileServer adds to the chi.Router a get path
	// it'll strip the path string e.g if may path is /static
	// and I call /static/files
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func loadTemplates() (map[string]*template.Template, error) {
	var pages = map[string]*template.Template{}
	// layouts, err := filepath.Glob("static/website/*.html")
	partials, err := filepath.Glob("static/partials/*.html")
	if err != nil {
		return nil, err
	}
	pageFiles, err := filepath.Glob("static/website/*.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pageFiles {
		var files []string
		name := filepath.Base(page) // "home.html"
		files = append(files, partials...)
		files = append(files, page)
		pages[name] = template.Must(template.ParseFiles(files...))
	}
	return pages, nil
}

type PageNotes struct {
	Title string
	Body  string
	ID    uuid.UUID
}

type PageData struct {
	Notes []PageNotes
}

func (cfg apiConfig) homeHandler(w http.ResponseWriter, r *http.Request) {
	var NotesList []Note
	var err error
	NotesList, err = cfg.getAllNotes(r.Context())
	if err != nil {
		slog.Error("getting all notes error:", "error", err)
		NotesList = []Note{}
	}
	var dbNotes []PageNotes
	for _, note := range NotesList {
		dbNotes = append(dbNotes, PageNotes{
			Title: note.Title,
			Body:  *note.Body,
			ID:    note.ID,
		})
	}

	data := PageData{
		Notes: dbNotes,
	}
	if err := cfg.pages["home.html"].ExecuteTemplate(w, "home", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}
