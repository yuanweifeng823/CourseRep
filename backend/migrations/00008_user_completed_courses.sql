-- +goose Up
CREATE TABLE user_completed_courses (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(user_id, course_id)
);
CREATE INDEX user_completed_courses_course_idx ON user_completed_courses(course_id, user_id);

-- +goose Down
DROP TABLE user_completed_courses;
