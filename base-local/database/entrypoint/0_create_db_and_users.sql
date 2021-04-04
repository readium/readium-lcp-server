CREATE USER lcpserver WITH PASSWORD 'postgres';-- CREATEDB CREATEUSER ;
CREATE DATABASE lcpserver owner lcpserver;
-- CREATE user lcpserver identified by "postgres";
-- GRANT ALL ON lcpserver.* TO "lcpserver"@"%";

CREATE USER lsdserver WITH PASSWORD 'postgres';-- CREATEDB CREATEUSER ;
CREATE DATABASE lsdserver owner lsdserver;
-- CREATE user lsdserver identified by "postgres";
-- GRANT ALL ON lsdserver.* TO "lsdserver"@"%";

CREATE USER testfrontend WITH PASSWORD 'postgres';-- CREATEDB CREATEUSER ;
CREATE DATABASE testfrontend owner testfrontend;
-- CREATE user testfrontend identified by "postgres";
-- GRANT ALL ON testfrontend.* TO "testfrontend"@"%";
