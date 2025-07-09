-- 以上表位于 SqlDatabases=File 中
CREATE TABLE IF NOT EXISTS `files` (
    `hash` VARCHAR(256) NOT NULL PRIMARY KEY,
    `blocks` LONGBLOB NOT NULL,
    `filesize` INT NOT NULL,
    `upload_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `upload_uid` INT(32) NOT NULL,
    `delete` BOOLEAN NOT NULL DEFAULT FALSE
);