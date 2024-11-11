CREATE TABLE "User" (
    "id" SERIAL PRIMARY KEY,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "tg_user_id" BIGINT UNIQUE NOT NULL,
    "dental_pro_id" BIGINT UNIQUE,
    "name" VARCHAR(256),
    "lastname" VARCHAR(256),
    "phone" VARCHAR(20)
);


CREATE TABLE "Doctor" (
      "id" BIGINT PRIMARY KEY,
      "fio" VARCHAR(256) NOT NULL
);


CREATE TABLE "Register" (
    "id" SERIAL PRIMARY KEY,
    "user_id" BIGINT NOT NULL REFERENCES "User"("id") ON DELETE CASCADE,
    "message_id" BIGINT NOT NULL,
    "chat_id" BIGINT NOT NULL,
    "doctor_id" BIGINT REFERENCES "Doctor"("id"),
    "appointment_id" BIGINT,
    "datetime" TIMESTAMP,

    UNIQUE (user_id, message_id, chat_id)
);
