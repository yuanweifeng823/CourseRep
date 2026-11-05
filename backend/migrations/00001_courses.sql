-- +goose Up
CREATE TABLE courses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_code TEXT NOT NULL,
    course_name TEXT NOT NULL,
    professor TEXT NOT NULL DEFAULT '',
    semester TEXT NOT NULL,
    department TEXT NOT NULL,
    credits NUMERIC(4, 1) NOT NULL CHECK (credits >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT courses_course_code_semester_key UNIQUE (course_code, semester)
);

CREATE INDEX courses_course_code_idx ON courses (course_code);
CREATE INDEX courses_course_name_idx ON courses (course_name);
CREATE INDEX courses_professor_idx ON courses (professor);
CREATE INDEX courses_department_idx ON courses (department);

-- +goose Down
DROP TABLE courses;
