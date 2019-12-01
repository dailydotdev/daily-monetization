ALTER TABLE `ads`
    ADD COLUMN `probability` FLOAT(8,2) DEFAULT 0.0,
    ADD COLUMN `fallback` TINYINT DEFAULT 0,
    ADD COLUMN `company` varchar(255) CHARACTER SET utf8mb4;
