CREATE TABLE IF NOT EXISTS `{{ index .Options "Namespace" }}audit_log_entries` (
  `instance_id` varchar(255) DEFAULT NULL,
  `id` varchar(255) NOT NULL,
  `payload` JSON NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `audit_logs_instance_id_idx` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
