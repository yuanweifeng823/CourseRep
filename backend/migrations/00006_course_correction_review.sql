-- +goose Up
ALTER TABLE course_corrections
    ADD COLUMN reviewed_by BIGINT REFERENCES users(id),
    ADD COLUMN reviewed_at TIMESTAMPTZ,
    ADD COLUMN resolution TEXT NOT NULL DEFAULT '';
CREATE INDEX course_corrections_queue_idx ON course_corrections(created_at) WHERE status='open';

-- +goose Down
DROP INDEX course_corrections_queue_idx;
ALTER TABLE course_corrections DROP COLUMN resolution;
ALTER TABLE course_corrections DROP COLUMN reviewed_at;
ALTER TABLE course_corrections DROP COLUMN reviewed_by;
