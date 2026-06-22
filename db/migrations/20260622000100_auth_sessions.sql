CREATE TABLE "auth_sessions" (
    "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id" text NOT NULL,
    "role" text NOT NULL CHECK ("role" IN ('client', 'provider')),
    "email" text NOT NULL,
    "family_id" uuid NOT NULL,
    "token_hash" text NOT NULL UNIQUE,
    "expires_at" timestamptz NOT NULL,
    "used_at" timestamptz NULL,
    "revoked_at" timestamptz NULL,
    "created_at" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX "auth_sessions_family_id_idx" ON "auth_sessions" ("family_id");
CREATE INDEX "auth_sessions_expires_at_idx" ON "auth_sessions" ("expires_at");
