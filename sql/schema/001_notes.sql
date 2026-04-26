-- +goose Up
CREATE TABLE notes (
    id uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    title TEXT NOT NULL,
    body TEXT
);

-- +goose Down 
DROP TABLE notes;