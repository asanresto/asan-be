CREATE TYPE "promotion_code_status" AS ENUM ('active', 'used');

CREATE TABLE "promotion_codes" (
    "id" BIGSERIAL PRIMARY KEY,
    "code" TEXT UNIQUE,
    "status" "promotion_code_status" DEFAULT 'active',
    "created_at" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
