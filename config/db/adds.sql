CREATE TYPE save_type AS ENUM ('fav');

CREATE TABLE post_saves (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    post_id INTEGER REFERENCES posts(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    type save_type,
    UNIQUE(user_id, post_id)
);

CREATE INDEX idx_post_saves_post_id ON post_saves (post_id);

CREATE TABLE reacts (
    id SERIAL PRIMARY KEY,
    emoji VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    front_id VARCHAR(50) NOT NULL,
    describe VARCHAR(255),
    UNIQUE(front_id)
);

CREATE TABLE post_reacts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    post_id INTEGER REFERENCES posts(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    react_id INTEGER REFERENCES reacts(id) NOT NULL,
    UNIQUE(user_id, post_id),
    UNIQUE(user_id, post_id, react_id)
);

CREATE INDEX idx_post_reacts_post_id ON post_reacts (post_id);

CREATE TABLE post_subs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    post_id INTEGER REFERENCES posts(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, post_id)
);

CREATE INDEX idx_post_subs_post_id ON post_subs (post_id);

CREATE TYPE message_type AS ENUM ('reply', 'category', 'system');

CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    sender_id INTEGER REFERENCES users(id) NOT NULL,
    reciever_id INTEGER REFERENCES users(id) NOT NULL,
    source_article_id INTEGER REFERENCES posts(id),
    source_category_id INTEGER REFERENCES posts(id),
    content_id INTEGER REFERENCES posts(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_read BOOLEAN NOT NULL DEFAULT false,
    type message_type NOT NULL
);

CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    front_id VARCHAR(50) NOT NULL,
    name VARCHAR(50) NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_default BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(front_id)
);

-- CREATE TYPE permission_module AS ENUM ('user', 'article', 'permission', 'role');

CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    front_id VARCHAR(50) NOT NULL,
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    module VARCHAR(50) NOT NULL,
    UNIQUE(front_id)
);

CREATE TABLE role_permissions (
    id SERIAL PRIMARY KEY,
    role_id INTEGER REFERENCES roles(id) NOT NULL,
    permission_id INTEGER REFERENCES permissions(id) NOT NULL,
    UNIQUE(role_id, permission_id)
);

CREATE TABLE user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    role_id INTEGER REFERENCES roles(id) NOT NULL,
    UNIQUE(user_id)
);

CREATE TABLE activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    type VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    target_model VARCHAR(255),
    target_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_address VARCHAR(255) NOT NULL,
    device_info VARCHAR(255),
    details TEXT
);

CREATE TABLE tags (
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

CREATE TABLE post_tags (
    id SERIAL PRIMARY KEY,
    post_id INTEGER REFERENCES posts(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    tag_id INTEGER REFERENCES tags(id) NOT NULL
);

CREATE INDEX idx_post_tags_post_id ON post_tags (post_id);

CREATE TABLE post_history (
    id SERIAL PRIMARY KEY,
    post_id INTEGER REFERENCES posts(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    operator_id INTEGER REFERENCES users(id) NOT NULL,
    curr TIMESTAMP NOT NULL,
    prev TIMESTAMP NOT NULL,
    version_num INTEGER NOT NULL,
    title_delta TEXT,
    url_delta TEXT,
    content_delta TEXT,
    category_front_delta TEXT
);

ALTER TABLE post_history ADD COLUMN is_hidden BOOLEAN NOT NULL DEFAULT false;

-- CREATE TABLE verification_codes (
--     id SERIAL PRIMARY KEY,
--     email VARCHAR(255) NOT NULL,
--     code VARCHAR(255) NOT NULL,
--     created_at TIMESTAMP NOT NULL DEFAULT NOW(),
--     expired_at TIMESTAMP NOT NULL DEFAULT NOW(),
--     is_used BOOLEAN NOT NULL DEFAULT false
-- )

-- INSERT INTO roles (front_id, name) VALUES ('user', 'User');
-- INSERT INTO roles (front_id, name) VALUES ('moderator', 'Moderator');
-- INSERT INTO roles (front_id, name) VALUES ('admin', 'Admin');

-- INSERT INTO permissions (front_id, name, module) VALUES ('create_article', 'Create article', 'article');
-- INSERT INTO permissions (front_id, name, module) VALUES ('create_reply', 'Create reply', 'article');

-- INSERT INTO permissions (front_id, name, module) VALUES ('create_role', 'Create Role', 'role');
-- INSERT INTO permissions (front_id, name, module) VALUES ('create_permission', 'Create Permission', 'permission');

-- INSERT INTO role_permissions (role_id, permission_id) VALUES (1, 1);
-- INSERT INTO role_permissions (role_id, permission_id) VALUES (1, 2);

-- INSERT INTO user_roles SET (user_id, role_id) VALUES (1, 1);
-- INSERT INTO user_roles (user_id, role_id)
-- SELECT id, 1
-- FROM users;

INSERT INTO reacts (emoji, front_id, describe) VALUES
('♥️', 'thanks', 'Thanks'),
('😀', 'happy', 'Haha'),
('😕', 'confused', 'Confuse'),
('👀', 'eyes', 'Watching'),
('🎉', 'party', 'Yeah');


INSERT INTO tags (front_id, name, author_id, describe, approved)
VALUES
('linux', 'Linux', 1, '', true),
('golang', 'Go', 1, '', true),
('clang', 'C', 1, '', true),
('cpp', 'C++', 1, '', true),
('emacs', 'Emacs', 1, '', true),
('ai', '人工智能', 1, '', true),
('typescript', 'TypeScript', 1, '', true);
