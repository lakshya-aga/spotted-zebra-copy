-- name: GetParam :many
SELECT *
FROM "modelparameters"
WHERE "date" IN ($1);
-- name: GetLatestParamDate :one
SELECT DISTINCT "date"
FROM "modelparameters"
ORDER BY "date" DESC
LIMIT 1;
-- name: InsertParam :one
INSERT INTO "modelparameters" (
    "date",
    "ticker",
    "sigma",
    "alpha",
    "beta",
    "kappa",
    "rho"
  )
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;
-- name: GetCorr :many
SELECT *
FROM "corrpairs"
WHERE "date" in ($1)
ORDER BY "x0",
  "x1";
-- name: GetLatestCorrDate :one
SELECT DISTINCT "date"
FROM "corrpairs"
ORDER BY "date" DESC
LIMIT 1;
-- name: InsertCorr :one
INSERT INTO "corrpairs" ("date", "x0", "x1", "corr")
VALUES ($1, $2, $3, $4)
RETURNING *;
-- name: GetLatestPrice :many
SELECT "ticker",
  "fixing"
FROM "statistics"
WHERE "date" = (
    SELECT DISTINCT "date"
    FROM "statistics"
    ORDER BY "date" DESC
    LIMIT 1
  )
ORDER BY "ticker";
-- name: GetStats :many
SELECT *
FROM "statistics"
WHERE "date" IN ($1);
-- name: GetLatestStatsDate :one
SELECT DISTINCT "date"
FROM "statistics"
ORDER BY "date" DESC
LIMIT 1;
-- name: InsertStat :one
INSERT INTO "statistics" ("date", "ticker", "index", "mean", "fixing")
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
-- name: GetAllParam :many
SELECT *
FROM "modelparameters"
ORDER BY "date",
  "ticker";
-- name: GetAllStats :many
SELECT *
FROM "statistics"
ORDER BY "date",
  "ticker";
-- name: GetAllCorr :many
SELECT *
FROM "corrpairs"
ORDER BY "date",
  "x0",
  "x1";
-- name: GetAllDate :many
SELECT DISTINCT "date"
FROM "modelparameters"
ORDER BY "date";