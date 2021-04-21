\connect testfrontend;

CREATE TABLE "publication" (
    "id" serial PRIMARY KEY,
    "uuid" varchar(255) NOT NULL,	/* == content id */
    "title" varchar(255) NOT NULL,
    "status" varchar(255) NOT NULL
);

CREATE INDEX uuid_index ON publication ("uuid");

CREATE TABLE "users" (
    "id" serial PRIMARY KEY,
    "uuid" varchar(255) NOT NULL,
    "name" varchar(64) NOT NULL,
    "email" varchar(64) NOT NULL,
    "password" varchar(64) NOT NULL,
    "hint" varchar(64) NOT NULL
);

CREATE TABLE "purchase" (
    "id" serial PRIMARY KEY,
    "uuid" varchar(255) NOT NULL,
    "publication_id" int NOT NULL,
    "user_id" int NOT NULL,
    "license_uuid" varchar(255) NULL,
    "type" varchar(32) NOT NULL,
    "transaction_date" timestamp,
    "start_date" timestamp,
    "end_date" timestamp,
    "status" varchar(255) NOT NULL,
    FOREIGN KEY ("publication_id") REFERENCES "publication" ("id"),
    FOREIGN KEY ("user_id") REFERENCES "users" ("id")
);

CREATE INDEX "idx_purchase" ON "purchase" ("license_uuid");

CREATE TABLE "license_view" (
    "id" serial PRIMARY KEY,
    "uuid" varchar(255) NOT NULL,
    "device_count" int NOT NULL,
    "status" varchar(255) NOT NULL,
    "message" varchar(255) NOT NULL
);