
CREATE TABLE IF NOT EXISTS event (
  id  int(11) PRIMARY KEY NOT NULL,
  device_name varchar(255) NOT NULL,
  timestamp datetime NOT NULL,
  type  int(11) NOT NULL,
  device_id varchar(255) NOT NULL,
  license_status_fk int(11) NOT NULL
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
  content_fk varchar(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS content (
  id varchar(255) PRIMARY KEY NOT NULL,
  encryption_key varchar(64) NOT NULL,
  location text NOT NULL,
  FOREIGN KEY(id) REFERENCES license(content_fk)
);

CREATE TABLE IF NOT EXISTS license_status (
  id int(11) PRIMARY KEY NOT NULL,
  status int(11) NOT NULL,
  license_updated datetime DEFAULT NULL,
  status_updated datetime DEFAULT NULL,
  device_count int(11) DEFAULT NULL,
  potential_rights_end datetime DEFAULT NULL,
  license_ref varchar(255) NOT NULL,
  FOREIGN KEY(id) REFERENCES event(license_status_fk)
);
