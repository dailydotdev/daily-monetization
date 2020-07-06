CREATE TABLE `segments` (
  `user_id` varchar(255) NOT NULL,
  `segment` varchar(255) NOT NULL,
  PRIMARY KEY (`user_id`),
  KEY `segment_index` (`segment`)
);
