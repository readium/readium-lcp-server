CREATE TABLE publication (
  id integer IDENTITY PRIMARY KEY,
  uuid varchar(255) NOT NULL,
  title varchar(255) NOT NULL,
  status varchar(255) NOT NULL
);

CREATE INDEX uuid_index ON publication (uuid);

CREATE TABLE "user" (
  id integer IDENTITY PRIMARY KEY,
  uuid varchar(255) NOT NULL,
  name varchar(64) NOT NULL,
  email varchar(64) NOT NULL,
  password varchar(64) NOT NULL,
  hint varchar(64) NOT NULL
);

CREATE TABLE purchase (
  id integer IDENTITY PRIMARY KEY,
  uuid varchar(255) NOT NULL,
  publication_id integer NOT NULL,
  user_id integer NOT NULL,
  license_uuid varchar(255) NULL,
  type varchar(32) NOT NULL,
  transaction_date datetime,
  start_date datetime,
  end_date datetime,
  status varchar(255) NOT NULL,
  FOREIGN KEY (publication_id) REFERENCES publication(id),
  FOREIGN KEY (user_id) REFERENCES "user"(id)
);
  
CREATE INDEX idx_purchase ON purchase (license_uuid);


CREATE TABLE license_view (
  id integer IDENTITY PRIMARY KEY,
  uuid varchar(255) NOT NULL,
  device_count smallint NOT NULL,
  status varchar(255) NOT NULL,
  message varchar(255) NOT NULL
);
