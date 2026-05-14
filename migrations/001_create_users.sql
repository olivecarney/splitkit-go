CREATE TABLE users (
    id UUID PRIMARY KEY,
    external_auth_id TEXT UNIQUE,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
