-- +goose Up
ALTER TABLE users ADD COLUMN session_version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE refresh_sessions ADD COLUMN session_version BIGINT NOT NULL DEFAULT 0;

CREATE TABLE password_resets (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code_hash BYTEA NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5)
);

-- +goose Down
DELETE FROM refresh_sessions USING users
WHERE refresh_sessions.user_id = users.id
  AND refresh_sessions.session_version <> users.session_version;
DROP TABLE password_resets;
ALTER TABLE refresh_sessions DROP COLUMN session_version;
ALTER TABLE users DROP COLUMN session_version;
