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
SET title = COALESCE(sqlc.narg('title')::text, title), body = COALESCE(sqlc.narg('body')::text, body), updated_at = NOW()
WHERE id = @id 
RETURNING *;

-- name: DeleteNoteByID :exec
DELETE FROM notes
    WHERE id = $1;

-- name: DeleteAllNotes :exec
DELETE FROM notes;