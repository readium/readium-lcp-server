version: "3.6"

x-app: &default-app
  restart: always
  environment:
    - "READIUM_DATABASE_HOST=${READIUM_DATABASE_HOST}"
    - "READIUM_DATABASE_PORT=${READIUM_DATABASE_PORT}"
    - "READIUM_DATABASE_USERNAME=${READIUM_DATABASE_USERNAME}"
    - "READIUM_DATABASE_PASSWORD=${READIUM_DATABASE_PASSWORD}"
    - "READIUM_LCPSERVER_HOST=http://${READIUM_LCPSERVER_HOST}:${READIUM_LCPSERVER_PORT}"
    - "READIUM_LCPSERVER_PORT=${READIUM_LCPSERVER_PORT}"
    - "READIUM_LCPSERVER_DATABASE=${READIUM_LCPSERVER_DATABASE}"
    - "READIUM_LCPSERVER_USERNAME=${READIUM_LCPSERVER_USERNAME}"
    - "READIUM_LCPSERVER_PASSWORD=${READIUM_LCPSERVER_PASSWORD}"
    - "READIUM_LSDSERVER_HOST=http://${READIUM_LSDSERVER_HOST}:${READIUM_LSDSERVER_PORT}"
    - "READIUM_LSDSERVER_PORT=${READIUM_LSDSERVER_PORT}"
    - "READIUM_LSDSERVER_DATABASE=${READIUM_LSDSERVER_DATABASE}"
    - "READIUM_FRONTEND_HOST=http://${READIUM_FRONTEND_HOST}:${READIUM_FRONTEND_PORT}"
    - "READIUM_FRONTEND_PORT=${READIUM_FRONTEND_PORT}"
    - "READIUM_FRONTEND_DATABASE=${READIUM_FRONTEND_DATABASE}"
    - "READIUM_ENC_CONTENT=/opt/readium/files/encrypted"

services:
  database:
    build: ./database
    image: database
    restart: always
    ports:
      - "${READIUM_DATABASE_EXTERNAL_PORT}:${READIUM_DATABASE_PORT}"
    environment:
      MYSQL_ROOT_PASSWORD: "${READIUM_DATABASE_PASSWORD}"
    volumes:
      - "dbdata:/var/lib/mysql"

  sftp:
    image: "atmoz/sftp:alpine"
    restart: always
    ports:
      - "${READIUM_SFTP_EXTERNAL_PORT}:${READIUM_SFTP_PORT}"
    volumes:
      - "./files/users.conf:/etc/sftp/users.conf:ro"
      - "rawfiles:/home"

  lcpencrypt:
    <<: *default-app
    image: "readium/lcpencrypt:working"
    build:
      context: .
      target: "lcpencrypt"
    volumes:
      - "encfiles:/opt/readium/files/encrypted"
      - "rawfiles:/opt/readium/files/raw"

  lcpserver:
    <<: *default-app
    image: "readium/lcpserver:working"
    build:
      context: .
      target: "lcpserver"
    ports:
      - "${READIUM_LCPSERVER_EXTERNAL_PORT}:${READIUM_LCPSERVER_PORT}"
    volumes:
      - "encfiles:/opt/readium/files/encrypted"
      - "./etc:/etc/readium"
    depends_on:
      - database

  lsdserver:
    <<: *default-app
    image: "readium/lsdserver:working"
    build:
      context: .
      target: "lsdserver"
    ports:
      - "${READIUM_LSDSERVER_EXTERNAL_PORT}:${READIUM_LSDSERVER_PORT}"
    volumes:
      - "./etc:/etc/readium"
    depends_on:
      - database

  testfrontend:
    <<: *default-app
    image: "readium/testfrontend:working"
    build:
      context: .
      target: "testfrontend"
    ports:
      - "${READIUM_FRONTEND_EXTERNAL_PORT}:${READIUM_FRONTEND_PORT}"
    volumes:
      - "encfiles:/opt/readium/files/encrypted"
      - "rawfiles:/opt/readium/files/raw"
      - "./etc:/etc/readium"
    depends_on:
      - database

volumes:
  encfiles:
  dbdata:
  rawfiles:
