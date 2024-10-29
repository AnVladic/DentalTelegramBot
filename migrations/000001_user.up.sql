CREATE TABLE "User" (
    "id" BIGINT PRIMARY KEY,
    "tg_user_id" BIGINT UNIQUE NOT NULL,
    "name" VARCHAR(256),
    "lastname" VARCHAR(256),
    "phone" VARCHAR(20)
);
