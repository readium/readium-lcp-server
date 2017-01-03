
CREATE TABLE IF NOT EXISTS content (
  id varchar(255) PRIMARY KEY NOT NULL,
  encryption_key varchar(64) NOT NULL,
  location text NOT NULL,
  length int(11) NOT NULL,
  sha256 varchar(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS license (
  id varchar(255) PRIMARY KEY NOT NULL,
  user_id varchar(255) NOT NULL,
  provider varchar(255) NOT NULL,
  issued datetime NOT NULL,
  updated datetime DEFAULT NULL,
  rights_print int(11) DEFAULT NULL,
  rights_copy int(11) DEFAULT NULL,
  rights_start datetime DEFAULT NULL,
  rights_end datetime DEFAULT NULL,
  user_key_hint text NOT NULL,
  user_key_hash varchar(64) NOT NULL,
  user_key_algorithm varchar(255) NOT NULL,
  content_fk varchar(255) NOT NULL,
  FOREIGN KEY(content_fk) REFERENCES content(id)
);

CREATE TABLE IF NOT EXISTS license_status (
  id INTEGER PRIMARY KEY,
  status int(11) NOT NULL,
  license_updated datetime NOT NULL,
  status_updated datetime NOT NULL,
  device_count int(11) DEFAULT NULL,
  potential_rights_end datetime DEFAULT NULL,
  license_ref varchar(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS license_ref_index on license_status (license_ref);

CREATE TABLE IF NOT EXISTS event (
	id INTEGER PRIMARY KEY,
	device_name varchar(255) DEFAULT NULL,
	timestamp datetime NOT NULL,
	type int NOT NULL,
	device_id varchar(255) DEFAULT NULL,
	license_status_fk int NOT NULL,
  FOREIGN KEY(license_status_fk) REFERENCES license_status(id)
);

CREATE INDEX IF NOT EXISTS license_status_fk_index on event (license_status_fk);
