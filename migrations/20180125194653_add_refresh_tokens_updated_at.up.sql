ALTER TABLE `{{ index .Options "Namespace" }}refresh_tokens` ADD `updated_at` timestamp NULL DEFAULT NULL AFTER `created_at`;
