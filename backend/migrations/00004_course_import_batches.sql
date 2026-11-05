-- +goose Up
CREATE TABLE course_import_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id BIGINT NOT NULL REFERENCES users(id),
    rows JSONB NOT NULL,
    errors JSONB NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','discarded')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT now() + interval '24 hours'
);

-- +goose Down
DROP TABLE course_import_batches;
