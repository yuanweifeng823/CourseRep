-- +goose Up
CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nickname TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unverified' CHECK (status IN ('unverified','verified','alumni_readonly','disabled')),
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user','moderator','admin')),
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE email_verifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX email_verifications_email_created_idx ON email_verifications(email, created_at DESC);
CREATE TABLE refresh_sessions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE topic_tags (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE course_reviews (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    author_id BIGINT NOT NULL REFERENCES users(id),
    semester TEXT NOT NULL,
    grading INTEGER NOT NULL CHECK (grading BETWEEN 1 AND 5),
    workload INTEGER NOT NULL CHECK (workload BETWEEN 1 AND 5),
    assessment INTEGER NOT NULL CHECK (assessment BETWEEN 1 AND 5),
    usefulness INTEGER NOT NULL CHECK (usefulness BETWEEN 1 AND 5),
    tags TEXT[] NOT NULL DEFAULT '{}',
    body TEXT NOT NULL DEFAULT '' CHECK (char_length(body) <= 2000),
    status TEXT NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted','under_review','published','rejected','removed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX course_reviews_one_active_idx ON course_reviews(author_id, course_id, semester)
    WHERE status IN ('submitted','under_review','published');
CREATE INDEX course_reviews_public_idx ON course_reviews(course_id, semester) WHERE status='published';
CREATE TABLE course_posts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    author_id BIGINT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    image_urls TEXT[] NOT NULL DEFAULT '{}',
    topic_tags TEXT[] NOT NULL DEFAULT '{}',
    attendance_year INTEGER CHECK (attendance_year BETWEEN 1990 AND 2100),
    display_identity TEXT NOT NULL DEFAULT 'named' CHECK (display_identity IN ('named','anonymous')),
    status TEXT NOT NULL DEFAULT 'submitted' CHECK (status IN ('draft','submitted','under_review','published','rejected','removed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    CHECK (cardinality(image_urls) <= 9),
    CHECK (attendance_year IS NULL OR attendance_year <= EXTRACT(YEAR FROM CURRENT_DATE)::INTEGER)
);
CREATE INDEX course_posts_public_idx ON course_posts(course_id, created_at DESC) WHERE status='published';
CREATE INDEX course_posts_year_idx ON course_posts(course_id, attendance_year) WHERE status='published';
CREATE TABLE post_comments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES course_posts(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id),
    body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE post_likes (
    post_id BIGINT NOT NULL REFERENCES course_posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(post_id,user_id)
);
CREATE TABLE post_bookmarks (
    post_id BIGINT NOT NULL REFERENCES course_posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(post_id,user_id)
);
CREATE TABLE reports (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reporter_id BIGINT NOT NULL REFERENCES users(id),
    review_id BIGINT REFERENCES course_reviews(id),
    post_id BIGINT REFERENCES course_posts(id),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 1000),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','reviewing','resolved','dismissed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((review_id IS NOT NULL)::int + (post_id IS NOT NULL)::int = 1)
);
CREATE TABLE moderation_actions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    moderator_id BIGINT NOT NULL REFERENCES users(id),
    report_id BIGINT REFERENCES reports(id),
    review_id BIGINT REFERENCES course_reviews(id),
    post_id BIGINT REFERENCES course_posts(id),
    action TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE notifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_created_idx ON notifications(user_id, created_at DESC);
CREATE TABLE course_corrections (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    author_id BIGINT NOT NULL REFERENCES users(id),
    proposed_fields JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','accepted','rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE course_corrections;
DROP TABLE notifications;
DROP TABLE moderation_actions;
DROP TABLE reports;
DROP TABLE post_bookmarks;
DROP TABLE post_likes;
DROP TABLE post_comments;
DROP TABLE course_posts;
DROP TABLE course_reviews;
DROP TABLE topic_tags;
DROP TABLE refresh_sessions;
DROP TABLE email_verifications;
DROP TABLE users;
