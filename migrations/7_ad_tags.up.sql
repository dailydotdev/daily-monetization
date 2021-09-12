CREATE TABLE `ad_tags` (
  `ad_id` varchar(255) NOT NULL REFERENCES ads(id),
  `tag` varchar(255) CHARACTER SET utf8mb4 NOT NULL,
  PRIMARY KEY (`ad_id`, `tag`),
  KEY `ad_tags_ad_id_index` (`ad_id`)
);