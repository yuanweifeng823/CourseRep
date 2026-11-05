-- +goose Up
CREATE TABLE course_view_events (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    viewed_on DATE NOT NULL DEFAULT CURRENT_DATE,
    PRIMARY KEY(user_id, course_id, viewed_on)
);
CREATE INDEX course_view_events_viewed_idx ON course_view_events(viewed_on);

-- +goose Down
DROP TABLE course_view_events;
