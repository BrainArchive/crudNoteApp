package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/a-h/templ"
	"github.com/brainarchive/crudNoteApp/internal/database"
	"github.com/brainarchive/crudNoteApp/static/website"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	db *database.Queries
}

//go:embed static
var staticFiles embed.FS

func main() {
	godotenv.Load()
	err := mime.AddExtensionType(".js", "text/javascript")
	if err != nil {
		log.Fatal("can't add extension type to javascript", err)
	}
	dbUrl := os.Getenv("DB_URL")
	dbConnection, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("cannot connect to database:", err)
	}
	dbQueries := database.New(dbConnection)
	apiCfg := apiConfig{
		db: dbQueries,
	}

	index := website.Index()

	slog.Info("running server", "port", 3000)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", templ.Handler(index).ServeHTTP)
	r.Get("/notes", apiCfg.loadAllNotes)

	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
	}
	fileServer(r, "/static", http.FS(files))

	// file servers in golang
	//

	//	apiRouter := chi.NewRouter()
	//	apiRouter.Post("/v1/notes", apiCfg.createNoteHandler)
	//	apiRouter.Get("/v1/notes", apiCfg.getAllNoteHandler)
	//	apiRouter.Get("/v1/notes/{noteID}", apiCfg.getNoteHandler)
	//	apiRouter.Put("/v1/notes/{noteID}", apiCfg.updateNoteHandler)
	//	apiRouter.Delete("/v1/notes/{noteID}", apiCfg.deleteNoteHandler)
	//	r.Mount("/api/", apiRouter)
	http.ListenAndServe(":3000", r)
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
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
