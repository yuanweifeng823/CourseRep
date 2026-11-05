-- +goose Up
-- 学生专业：首页按专业推送课程与经验帖。空字符串表示尚未选择。
ALTER TABLE users ADD COLUMN department TEXT NOT NULL DEFAULT '';

-- 经验帖浏览去重：同一用户同一天对同一帖只记一次，用于首页「近期热门分享」。
CREATE TABLE post_view_events (
    post_id BIGINT NOT NULL REFERENCES course_posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    viewed_on DATE NOT NULL DEFAULT CURRENT_DATE,
    PRIMARY KEY (post_id, user_id, viewed_on)
);

CREATE INDEX post_view_events_viewed_idx ON post_view_events (viewed_on);

-- +goose Down
DROP TABLE post_view_events;
ALTER TABLE users DROP COLUMN department;
