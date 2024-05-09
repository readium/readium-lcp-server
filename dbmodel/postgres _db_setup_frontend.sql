CREATE SEQUENCE publication_seq;

CREATE TABLE publication (
    id int PRIMARY KEY DEFAULT NEXTVAL ('publication_seq'),
    uuid varchar(255) NOT NULL,	/* == content id */
    title varchar(255) NOT NULL,
    status varchar(255) NOT NULL
);

CREATE INDEX uuid_index ON publication (uuid);

CREATE SEQUENCE user_seq;

CREATE TABLE "user" (
    id int PRIMARY KEY DEFAULT NEXTVAL ('user_seq'),
    uuid varchar(255) NOT NULL,
    name varchar(64) NOT NULL,
    email varchar(64) NOT NULL,
    password varchar(64) NOT NULL,
    hint varchar(64) NOT NULL
);

CREATE SEQUENCE purchase_seq;

CREATE TABLE purchase (
    id int PRIMARY KEY DEFAULT NEXTVAL ('purchase_seq'),
    uuid varchar(255) NOT NULL,
    publication_id int NOT NULL,
    user_id int NOT NULL,
    license_uuid varchar(255) NULL,
    type varchar(32) NOT NULL,
    transaction_date timestamp(0),
    start_date timestamp(0),
    end_date timestamp(0),
    status varchar(255) NOT NULL,
    FOREIGN KEY (publication_id) REFERENCES publication (id),
    FOREIGN KEY (user_id) REFERENCES "user" (id)
);

CREATE INDEX idx_purchase ON purchase (license_uuid);

CREATE SEQUENCE license_view_seq;

CREATE TABLE license_view (
    id int PRIMARY KEY DEFAULT NEXTVAL ('license_view_seq'),
    uuid varchar(255) NOT NULL,
    device_count int NOT NULL,
    status varchar(255) NOT NULL,
    message varchar(255) NOT NULL
);