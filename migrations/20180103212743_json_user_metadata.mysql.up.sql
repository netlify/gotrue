ALTER TABLE `{{ index .Options "Namespace" }}users` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

UPDATE `{{ index .Options "Namespace" }}users` SET `raw_app_meta_data` = '{}' WHERE `raw_app_meta_data` = '';
UPDATE `{{ index .Options "Namespace" }}users` SET `raw_user_meta_data` = '{}' WHERE `raw_user_meta_data` = '';

ALTER TABLE `{{ index .Options "Namespace" }}users` 
CHANGE COLUMN `raw_app_meta_data` `raw_app_meta_data` JSON NULL DEFAULT NULL ,
CHANGE COLUMN `raw_user_meta_data` `raw_user_meta_data` JSON NULL DEFAULT NULL ;
