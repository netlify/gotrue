ALTER TABLE `{{ index .Options "Namespace" }}users` 
CHANGE COLUMN `raw_app_meta_data` `raw_app_meta_data` text DEFAULT NULL ,
CHANGE COLUMN `raw_user_meta_data` `raw_user_meta_data` text DEFAULT NULL ;
