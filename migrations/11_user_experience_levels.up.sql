CREATE TABLE `user_experience_levels` (
  `user_id` varchar(255) NOT NULL,
  `experience_level` varchar(255) CHARACTER SET utf8mb4 NOT NULL,
  `d_update` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`),
  KEY `user_experience_levels_user_id_index` (`user_id`),
  KEY `user_experience_levels_experience_level_index` (`experience_level`),
  KEY `user_experience_levels_d_update_index` (`d_update`)
);