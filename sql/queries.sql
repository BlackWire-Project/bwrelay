-- name: GetUser :one
SELECT * FROM Users
WHERE public_key = $1 LIMIT 1;
