package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/starfederation/datastar-go/datastar"
)

func (cfg *apiConfig) editNoteStarHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("editNoteStarHandler Called")
	var id uuid.UUID
	var err error
	noteID := chi.URLParam(r, "noteID")
	id, err = uuid.Parse(noteID)
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}
	//TODO: figure out a way to get the title and the body from the original note
	// then pass them into here as data
	// ideally server-side note should be saved somewhere where I can get it.
	// and only get the note it's not in that last, append to the list and flush out oldest
	// should be a read-only cache. when write, update cache.
	// maybe? There might be a simpler way to do this with datastar
	newNote, err := cfg.db.GetNoteByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "note not found", err)
		return
	}

	sse := datastar.NewSSE(w, r)
	var buf bytes.Buffer
	data := struct {
		ID    uuid.UUID
		Title string
		Body  string
	}{
		ID:    id,
		Title: newNote.Title,
		Body:  newNote.Body.String,
	}
	cfg.pages["home.html"].ExecuteTemplate(&buf, "note-edit", data)

	res := buf.String()
	err = sse.PatchElements(res)

	if err != nil {
		slog.Error("error with PatchElements", "error", err)
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) noteDisplayStarHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("noteDisplayStarHandler called")
	var id uuid.UUID
	var err error
	noteID := chi.URLParam(r, "noteID")
	id, err = uuid.Parse(noteID)
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)
	newNote, err := cfg.db.GetNoteByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "note not found", err)
		return
	}
	var buf bytes.Buffer
	data := struct {
		ID    uuid.UUID
		Title string
		Body  string
	}{
		ID:    id,
		Title: newNote.Title,
		Body:  newNote.Body.String,
	}

	cfg.pages["home.html"].ExecuteTemplate(&buf, "note-display", data)
	res := buf.String()
	err = sse.PatchElements(res)

	if err != nil {
		slog.Error("error with PatchElements", "error", err)
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) deleteNoteStarHandler(w http.ResponseWriter, r *http.Request) {
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

	sse := datastar.NewSSE(w, r)
	s := fmt.Sprintf("note-%s", noteUUID)
	print(s)
	err = sse.RemoveElement(s)
	if err != nil {
		slog.Error("error with RemoveElement", "error", err)
		respondWithError(w, http.StatusInternalServerError, "failed to delete from frontend: %s", err)
		return
	}
}

//func (cfg *apiConfig) editNoteFormHandler(w http.ResponseWriter, r *http.Request) {
//	id, err := uuid.Parse(chi.URLParam(r, "noteID"))
//	if err != nil {
//		http.Error(w, "invalid note id", http.StatusBadRequest)
//		return
//	}
//
//	note, err := cfg.db.GetNoteByID(r.Context(), id)
//	if err != nil {
//		if errors.Is(err, sql.ErrNoRows) {
//			http.NotFound(w, r)
//			return
//		}
//		http.Error(w, "could not load note", http.StatusInternalServerError)
//		return
//	}
//	data := struct {
//		ID    uuid.UUID
//		Title string
//		Body  string
//	}{
//		ID:    note.ID,
//		Title: note.Title,
//		Body:  note.Body.String, // sql.NullString's zero value is "" when not Valid — safe as-is
//	}
//	if err := cfg.pages["edit.html"].ExecuteTemplate(w, "edit", data); err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//	}
//}
