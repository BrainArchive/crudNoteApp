package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

func (cfg *apiConfig) insertNote(ctx context.Context, title, body string) (database.Note, error) {

	return cfg.db.CreateNote(ctx, database.CreateNoteParams{
		Title: title,
		Body:  sql.NullString{String: body, Valid: body != ""},
	})
}

func (cfg *apiConfig) createNoteFormHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form data", http.StatusBadRequest)
		return
	}
	cfg.insertNote(r.Context(), r.FormValue("title"), r.FormValue("body"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
		respondWithError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if len(params.Title) == 0 {
		respondWithError(w, http.StatusBadRequest, "required Title", nil)
		return
	}

	newNote, err := cfg.insertNote(r.Context(), params.Title, params.Body)
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

func (cfg *apiConfig) getAllNotes(ctx context.Context) ([]Note, error) {
	notes, err := cfg.db.GetAllNotes(ctx)
	if err != nil {
		return nil, err
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
	return notesList, nil
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
	sse := datastar.NewSSE(w, r)
	notesList, err := cfg.getAllNotes(r.Context())
	if err != nil {
		sse.PatchElements(`<div id='notes'></div>`)
		respondWithError(w, http.StatusInternalServerError, "could not get notes", err)
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
	notesList, err := cfg.getAllNotes(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get notes", err)
		return
	}
	respondWithJSON(w, http.StatusOK, notesList)
}

func (cfg *apiConfig) updateNote(ctx context.Context, uuid uuid.UUID, title, body *string) (Note, error) {

	var noteTitle sql.NullString
	var noteBody sql.NullString
	if title != nil {
		noteTitle = sql.NullString{String: *title, Valid: true}
	}
	if body != nil {
		noteBody = sql.NullString{String: *body, Valid: true}
	}
	resNote, err := cfg.db.UpdateNote(ctx, database.UpdateNoteParams{
		ID:    uuid,
		Title: noteTitle,
		Body:  noteBody,
	})
	if err != nil {
		return Note{}, err
	}
	var newBody *string
	if resNote.Body.Valid {
		newBody = &resNote.Body.String
	}

	newNote := Note{
		ID:        resNote.ID,
		CreatedAt: resNote.CreatedAt,
		UpdatedAt: resNote.UpdatedAt,
		Title:     resNote.Title,
		Body:      newBody,
	}
	return newNote, nil
}

//The form path can never actually trigger a partial update.
//r.FormValue("title") returns "" when a field is genuinely missing —
//it never returns nil — so &noteTitle and &noteBody are always non-nil pointers, no matter what the user submitted. Your updateNote function was built to support "skip this field" via nil, but the form handler structurally can't produce that.

//For an edit form that always shows both fields and resubmits both, this is actually fine — you want a full overwrite there.
//Just be clear with yourself that it's fine by circumstance, not because the code is doing anything clever; the partial-update path only means something for your JSON API callers.

func (cfg *apiConfig) editNoteFormHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	note, err := cfg.db.GetNoteByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "could not load note", http.StatusInternalServerError)
		return
	}
	data := struct {
		ID    uuid.UUID
		Title string
		Body  string
	}{
		ID:    note.ID,
		Title: note.Title,
		Body:  note.Body.String, // sql.NullString's zero value is "" when not Valid — safe as-is
	}
	if err := cfg.pages["edit.html"].ExecuteTemplate(w, "edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) updateNoteFormHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var noteTitle string
	var noteBody string
	noteID := chi.URLParam(r, "noteID")
	noteUUID, err := uuid.Parse(noteID)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid noteID", err)
		return
	}
	if err = r.ParseForm(); err != nil {
		http.Error(w, "bad form data", http.StatusBadRequest)
		return
	}
	noteTitle = r.FormValue("title")
	noteBody = r.FormValue("body")

	if _, err := cfg.updateNote(r.Context(), noteUUID, &noteTitle, &noteBody); err != nil {
		http.Error(w, "could not update note", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	newNote, err := cfg.updateNote(r.Context(), noteUUID, params.Title, params.Body)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "note not found", err)
		return
	}
	respondWithJSON(w, http.StatusOK, newNote)
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
