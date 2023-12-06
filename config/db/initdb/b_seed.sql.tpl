DROP DATABASE IF EXISTS :db_name;
DROP ROLE IF EXISTS :db_user;

CREATE ROLE :db_user WITH SUPERUSER CREATEDB CREATEROLE LOGIN PASSWORD :admin_password;
CREATE DATABASE :db_name OWNER :db_user ENCODING 'UTF-8';

-- 连接到 discuss 数据库
\c :db_name :db_user;

CREATE TYPE auth_type AS ENUM ('self', 'google', 'github', 'microsoft');

-- 用户
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255),
    username VARCHAR(255) NOT NULL,
    introduction TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    super_admin BOOLEAN NOT NULL DEFAULT false,
    deleted BOOLEAN NOT NULL DEFAULT false,
    banned BOOLEAN NOT NULL DEFAULT false,
    auth_from auth_type NOT NULL DEFAULT 'self',
    UNIQUE(email),
    UNIQUE(username)
);

ALTER TABLE users ADD CONSTRAINT user_password_check CHECK(
    (auth_from = 'self' AND password IS NOT NULL) OR
    (auth_from <> 'self')
);

CREATE UNIQUE INDEX idx_unique_username
ON users (LOWER(username));

CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    front_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    describe TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    author_id INTEGER REFERENCES users(id) NOT NULL,
    approved BOOLEAN NOT NULL DEFAULT false,
    approval_comment TEXT,
    deleted BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(front_id)
);

CREATE TABLE category_subs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    category_id INTEGER REFERENCES categories(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, category_id)
);

CREATE TABLE category_ignores (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    category_id INTEGER REFERENCES categories(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, category_id)
);

-- 创建文章数据表格
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255),
    url VARCHAR(255),
    author_id INTEGER REFERENCES users(id) NOT NULL,
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    reply_to INTEGER DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT false,
    depth INTEGER DEFAULT 0 NOT NULL,
    root_article_id INTEGER DEFAULT 0 NOT NULL,
    list_weight DOUBLE PRECISION DEFAULT 0 NOT NULL,
    participate_count INTEGER DEFAULT 0 NOT NULL,
    reply_weight DOUBLE PRECISION DEFAULT 0 NOT NULL,
    category_id INTEGER REFERENCES categories(id) NOT NULL,
    locked BOOLEAN NOT NULL DEFAULT false,
    pinned_expire_at TIMESTAMP
);

CREATE INDEX idx_posts_reply_to ON posts (reply_to);
CREATE INDEX idx_posts_root_article_id ON posts (root_article_id);

CREATE TYPE vote_type AS ENUM ('up', 'down');

CREATE TABLE post_votes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    post_id INTEGER REFERENCES posts(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    type vote_type,
    UNIQUE(user_id, post_id)
);

CREATE INDEX idx_posts_votes_post_id ON post_votes (post_id);

-- reply_to 为空时候标题不能为空，replay_to 不为空时标题可以为空
ALTER TABLE posts ADD CONSTRAINT posts_reply_to_title_check CHECK (
    (reply_to IS NULL AND title IS NOT NULL) OR
    (reply_to IS NOT NULL)
);


CREATE RULE post_del_protect AS ON DELETE TO posts DO INSTEAD NOTHING;


INSERT INTO users (email, password, username, introduction, super_admin)
VALUES
('anonymous@example.com', :user_default_password, 'anonymous', 'Anonymous placeholder', false),
('oodzchen@gmail.com', :user_default_password, 'oodzchen', '这是欧辰的自我介绍', true),
('zhangsan@example.com', :user_default_password, 'zhangsan', '这是张三的自我介绍', false),
('lisi@example.com', :user_default_password, 'lisi', '这是李四的自我介绍', false),
('wangwu@example.com', :user_default_password, 'wangwu', '这是王五的自我介绍', false),
('mazi@example.com', :user_default_password, 'mazi', '这是麻子的自我介绍', false);

INSERT INTO categories (front_id, name, author_id, describe, approved)
VALUES
-- ('computer-science', '计算机技术', 1, '所有与计算机相关的技术讨论，包括不限于编程、操作系统、数据库、机器学习等', true),
-- ('internet', '网上冲浪', 1, '互联网冲浪时发现的值得分享和讨论的文章、博客、推文等', true),
('hacker-news', '黑客新闻', 1, '黑客新闻', true),
('qna', '问与答', 1, '任何你想问的', true),
('show', '作品展示', 1, '展示你参与的项目', true),
('dizkaz', '笛卡', 1, '本站点功能建议和Bug反馈', true),
('general', '常规', 1, '暂时找不到分类的内容', true);

-- -- 文章样例
-- INSERT INTO posts (title, content, author_id)
-- VALUES
-- ('第一篇博客', '这是我的第一篇博客，欢迎大家阅读。', 1),
-- ('My First Blog Post', 'This is my first blog post. Welcome to read.', 2),
-- ('如何学习编程', '编程是一门很有意思的技能，下面是我的学习心得。', 2),
-- ('How to Learn Programming', 'Programming is a very interesting skill. Here are my learning experience.', 3),
-- ('Python 入门教程', 'Python 是一门非常受欢迎的编程语言，这里是一份简单的入门教程。', 3);

-- INSERT INTO posts (title, content, author_id, reply_to, depth, root_article_id, deleted)
-- VALUES
-- ('', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 1, 1, 1, false),
-- ('', '我就是想回复一下', 2, 1, 1, 1, false),
-- ('', '我就是想回复一下', 2, 6, 2, 1, false),
-- ('', '这是新的一片文章，测试看看', 3, 3, 1, 3, false),
-- ('', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 8, 3, 1, false),
-- ('', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 10, 4, 1, true);
