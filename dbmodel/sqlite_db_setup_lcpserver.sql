CREATE TABLE content (
  id varchar(255) PRIMARY KEY NOT NULL,
  encryption_key varchar(64) NOT NULL,
  location text NOT NULL, 
  length bigint,
  sha256 varchar(64),
  "type" varchar(255) NOT NULL DEFAULT 'application/epub+zip'
);

CREATE TABLE license (
  id varchar(255) PRIMARY KEY NOT NULL,
  user_id varchar(255) NOT NULL,
  provider varchar(255) NOT NULL,
  issued datetime NOT NULL,
  updated datetime DEFAULT NULL,
  rights_print int(11) DEFAULT NULL,
  rights_copy int(11) DEFAULT NULL,
  rights_start datetime DEFAULT NULL,
  rights_end datetime DEFAULT NULL,
  content_fk varchar(255) NOT NULL,
  lsd_status integer default 0,
  FOREIGN KEY(content_fk) REFERENCES content(id)
);