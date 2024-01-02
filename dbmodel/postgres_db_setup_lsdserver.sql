CREATE TABLE license_status (
  id serial4 NOT NULL,
  status smallint NOT NULL,
  license_updated timestamp(3) NOT NULL,
  status_updated timestamp(3) NOT NULL,
  device_count smallint DEFAULT NULL,
  potential_rights_end timestamp(3) DEFAULT NULL,
  license_ref varchar(255) NOT NULL,
  rights_end timestamp(3) DEFAULT NULL,
  CONSTRAINT license_status_pkey PRIMARY KEY (id)
);

CREATE INDEX license_ref_index ON license_status (license_ref);

CREATE TABLE event (
	id serial4 NOT NULL,
	device_name varchar(255) DEFAULT NULL,
	timestamp timestamp(3) NOT NULL,
	type int NOT NULL,
	device_id varchar(255) DEFAULT NULL,
	license_status_fk int NOT NULL,
  CONSTRAINT event_pkey PRIMARY KEY (id),
  FOREIGN KEY(license_status_fk) REFERENCES license_status(id)
);

CREATE INDEX license_status_fk_index on event (license_status_fk);