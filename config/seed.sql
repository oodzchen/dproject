-- 创建超级管理员
CREATE ROLE admin WITH SUPERUSER CREATEDB CREATEROLE LOGIN PASSWORD '88886666';

-- 删除旧的数据库
DROP DATABASE discuss;

-- 创建数据库
CREATE DATABASE discuss OWNER admin ENCODING 'UTF-8';

-- 连接到 discuss 数据库
\c discuss admin

-- 用户
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 创建文章数据表格
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255),
    author_id INTEGER REFERENCES users(id),
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    reply_to INTEGER,
    deleted BOOLEAN NOT NULL DEFAULT false
);

-- reply_to 为空时候标题不能为空，replay_to 不为空时标题可以为空
ALTER TABLE posts ADD CONSTRAINT posts_reply_to_title_check CHECK (
    (reply_to IS NULL AND title IS NOT NULL) OR
    (reply_to IS NOT NULL)
);

-- 插入样例数据
-- 用户样例数据
INSERT INTO users (email, password, name)
VALUES
    ('zhangsan@example.com', '123456', '张三'),
    ('lisi@example.com', '123456', '李四'),
    ('wangwu@example.com', '123456', '王五');

-- 文章样例
INSERT INTO posts (title, content, author_id)
VALUES
    ('第一篇博客', '这是我的第一篇博客，欢迎大家阅读。', 1),
    ('My First Blog Post', 'This is my first blog post. Welcome to read.', 2),
    ('如何学习编程', '编程是一门很有意思的技能，下面是我的学习心得。', 2),
    ('How to Learn Programming', 'Programming is a very interesting skill. Here are my learning experience.', 3),
    ('Python 入门教程', 'Python 是一门非常受欢迎的编程语言，这里是一份简单的入门教程。', 3);

INSERT INTO posts (title, content, author_id, reply_to)
VALUES
    ('回“第一篇博客”', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 1),
    ('', '我就是想回复一下', 2, 1),
    ('另外一片新的文章', '这是新的一片文章，测试看看', 3, 0);
