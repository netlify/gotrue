CREATE TABLE IF NOT EXISTS `instances` (
  `id` varchar(255) NOT NULL,
  `uuid` varchar(255) DEFAULT NULL,
  `raw_base_config` longtext,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
