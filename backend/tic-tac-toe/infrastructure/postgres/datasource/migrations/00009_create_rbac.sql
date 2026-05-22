-- +goose Up
CREATE TABLE IF NOT EXISTS roles (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS permissions (
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (resource, action)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_uuid TEXT NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    role_name TEXT NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_uuid, role_name)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name TEXT NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_name, resource, action),
    FOREIGN KEY (resource, action) REFERENCES permissions(resource, action) ON DELETE CASCADE
);

INSERT INTO roles (name, description) VALUES
    ('player', 'Default player role'),
    ('admin', 'Administrator role')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (resource, action, description) VALUES
    ('games', 'create', 'Create a new game'),
    ('games', 'list', 'List active games'),
    ('games', 'history', 'List completed games'),
    ('games', 'leaderboard', 'View leaderboard'),
    ('games', 'read', 'Read a game'),
    ('games', 'join', 'Join a game'),
    ('games', 'move', 'Make a move'),
    ('users', 'read_self', 'Read current user'),
    ('users', 'delete_self', 'Delete current user'),
    ('users', 'read_any', 'Read any user')
ON CONFLICT (resource, action) DO NOTHING;

INSERT INTO role_permissions (role_name, resource, action)
SELECT 'player', resource, action
FROM permissions
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_name, resource, action)
SELECT 'admin', resource, action
FROM permissions
ON CONFLICT DO NOTHING;

CREATE INDEX IF NOT EXISTS user_roles_user_uuid_idx
    ON user_roles (user_uuid);

CREATE INDEX IF NOT EXISTS role_permissions_role_name_idx
    ON role_permissions (role_name);

-- +goose Down
DROP INDEX IF EXISTS role_permissions_role_name_idx;
DROP INDEX IF EXISTS user_roles_user_uuid_idx;

DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
