
CREATE TABLE IF NOT EXISTS content (
  id varchar(255) PRIMARY KEY NOT NULL,
  encryption_key varchar(64) NOT NULL,
  location text NOT NULL
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
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  status int(11) NOT NULL,
  license_updated datetime DEFAULT NULL,
  status_updated datetime DEFAULT NULL,
  device_count int(11) DEFAULT NULL,
  potential_rights_end datetime DEFAULT NULL,
  license_ref varchar(255) NOT NULL,
  CONSTRAINT `license_ref_UNIQUE` UNIQUE (`license_ref`)
);
CREATE TABLE IF NOT EXISTS event (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	device_name varchar(255) DEFAULT NULL,
	timestamp datetime NOT NULL,
	type int NOT NULL,
	device_id varchar(255) DEFAULT NULL,
	license_status_fk int NOT NULL,
  FOREIGN KEY(license_status_fk) REFERENCES license_status(id),
  CONSTRAINT license_status_fk_UNIQUE UNIQUE (license_status_fk)
);
