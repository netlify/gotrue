CREATE TABLE IF NOT EXISTS `{{ index .Options "Namespace" }}refresh_tokens` (
  `instance_id` varchar(255) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `token` varchar(255) DEFAULT NULL,
  `user_id` varchar(255) DEFAULT NULL,
  `revoked` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `refresh_tokens_instance_id_idx` (`instance_id`),
  KEY `refresh_tokens_instance_id_user_id_idx` (`instance_id`,`user_id`),
  KEY `refresh_tokens_token_idx` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
