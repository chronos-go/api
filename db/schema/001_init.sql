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
