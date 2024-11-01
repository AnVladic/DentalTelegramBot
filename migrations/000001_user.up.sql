CREATE TABLE "User" (
    "id" SERIAL PRIMARY KEY,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "tg_user_id" BIGINT UNIQUE NOT NULL,
    "dental_pro_id" BIGINT UNIQUE,
    "name" VARCHAR(256),
    "lastname" VARCHAR(256),
    "phone" VARCHAR(20)
);
