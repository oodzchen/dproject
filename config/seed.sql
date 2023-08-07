DROP DATABASE IF EXISTS discuss;
DROP ROLE IF EXISTS admin;

CREATE ROLE admin WITH SUPERUSER CREATEDB CREATEROLE LOGIN PASSWORD '88886666';
CREATE DATABASE discuss OWNER admin ENCODING 'UTF-8';

-- 连接到 discuss 数据库
\c discuss admin

-- 用户
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    introduction TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_admin BOOLEAN NOT NULL DEFAULT false,
    deleted BOOLEAN NOT NULL DEFAULT false,
    banned BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(email)
);

-- 创建文章数据表格
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255),
    author_id INTEGER REFERENCES users(id),
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    reply_to INTEGER DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT false,
    depth INTEGER DEFAULT 0 NOT NULL,
    root_article_id INTEGER DEFAULT 0,
    total_reply_count INTEGER DEFAULT 0 NOT NULL
);

-- reply_to 为空时候标题不能为空，replay_to 不为空时标题可以为空
ALTER TABLE posts ADD CONSTRAINT posts_reply_to_title_check CHECK (
    (reply_to IS NULL AND title IS NOT NULL) OR
    (reply_to IS NOT NULL)
);

CREATE OR REPLACE FUNCTION recursive_count_reply() RETURNS TRIGGER AS $$
    DECLARE
        root_id int;
    BEGIN
        IF (TG_OP = 'INSERT') THEN
	    root_id = NEW.reply_to;
	ELSE
	    root_id = OLD.reply_to;
	END IF;
	
        WITH RECURSIVE RecurPosts AS (
            SELECT id, reply_to, deleted, 0::bigint AS child_count
            FROM posts p
            WHERE p.id = root_id AND p.deleted = false
        
            UNION ALL
        
            SELECT p1.id, p1.reply_to, p1.deleted, rp.child_count + subquery.child_count
            FROM posts p1
            INNER JOIN RecurPosts rp ON p1.reply_to = rp.id
            INNER JOIN (SELECT reply_to, COUNT(*) AS child_count FROM posts WHERE deleted = false GROUP BY reply_to) AS subquery
            ON rp.id = subquery.reply_to
            WHERE p1.deleted = false
        )
        , MaxChildCount AS (
            SELECT MAX(child_count) AS max_child_count FROM RecurPosts
        )
        UPDATE posts p3 SET total_reply_count = mc.max_child_count
        FROM MaxChildCount mc
        WHERE p3.id = root_id;
	RETURN NULL;
    END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER update_reply_count
    AFTER INSERT OR UPDATE OR DELETE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION recursive_count_reply();

-- 插入样例数据
-- 用户样例数据
INSERT INTO users (email, password, name, introduction, is_admin)
VALUES
    ('oodzchen@gmail.com', '666666@#$abc', '欧辰', '这是欧辰的自我介绍', true),
    ('zhangsan@example.com', '123456@#abc', '张三', '这是张三的自我介绍', false),
    ('lisi@example.com', '123456!@abc', '李四', '这是李四的自我介绍', false),
    ('wangwu@example.com', '123456!@ABC', '王五', '这是王五的自我介绍', false),
    ('mazi@example.com', '123456$#abc', '麻子', '这是麻子的自我介绍', false);

-- 文章样例
INSERT INTO posts (title, content, author_id)
VALUES
    ('第一篇博客', '这是我的第一篇博客，欢迎大家阅读。', 1),
    ('My First Blog Post', 'This is my first blog post. Welcome to read.', 2),
    ('如何学习编程', '编程是一门很有意思的技能，下面是我的学习心得。', 2),
    ('How to Learn Programming', 'Programming is a very interesting skill. Here are my learning experience.', 3),
    ('Python 入门教程', 'Python 是一门非常受欢迎的编程语言，这里是一份简单的入门教程。', 3);

INSERT INTO posts (title, content, author_id, reply_to, depth, root_article_id, deleted)
VALUES
    ('回“第一篇博客”', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 1, 1, 1, false),
    ('', '我就是想回复一下', 2, 1, 1, 1, false),
    ('', '我就是想回复一下', 2, 6, 2, 1, false),
    ('另外一片新的文章', '这是新的一片文章，测试看看', 3, 3, 1, 3, false),
    ('回“第一篇博客”', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 8, 3, 1, false),
    ('回“第一篇博客”', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 10, 4, 1, true);
