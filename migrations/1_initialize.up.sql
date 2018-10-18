CREATE TABLE `ads` (
  `id` varchar(255) NOT NULL,
  `title` varchar(255) CHARACTER SET utf8mb4 NOT NULL,
  `url` text NOT NULL,
  `image` text DEFAULT NULL,
  `ratio` float(8,2) DEFAULT NULL,
  `placeholder` text,
  `source` varchar(255) DEFAULT NULL,
  `start` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `end` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `ads_start_end_index` (`start`,`end`)
);
