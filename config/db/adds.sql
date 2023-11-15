CREATE TYPE save_type AS ENUM ('fav');

CREATE TABLE post_saves (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    post_id INTEGER REFERENCES posts(id),
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
    user_id INTEGER REFERENCES users(id),
    post_id INTEGER REFERENCES posts(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    react_id INTEGER REFERENCES reacts(id),
    UNIQUE(user_id, post_id),
    UNIQUE(user_id, post_id, react_id)
);

CREATE INDEX idx_post_reacts_post_id ON post_reacts (post_id);

CREATE TABLE post_subs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    post_id INTEGER REFERENCES posts(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, post_id)
);

CREATE INDEX idx_post_subs_post_id ON post_subs (post_id);

CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    sender_id INTEGER REFERENCES users(id),
    reciever_id INTEGER REFERENCES users(id),
    source_id INTEGER REFERENCES posts(id),
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_read BOOLEAN NOT NULL DEFAULT false
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
    role_id INTEGER REFERENCES roles(id),
    permission_id INTEGER REFERENCES permissions(id),
    UNIQUE(role_id, permission_id)
);

CREATE TABLE user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    role_id INTEGER REFERENCES roles(id),
    UNIQUE(user_id)
);

CREATE TABLE activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    type VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    target_model VARCHAR(255),
    target_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_address VARCHAR(255) NOT NULL,
    device_info VARCHAR(255),
    details TEXT
);

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
('❤', 'thanks', 'Thanks'),
('😀', 'happy', 'Haha'),
('😕', 'confused', 'Confuse'),
('👀', 'eyes', 'Watching'),
('🎉', 'party', 'Yeah');

