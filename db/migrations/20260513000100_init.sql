-- Create clients table.
CREATE TABLE "clients" (
    "id" uuid PRIMARY KEY,
    "name" text NOT NULL,
    "email" text NOT NULL UNIQUE,
    "birth_date" date NOT NULL,
    "password" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now()
);

-- Create providers table.
CREATE TABLE "providers" (
    "id" uuid PRIMARY KEY,
    "name" text NOT NULL,
    "email" text NOT NULL UNIQUE,
    "document" text NOT NULL,
    "password" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now()
);

-- Create services table with providers -> services relationship.
CREATE TABLE "services" (
    "id" uuid PRIMARY KEY,
    "provider_id" uuid NOT NULL REFERENCES "providers" ("id") ON DELETE CASCADE,
    "name" text NOT NULL,
    "description" text NOT NULL DEFAULT '',
    "price_cents" integer NOT NULL CHECK ("price_cents" >= 0),
    "duration_minutes" integer NOT NULL CHECK ("duration_minutes" > 0),
    "created_at" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX "services_provider_id_idx"
    ON "services" ("provider_id");
