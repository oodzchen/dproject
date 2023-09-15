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
    name VARCHAR(50) NOT NULL
);

CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    front_id VARCHAR(50) NOT NULL,
    name VARCHAR(50) NOT NULL
);

CREATE TABLE role_permissions (
    role_id INTEGER REFERENCES roles(id),
    permission_id INTEGER REFERENCES permissions(id)
);

CREATE TABLE user_roles (
    user_id INTEGER REFERENCES users(id),
    role_id INTEGER REFERENCES roles(id)
);

INSERT INTO roles (front_id, name) VALUES ('user', 'User');
INSERT INTO roles (front_id, name) VALUES ('moderator', 'Moderator');
INSERT INTO roles (front_id, name) VALUES ('admin', 'Admin');

INSERT INTO permissions (front_id, name) VALUES ('create_article', 'Create article');
INSERT INTO permissions (front_id, name) VALUES ('create_reply', 'Create reply');

INSERT INTO permissions (front_id, name) VALUES ('access_manage', 'Access Manage');
INSERT INTO permissions (front_id, name) VALUES ('create_role', 'Create Role');
INSERT INTO permissions (front_id, name) VALUES ('create_permission', 'Create Permission');

INSERT INTO role_permissions (role_id, permission_id) VALUES (1, 1);
INSERT INTO role_permissions (role_id, permission_id) VALUES (1, 2);

-- INSERT INTO user_roles SET (user_id, role_id) VALUES (1, 1);
INSERT INTO user_roles (user_id, role_id)
SELECT id, 1
FROM users;
