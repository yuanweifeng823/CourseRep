-- +goose Up
ALTER TABLE email_verifications
    ADD COLUMN password_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN nickname TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE email_verifications DROP COLUMN nickname;
ALTER TABLE email_verifications DROP COLUMN password_hash;
