-- name: CreateNote :one
INSERT INTO notes (
    title, body 
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetNoteByID :one
SELECT * FROM notes WHERE id = $1;

-- name: GetAllNotes :many
SELECT * FROM notes;

-- name: UpdateNote :one
UPDATE notes
SET title = COALESCE($2, title), body = COALESCE($3, body), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteNoteByID :exec
DELETE FROM notes
    WHERE id = $1;

-- name: DeleteAllNotes :exec
DELETE FROM notes;