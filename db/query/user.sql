-- name: InsertUser :one
INSERT INTO "users" (
    "email_address",
    "prefix",
    "token",
    "generated_at",
    "expired_at"
  )
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
-- name: GetUser :one
SELECT *
FROM "users"
WHERE "prefix" = $1;