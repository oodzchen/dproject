DROP DATABASE IF EXISTS discuss;
DROP ROLE IF EXISTS admin;

CREATE ROLE admin WITH SUPERUSER CREATEDB CREATEROLE LOGIN PASSWORD 'ADMIN_PASSWORD';
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
    root_article_id INTEGER DEFAULT 0 NOT NULL
    -- total_reply_count INTEGER DEFAULT 0 NOT NULL,
    -- child_count INTEGER DEFAULT 0 NOT NULL
);

-- reply_to 为空时候标题不能为空，replay_to 不为空时标题可以为空
ALTER TABLE posts ADD CONSTRAINT posts_reply_to_title_check CHECK (
    (reply_to IS NULL AND title IS NOT NULL) OR
    (reply_to IS NOT NULL)
);

-- CREATE OR REPLACE FUNCTION update_parent_child_count() RETURNS TRIGGER AS $$
--     BEGIN
--         UPDATE posts
-- 	SET child_count = child_count +1
-- 	WHERE id = NEW.reply_to;
-- 	RETURN NULL;
--     END
-- $$ LANGUAGE plpgsql;

-- CREATE OR REPLACE FUNCTION update_depth() RETURNS TRIGGER AS $$
--     DECLARE
--         parent_depth int;
--     BEGIN
-- 	IF NEW.reply_to = 0 THEN
-- 	    RETURN NEW;
-- 	END IF;
	
--         SELECT depth INTO parent_depth
-- 	FROM posts
-- 	WHERE id = NEW.reply_to;

--         NEW.depth = parent_depth + 1;
-- 	RETURN NEW;
--     END
-- $$ LANGUAGE plpgsql;

-- CREATE OR REPLACE FUNCTION pass_root_id() RETURNS TRIGGER AS $$
--     DECLARE
--         root_id int;
--     BEGIN
-- 	IF NEW.reply_to = 0 THEN
-- 	    RETURN NEW;
-- 	END IF;
	
--         SELECT root_article_id INTO root_id
-- 	FROM posts
-- 	WHERE id = NEW.reply_to;
	
--         IF root_id = 0 THEN
-- 	    NEW.root_article_id = NEW.reply_to;
-- 	ELSE
-- 	    NEW.root_article_id = root_id;
-- 	END IF;

-- 	RETURN NEW;
--     END
-- $$ LANGUAGE plpgsql;

-- CREATE OR REPLACE FUNCTION update_parent_total_reply_count() RETURNS TRIGGER AS $$
--     DECLARE
--         parent_id INT;
--     BEGIN
--         parent_id := NEW.reply_to;

--         WHILE parent_id != 0 LOOP
-- 	    UPDATE posts SET total_reply_count = (SELECT COALESCE(SUM(total_reply_count), 0)+COUNT(*) FROM posts WHERE reply_to = parent_id) WHERE id = parent_id;
-- 	    SELECT reply_to INTO parent_id FROM posts WHERE id = parent_id;
-- 	END LOOP;    
--         RETURN NULL;
--     END;
-- $$ LANGUAGE plpgsql;

CREATE RULE post_del_protect AS ON DELETE TO posts DO INSTEAD NOTHING;

-- CREATE OR REPLACE TRIGGER update_parent_child_count_trigger
--     AFTER INSERT ON posts
--     FOR EACH ROW
--     EXECUTE FUNCTION update_parent_child_count();

-- CREATE OR REPLACE TRIGGER update_depth_trigger
--     BEFORE INSERT ON posts
--     FOR EACH ROW
--     EXECUTE FUNCTION update_depth();

-- CREATE OR REPLACE TRIGGER pass_root_id_trigger
--     BEFORE INSERT ON posts
--     FOR EACH ROW
--     EXECUTE FUNCTION pass_root_id();

-- CREATE OR REPLACE TRIGGER update_parent_total_reply_count_trigger
--     AFTER INSERT ON posts
--     FOR EACH ROW
--     EXECUTE FUNCTION update_parent_total_reply_count();

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
    ('', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 1, 1, 1, false),
    ('', '我就是想回复一下', 2, 1, 1, 1, false),
    ('', '我就是想回复一下', 2, 6, 2, 1, false),
    ('', '这是新的一片文章，测试看看', 3, 3, 1, 3, false),
    ('', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 8, 3, 1, false),
    ('', '我是这样想的，哈哈，非常有意思，只是测试而已', 2, 10, 4, 1, true);
