-- name: CreateNote :one
INSERT INTO notes (
    title, body 
) VALUES (
    $1, $2
) RETURNING *;