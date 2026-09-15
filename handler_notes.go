package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/brainarchive/crudNoteApp/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/starfederation/datastar-go/datastar"
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
	noteUUID, err := uuid.Parse(noteIDParam)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid noteID", err)
		return
	}
	note, err := cfg.db.GetNoteByID(r.Context(), noteUUID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "note not found", err)
		return
	}
	var body *string
	if note.Body.Valid {
		body = &note.Body.String
	}

	respondWithJSON(w, http.StatusOK, Note{
		ID:        note.ID,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
		Title:     note.Title,
		Body:      body,
	})
}

func (cfg *apiConfig) loadAllNotes(w http.ResponseWriter, r *http.Request) {
	slog.Info("Loading notes", "endpoint", "/notes")
	notes, err := cfg.db.GetAllNotes(r.Context())
	sse := datastar.NewSSE(w, r)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get notes", err)
		return
	}
	var notesList []Note
	for _, notex := range notes {
		var body *string
		if notex.Body.Valid {
			body = &notex.Body.String
		}
		notesList = append(notesList, Note{
			ID:        notex.ID,
			CreatedAt: notex.CreatedAt,
			UpdatedAt: notex.UpdatedAt,
			Title:     notex.Title,
			Body:      body,
		})
	}
	var test string
	test = `<div id='notes'>`
	for _, note := range notesList {
		test += fmt.Sprintf(`<div>%s</div>`, note.Title)
		if *note.Body != "" {
			test += fmt.Sprintf(`<div>%s</div>`, *note.Body)
		} else {
			test += `<div>no body</div>`
		}
	}
	test += `</div>`

	sse.PatchElements(test)

}

func (cfg *apiConfig) getAllNoteHandler(w http.ResponseWriter, r *http.Request) {
	notes, err := cfg.db.GetAllNotes(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get notes", err)
		return
	}
	var notesList []Note
	for _, notex := range notes {
		var body *string
		if notex.Body.Valid {
			body = &notex.Body.String
		}
		notesList = append(notesList, Note{
			ID:        notex.ID,
			CreatedAt: notex.CreatedAt,
			UpdatedAt: notex.UpdatedAt,
			Title:     notex.Title,
			Body:      body,
		})
	}
	respondWithJSON(w, http.StatusOK, notesList)
}

func (cfg *apiConfig) updateNoteHandler(w http.ResponseWriter, r *http.Request) {
	noteIDParam := chi.URLParam(r, "noteID")
	noteUUID, err := uuid.Parse(noteIDParam)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid noteID", err)
		return
	}

	type parameters struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "required Title", err)
		return
	}
	var title sql.NullString
	if params.Title != nil {
		title = sql.NullString{String: *params.Title, Valid: true}
	}
	var textBody sql.NullString
	if params.Body != nil {
		textBody = sql.NullString{String: *params.Body, Valid: true}
	}

	newNote, err := cfg.db.UpdateNote(r.Context(), database.UpdateNoteParams{
		ID:    noteUUID,
		Title: title,
		Body:  textBody,
	})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "note not found", err)
		return
	}

	var body *string
	if newNote.Body.Valid {
		body = &newNote.Body.String
	}

	respondWithJSON(w, http.StatusOK, Note{
		ID:        newNote.ID,
		CreatedAt: newNote.CreatedAt,
		UpdatedAt: newNote.UpdatedAt,
		Title:     newNote.Title,
		Body:      body,
	})
}

func (cfg *apiConfig) deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	noteIDParam := chi.URLParam(r, "noteID")
	noteUUID, err := uuid.Parse(noteIDParam)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid noteID", err)
		return
	}

	err = cfg.db.DeleteNoteByID(r.Context(), noteUUID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "note not found", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, struct{}{})
}
