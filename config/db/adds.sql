CREATE TYPE save_type AS ENUM ('fav');

CREATE TABLE post_saves (
user_id INTEGER REFERENCES users(id),
post_id INTEGER REFERENCES posts(id),
created_at TIMESTAMP NOT NULL DEFAULT NOW(),
type save_type
);

CREATE TYPE react_type AS ENUM ('grinning', 'confused', 'eyes', 'party', 'thanks');

CREATE TABLE post_reacts (
user_id INTEGER REFERENCES users(id),
post_id INTEGER REFERENCES posts(id),
created_at TIMESTAMP NOT NULL DEFAULT NOW(),
type react_type
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
    role_id INTEGER REFERENCES roles(id),
    permission_id INTEGER REFERENCES permissions(id),
    UNIQUE(role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id INTEGER REFERENCES users(id),
    role_id INTEGER REFERENCES roles(id),
    UNIQUE(user_id)
);

CREATE TYPE activity_type AS ENUM ('user', 'manage', 'anonymous');

CREATE TABLE user_activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    type activity_type NOT NULL,
    action VARCHAR(255) NOT NULL,
    target_model VARCHAR(255),
    target_id INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_address VARCHAR(255) NOT NULL,
    device_info VARCHAR(255),
    details TEXT
);

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
