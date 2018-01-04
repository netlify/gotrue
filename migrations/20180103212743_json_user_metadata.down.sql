ALTER TABLE `users` 
CHANGE COLUMN `raw_app_meta_data` `raw_app_meta_data` varchar(255) DEFAULT NULL ,
CHANGE COLUMN `raw_user_meta_data` `raw_user_meta_data` varchar(255) DEFAULT NULL ;
