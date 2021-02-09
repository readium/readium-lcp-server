CREATE DATABASE lcpserver;
CREATE user lcpserver identified by 'secretpassword';
GRANT ALL ON lcpserver.* TO 'lcpserver'@'%';

CREATE DATABASE lsdserver;
CREATE user lsdserver identified by 'secretpassword';
GRANT ALL ON lsdserver.* TO 'lsdserver'@'%';

CREATE DATABASE testfrontend;
CREATE user testfrontend identified by 'secretpassword';
GRANT ALL ON testfrontend.* TO 'testfrontend'@'%';
