CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS clients (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    birth_date DATE        NOT NULL,
    password   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS providers (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    document   TEXT        NOT NULL,
    password   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS services (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id      UUID        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name             TEXT        NOT NULL,
    description      TEXT        NOT NULL DEFAULT '',
    price_cents      INTEGER     NOT NULL CHECK (price_cents >= 0),
    duration_minutes INTEGER     NOT NULL CHECK (duration_minutes > 0),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS services_provider_id_idx
    ON services(provider_id);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT        NOT NULL,
    role       TEXT        NOT NULL CHECK (role IN ('client', 'provider')),
    email      TEXT        NOT NULL,
    family_id  UUID        NOT NULL,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS auth_sessions_family_id_idx ON auth_sessions(family_id);
CREATE INDEX IF NOT EXISTS auth_sessions_expires_at_idx ON auth_sessions(expires_at);
