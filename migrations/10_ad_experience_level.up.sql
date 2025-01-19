CREATE TABLE IF NOT EXISTS `ad_experience_level` (
  `ad_id` varchar(255) NOT NULL REFERENCES ads(id),
  `experience_level` varchar(255) CHARACTER SET utf8mb4 NOT NULL,
  PRIMARY KEY (`ad_id`, `experience_level`),
  KEY `ad_experience_level_ad_id_index` (`ad_id`)
);