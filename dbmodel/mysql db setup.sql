
CREATE TABLE `content` (
    `id` varchar(255) PRIMARY KEY NOT NULL,
    `encryption_key` varbinary(64) NOT NULL,
    `location` text NOT NULL,
    `length` bigint(20),
    `sha256` varchar(64)
);

CREATE TABLE `license` (
    `id` varchar(255) PRIMARY KEY NOT NULL,
    `user_id` varchar(255) NOT NULL,
    `provider` varchar(255) NOT NULL,
    `issued` datetime NOT NULL,
    `updated` datetime DEFAULT NULL,
    `rights_print` int(11) DEFAULT NULL,
    `rights_copy` int(11) DEFAULT NULL,
    `rights_start` datetime DEFAULT NULL,
    `rights_end` datetime DEFAULT NULL,
    `content_fk` varchar(255) NOT NULL,
    `lsd_status` int(11) default 0,
    FOREIGN KEY(content_fk) REFERENCES content(id)
);

CREATE TABLE `license_status` (
    `id` int(11) PRIMARY KEY,
    `status` int(11) NOT NULL,
    `license_updated` datetime NOT NULL,
    `status_updated` datetime NOT NULL,
    `device_count` int(11) DEFAULT NULL,
    `potential_rights_end` datetime DEFAULT NULL,
    `license_ref` varchar(255) NOT NULL,
    `rights_end` datetime DEFAULT NULL 
);

CREATE INDEX `license_ref_index` ON `license_status` (`license_ref`);

CREATE TABLE `event` (
    `id` int(11) PRIMARY KEY,
    `device_name` varchar(255) DEFAULT NULL,
    `timestamp` datetime NOT NULL,
    `type` int NOT NULL,
    `device_id` varchar(255) DEFAULT NULL,
    `license_status_fk` int NOT NULL,
    FOREIGN KEY(`license_status_fk`) REFERENCES `license_status` (`id`)
);

CREATE INDEX `license_status_fk_index` on `event` (`license_status_fk`);

CREATE TABLE `publication` (
    `id` int(11) NOT NULL PRIMARY KEY,
    `uuid` varchar(255) NOT NULL,	/* == content id */
    `title` varchar(255) NOT NULL,
    `status` varchar(255) NOT NULL
);

CREATE INDEX uuid_index ON publication (`uuid`);

CREATE TABLE `user` (
    `id` int(11) NOT NULL PRIMARY KEY,
    `uuid` varchar(255) NOT NULL,
    `name` varchar(64) NOT NULL,
    `email` varchar(64) NOT NULL,
    `password` varchar(64) NOT NULL,
    `hint` varchar(64) NOT NULL
);

CREATE TABLE `purchase` (
    `id` int(11) PRIMARY KEY NOT NULL,
    `uuid` varchar(255) NOT NULL,
    `publication_id` int(11) NOT NULL,
    `user_id` int(11) NOT NULL,
    `license_uuid` varchar(255) NULL,
    `type` varchar(32) NOT NULL,
    `transaction_date` datetime,
    `start_date` datetime,
    `end_date` datetime,
    `status` varchar(255) NOT NULL,
    FOREIGN KEY (`publication_id`) REFERENCES `publication` (`id`),
    FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
);

CREATE INDEX `idx_purchase` ON `purchase` (`license_uuid`)

CREATE TABLE `license_view` (
    `id` int(11) NOT NULL PRIMARY KEY,
    `uuid` varchar(255) NOT NULL,
    `device_count` int(11) NOT NULL,
    `status` varchar(255) NOT NULL,
    `message` varchar(255) NOT NULL
);

