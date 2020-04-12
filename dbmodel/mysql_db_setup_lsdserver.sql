CREATE TABLE `license_status` (
    `id` int(11) PRIMARY KEY AUTO_INCREMENT,
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
    `id` int(11) PRIMARY KEY AUTO_INCREMENT,
    `device_name` varchar(255) DEFAULT NULL,
    `timestamp` datetime NOT NULL,
    `type` int NOT NULL,
    `device_id` varchar(255) DEFAULT NULL,
    `license_status_fk` int NOT NULL,
    FOREIGN KEY(`license_status_fk`) REFERENCES `license_status` (`id`)
);

CREATE INDEX `license_status_fk_index` on `event` (`license_status_fk`);