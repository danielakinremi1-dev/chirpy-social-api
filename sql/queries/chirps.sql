-- name: CreateChirp :one
INSERT INTO
    chirps (id, created_at, updated_at, body, user_id)
VALUES
    (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING
    *;

-- name: GetChirp :one
SELECT
    *
FROM
    chirps
WHERE
    id = $1;

-- name: DeleteChirp :exec
DELETE FROM chirps
WHERE
    id = $1;

-- name: ListChirpsAsc :many
SELECT
    *
FROM
    chirps
WHERE
    (
        sqlc.narg('author_id')::uuid IS NULL
        OR user_id = sqlc.narg('author_id')::uuid
    )
    AND (
        sqlc.narg('cursor_created_at')::timestamptz IS NULL
        OR (created_at, id) > (
            sqlc.narg('cursor_created_at')::timestamptz,
            sqlc.narg('cursor_id')::uuid
        )
    )
ORDER BY
    created_at ASC,
    id ASC
LIMIT
    sqlc.arg('row_limit');

-- name: ListChirpsDesc :many
SELECT
    *
FROM
    chirps
WHERE
    (
        sqlc.narg('author_id')::uuid IS NULL
        OR user_id = sqlc.narg('author_id')::uuid
    )
    AND (
        sqlc.narg('cursor_created_at')::timestamptz IS NULL
        OR (created_at, id) < (
            sqlc.narg('cursor_created_at')::timestamptz,
            sqlc.narg('cursor_id')::uuid
        )
    )
ORDER BY
    created_at DESC,
    id DESC
LIMIT
    sqlc.arg('row_limit');