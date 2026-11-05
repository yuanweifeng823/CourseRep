-- +goose Up
CREATE TABLE community_settings (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    content_review_sla_hours INTEGER CHECK (content_review_sla_hours IS NULL OR content_review_sla_hours BETWEEN 1 AND 720),
    community_guidelines TEXT NOT NULL DEFAULT '',
    updated_by BIGINT REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO community_settings(id, content_review_sla_hours, community_guidelines)
VALUES (1, NULL, '');

-- +goose Down
DROP TABLE community_settings;
