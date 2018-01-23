ALTER TABLE `{{ index .Options "Namespace" }}instances` ADD `deleted_at` timestamp NULL DEFAULT NULL AFTER `updated_at`;
