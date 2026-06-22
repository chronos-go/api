ALTER TABLE "clients" ALTER COLUMN "id" SET DEFAULT gen_random_uuid();
ALTER TABLE "providers" ALTER COLUMN "id" SET DEFAULT gen_random_uuid();
ALTER TABLE "services" ALTER COLUMN "id" SET DEFAULT gen_random_uuid();
