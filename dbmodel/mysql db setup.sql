
CREATE TABLE `lcp_contents` (
    `id` varchar(255) PRIMARY KEY NOT NULL,
    `encryption_key` varbinary(64) NOT NULL,
    `location` text NOT NULL,
    `length` bigint(20),
    `sha256` varchar(64)
);

CREATE TABLE `lcp_licenses` (
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

ALTER TABLE `lcp_licenses` ADD CONSTRAINT `lcp_licenses_content_fk_lcp_contents_id_foreign` FOREIGN KEY (`content_fk`) REFERENCES `lcp_contents` (`id`);

CREATE TABLE `lsd_license_statuses` (
    `id` int(11) PRIMARY KEY,
    `status` int(11) NOT NULL,
    `license_updated` datetime NOT NULL,
    `status_updated` datetime NOT NULL,
    `device_count` int(11) DEFAULT NULL,
    `potential_rights_end` datetime DEFAULT NULL,
    `license_ref` varchar(255) NOT NULL,
    `rights_end` datetime DEFAULT NULL 
);

CREATE INDEX `license_ref_index` ON `lsd_license_statuses` (`license_ref`);

CREATE TABLE `lsd_events` (
    `id` int(11) PRIMARY KEY,
    `device_name` varchar(255) DEFAULT NULL,
    `timestamp` datetime NOT NULL,
    `type` int NOT NULL,
    `device_id` varchar(255) DEFAULT NULL,
    `license_status_fk` int NOT NULL,
    FOREIGN KEY(`license_status_fk`) REFERENCES `license_status` (`id`)
);

CREATE INDEX `license_status_fk_index` on `lsd_events` (`license_status_fk`);

ALTER TABLE `lsd_events`
ADD CONSTRAINT `lsd_events_license_status_fk_lsd_license_statuses_id_foreign` FOREIGN KEY (`license_status_fk`) REFERENCES `lsd_license_statuses` (`id`);


CREATE TABLE `lut_publications` (
    `id` int(11) NOT NULL PRIMARY KEY,
    `uuid` varchar(255) NOT NULL,	/* == content id */
    `title` varchar(255) NOT NULL,
    `status` varchar(255) NOT NULL
);

CREATE INDEX uuid_index ON lut_publications (`uuid`);

CREATE TABLE `lut_users` (
    `id` int(11) NOT NULL PRIMARY KEY,
    `uuid` varchar(255) NOT NULL,
    `name` varchar(64) NOT NULL,
    `email` varchar(64) NOT NULL,
    `password` varchar(64) NOT NULL,
    `hint` varchar(64) NOT NULL
);

CREATE TABLE `lut_purchase` (
     `id` bigint(20) NOT NULL,
     `publication_id` bigint(20) DEFAULT NULL,
     `user_id` bigint(20) DEFAULT NULL,
     `uuid` varchar(255) NOT NULL,
     `type` varchar(255) DEFAULT NULL,
     `status` varchar(255) DEFAULT NULL,
     `transaction_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
     `license_uuid` varchar(255) DEFAULT NULL,
     `start_date` timestamp NULL DEFAULT NULL,
     `end_date` timestamp NULL DEFAULT NULL,
     `max_end_date` timestamp NULL DEFAULT NULL
    FOREIGN KEY (`publication_id`) REFERENCES `publication` (`id`),
    FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
);

CREATE INDEX `idx_purchase` ON `lut_purchase` (`license_uuid`);

CREATE TABLE `lut_license_views` (
     `id` varchar(255) NOT NULL DEFAULT '',
     `publication_title` varchar(255) DEFAULT NULL,
     `user_name` varchar(255) DEFAULT NULL,
     `type` varchar(255) DEFAULT NULL,
     `uuid` varchar(255) DEFAULT NULL,
     `device_count` bigint(20) DEFAULT NULL,
     `status` varchar(255) DEFAULT NULL,
     `purchase_id` int(11) DEFAULT NULL,
     `message` varchar(255) DEFAULT NULL
);

