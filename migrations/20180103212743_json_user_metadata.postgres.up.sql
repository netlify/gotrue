UPDATE {{ index .Options "Namespace" }}users SET raw_app_meta_data = '{}' WHERE raw_app_meta_data = '';
UPDATE {{ index .Options "Namespace" }}users SET raw_user_meta_data = '{}' WHERE raw_user_meta_data = '';

ALTER TABLE {{ index .Options "Namespace" }}users
ALTER raw_app_meta_data TYPE JSONB USING raw_app_meta_data::JSONB,
ALTER raw_user_meta_data TYPE JSONB USING raw_user_meta_data::JSONB;
