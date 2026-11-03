-- +goose Up
-- Columns filled by NOW() hold wall-clock time in the server's TimeZone, which
-- the default cast assumes. expires_at is written by the app in UTC, so it is
-- converted explicitly.
ALTER TABLE users
    ALTER COLUMN created_at TYPE TIMESTAMPTZ,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ,
    ALTER COLUMN hashed_password DROP DEFAULT;

ALTER TABLE chirps
    ALTER COLUMN created_at TYPE TIMESTAMPTZ,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE refresh_tokens
    ALTER COLUMN created_at TYPE TIMESTAMPTZ,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ,
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN revoked_at TYPE TIMESTAMPTZ;

CREATE INDEX chirps_user_id_created_at_id_idx ON chirps (user_id, created_at, id);
CREATE INDEX chirps_created_at_id_idx ON chirps (created_at, id);

-- +goose Down
DROP INDEX chirps_created_at_id_idx;
DROP INDEX chirps_user_id_created_at_id_idx;

ALTER TABLE refresh_tokens
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN expires_at TYPE TIMESTAMP USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN revoked_at TYPE TIMESTAMP;

ALTER TABLE chirps
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE users
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN hashed_password SET DEFAULT 'unset';
