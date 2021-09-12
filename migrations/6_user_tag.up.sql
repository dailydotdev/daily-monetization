CREATE TABLE `user_tags` (
  `user_id` varchar(255) NOT NULL,
  `tag` varchar(255) CHARACTER SET utf8mb4 NOT NULL,
  `last_read` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`, `tag`),
  KEY `user_tags_user_id_index` (`user_id`),
  KEY `user_tags_tag_index` (`tag`),
  KEY `user_tags_last_read_index` (`last_read`)
);