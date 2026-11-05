-- +goose Up
-- 初始话题词典：平台维护，运营可在 /api/v1/admin/topic-tags 新增、改名或停用。
-- 词条仅为可用性兜底，具体清单待产品确认。
INSERT INTO topic_tags(name) VALUES
    ('选课经验'),
    ('避雷提醒'),
    ('学习资源'),
    ('给分友好'),
    ('工作量大'),
    ('小组作业'),
    ('期末考试'),
    ('论文项目')
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM topic_tags WHERE name IN (
    '选课经验',
    '避雷提醒',
    '学习资源',
    '给分友好',
    '工作量大',
    '小组作业',
    '期末考试',
    '论文项目'
);
