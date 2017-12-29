BEGIN TRANSACTION;
 
ALTER TABLE license RENAME TO temp_license;
 
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

INSERT INTO license 
SELECT
 id, user_id, provider, issued, updated, rights_print, rights_copy, rights_start, rights_end, content_fk, lsd_status
FROM
 temp_license;
 
DROP TABLE temp_license;
 
COMMIT;