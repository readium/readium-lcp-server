FROM mariadb:latest

RUN  apt-get update \
  && apt-get install -y wget \
  && rm -rf /var/lib/apt/lists/*

COPY ./scripts/01_create_db_and_users.sql /docker-entrypoint-initdb.d/

RUN wget https://raw.githubusercontent.com/readium/readium-lcp-server/master/dbmodel/mysql_db_setup_lcpserver.sql -O /docker-entrypoint-initdb.d/02_mysql_db_setup_lcpserver.sql \
  && sed -i '1 s/^/USE lcpserver;\n/' /docker-entrypoint-initdb.d/02_mysql_db_setup_lcpserver.sql \
  && wget https://raw.githubusercontent.com/readium/readium-lcp-server/master/dbmodel/mysql_db_setup_lsdserver.sql -O /docker-entrypoint-initdb.d/03_mysql_db_setup_lsdserver.sql \
  && sed -i '1 s/^/USE lsdserver;\n/' /docker-entrypoint-initdb.d/03_mysql_db_setup_lsdserver.sql \
  && wget https://raw.githubusercontent.com/readium/readium-lcp-server/master/dbmodel/mysql_db_setup_frontend.sql -O /docker-entrypoint-initdb.d/04_mysql_db_setup_frontend.sql \
  && sed -i '1 s/^/USE testfrontend;\n/' /docker-entrypoint-initdb.d/04_mysql_db_setup_frontend.sql