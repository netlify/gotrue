CREATE TABLE `refresh_tokens` (
  `instance_id` varchar(255) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `token` varchar(255) DEFAULT NULL,
  `user_id` varchar(255) DEFAULT NULL,
  `revoked` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE INDEX refresh_tokens_instance_id_idx ON refresh_tokens (instance_id);
CREATE INDEX refresh_tokens_instance_id_user_id_idx ON refresh_tokens (instance_id, user_id);
CREATE INDEX refresh_tokens_token_idx ON refresh_tokens (token);
