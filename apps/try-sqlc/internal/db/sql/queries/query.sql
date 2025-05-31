-- name: GetAuthorByAge :one
SELECT * FROM authors
WHERE age = ? LIMIT 1;