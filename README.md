# Notes API

A simple REST API for managing notes, built with Go and PostgreSQL.

## Tech Stack

- Go 1.26
- PostgreSQL 16
- Chi (HTTP router)
- SQLC (type-safe SQL queries)
- Goose (database migrations)

## Setup

### Prerequisites
- Go 1.26
- PostgreSQL 16

### Steps

1. Clone the repository
2. Set your database URL as an environment variable:
   DB_URL=postgres://username:password@localhost:5432/notesapp?sslmode=disable
3. Run migrations:
   goose postgres $DB_URL up
4. Run the server:
   go run main.go

## Endpoints

Base URL: /api/v1

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /notes | Create a note |
| GET | /notes | List all notes |
| GET | /notes/{id} | Get a note by ID |
| PUT | /notes/{id} | Update a note |
| DELETE | /notes/{id} | Delete a note |

## Notes

- Title is required, body is optional
- Partial updates supported on PUT — only send fields you want to change