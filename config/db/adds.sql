-- CREATE TYPE save_type AS ENUM ('fav');
-- CREATE TABLE post_saves (
-- user_id INTEGER REFERENCES users(id),
-- post_id INTEGER REFERENCES posts(id),
-- created_at TIMESTAMP NOT NULL DEFAULT NOW(),
-- type save_type
-- );

-- CREATE TYPE react_type AS ENUM ('grinning', 'confused', 'eyes', 'party', 'thanks');
-- CREATE TABLE post_reacts (
-- user_id INTEGER REFERENCES users(id),
-- post_id INTEGER REFERENCES posts(id),
-- created_at TIMESTAMP NOT NULL DEFAULT NOW(),
-- type react_type
-- );

CREATE TABLE roles (
    id INT PRIMARY KEY,
    front_id VARCHAR(50) NOT NULL,
    comment VARCHAR(50) NOT NULL
);

CREATE TABLE permissions (
    id INT PRIMARY KEY,
    front_id VARCHAR(50) NOT NULL,
    comment VARCHAR(50) NOT NULL
);

CREATE TABLE role_permissions (
    role_id INTEGER REFERENCES roles(id),
    permission_id INTEGER REFERENCES permissions(id),
);

CREATE TABLE user_roles (
    user_id INTEGER REFERENCES users(id),
    role_id INTEGER REFERENCES roles(id),
);

ALTER TABLE users
ADD COLUMN main_role_id int NOT NULL DEFAULT 1;

ALTER TABLE users
ADD CONSTRAINT fk_user_role
FOREIGN KEY (main_role_id)
REFERENCES roles(id);

INSERT INTO roles SET (front_id, comment) VALUES ('user', 'Normal User');
INSERT INTO roles SET (front_id, comment) VALUES ('modrator', 'Modrator');
INSERT INTO roles SET (front_id, comment) VALUES ('admin', 'Website Admin');

INSERT INTO permissions SET (front_id, comment) VALUES ('create_article', 'Create article');
INSERT INTO permissions SET (front_id, comment) VALUES ('create_reply', 'Create reply');
INSERT INTO permissions SET (front_id, comment) VALUES ('create_role', 'Create Role');
INSERT INTO permissions SET (front_id, comment) VALUES ('create_permission', 'Create Permission');

-- INSERT INTO user_roles SET (user_id, role_id) VALUES (1, 1);
INSERT INTO user_roles (user_id, role_id)
SELECT id, 1
FROM users;
