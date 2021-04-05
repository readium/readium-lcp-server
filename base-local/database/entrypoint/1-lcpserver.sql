\connect lcpserver;

CREATE TABLE "content" (
    "id" varchar(255) PRIMARY KEY NOT NULL,
    "encryption_key" bytea NOT NULL,
    "location" text NOT NULL,
    "length" bigint,
    "sha256" varchar(64),
    "type" varchar(255) NOT NULL DEFAULT 'application/epub+zip'
);

CREATE TABLE "license" (
    "id" varchar(255) PRIMARY KEY NOT NULL,
    "user_id" varchar(255) NOT NULL,
    "provider" varchar(255) NOT NULL,
    "issued" timestamp NOT NULL,
    "updated" timestamp DEFAULT NULL,
    "rights_print" int DEFAULT NULL,
    "rights_copy" int DEFAULT NULL,
    "rights_start" timestamp DEFAULT NULL,
    "rights_end" timestamp DEFAULT NULL,
    "content_fk" varchar(255) NOT NULL,
    "lsd_status" int default 0,
    FOREIGN KEY(content_fk) REFERENCES content(id)
);