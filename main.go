package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/brainarchive/crudNoteApp/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	db *database.Queries
}

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	dbConnection, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("cannot connect to database:", err)
	}
	dbQueries := database.New(dbConnection)
	apiCfg := apiConfig{
		db: dbQueries,
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", apiCfg.getHealth)
	r.Post("/api/v1/notes", apiCfg.createNoteHandler)
	r.Get("/api/v1/notes", apiCfg.getAllNoteHandler)
	r.Get("/api/v1/notes/{noteID}", apiCfg.getNoteHandler)
	r.Put("/api/v1/notes/{noteID}", apiCfg.updateNoteHandler)
	r.Delete("/api/v1/notes/{noteID}", apiCfg.deleteNoteHandler)
	http.ListenAndServe(":3000", r)
}

func (cfg *apiConfig) getHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}
