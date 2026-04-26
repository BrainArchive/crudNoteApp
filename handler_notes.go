package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/brainarchive/crudNoteApp/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Title     string    `json:"title"`
	Body      *string   `json:"body"`
}

func (cfg *apiConfig) createNoteHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "required Title", err)
		return
	}

	if len(params.Title) == 0 {
		respondWithError(w, http.StatusBadRequest, "required Title", nil)
		return
	}

	newNote, err := cfg.db.CreateNote(r.Context(), database.CreateNoteParams{
		Title: params.Title,
		Body: sql.NullString{
			String: params.Body,
			Valid:  params.Body != "",
		},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create note", err)
		return
	}
	var body *string
	if newNote.Body.Valid {
		body = &newNote.Body.String
	}

	respondWithJSON(w, http.StatusCreated, Note{
		ID:        newNote.ID,
		CreatedAt: newNote.CreatedAt,
		UpdatedAt: newNote.UpdatedAt,
		Title:     newNote.Title,
		Body:      body,
	})
}

func (cfg *apiConfig) getNoteHandler(w http.ResponseWriter, r *http.Request) {
	noteIDParam := chi.URLParam(r, "noteID")
	return
}
