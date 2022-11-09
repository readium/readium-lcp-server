CREATE TABLE license_status (
  id integer IDENTITY PRIMARY KEY,
  status tinyint NOT NULL,
  license_updated datetime NOT NULL,
  status_updated datetime NOT NULL,
  device_count smallint DEFAULT NULL,
  potential_rights_end datetime DEFAULT NULL,
  license_ref varchar(255) NOT NULL,
  rights_end datetime DEFAULT NULL 
);

CREATE INDEX license_ref_index ON license_status (license_ref);

CREATE TABLE event (
	id integer IDENTITY PRIMARY KEY,
	device_name varchar(255) DEFAULT NULL,
	timestamp datetime NOT NULL,
	type int NOT NULL,
	device_id varchar(255) DEFAULT NULL,
	license_status_fk int NOT NULL,
  FOREIGN KEY(license_status_fk) REFERENCES license_status(id)
);

CREATE INDEX license_status_fk_index on event (license_status_fk);