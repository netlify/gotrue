UPDATE `users` SET `raw_app_meta_data` = '{}' WHERE `raw_app_meta_data` = '';
UPDATE `users` SET `raw_user_meta_data` = '{}' WHERE `raw_user_meta_data` = '';

ALTER TABLE `users` 
CHANGE COLUMN `raw_app_meta_data` `raw_app_meta_data` JSON NULL DEFAULT NULL ,
CHANGE COLUMN `raw_user_meta_data` `raw_user_meta_data` JSON NULL DEFAULT NULL ;
