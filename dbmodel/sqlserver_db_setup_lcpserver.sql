CREATE TABLE content (
    id varchar(255) PRIMARY KEY NOT NULL,
    encryption_key varbinary(64) NOT NULL,
    location text NOT NULL,
    length int,
    sha256 varchar(64),
    type varchar(255)
);

CREATE TABLE license (
    id varchar(255) PRIMARY KEY NOT NULL,
    user_id varchar(255) NOT NULL,
    provider varchar(255) NOT NULL,
    issued datetime NOT NULL,
    updated datetime DEFAULT NULL,
    rights_print smallint DEFAULT NULL,
    rights_copy smallint DEFAULT NULL,
    rights_start datetime DEFAULT NULL,
    rights_end datetime DEFAULT NULL,
    content_fk varchar(255) NOT NULL,
    lsd_status tinyint default 0,
    FOREIGN KEY(content_fk) REFERENCES content(id)
);