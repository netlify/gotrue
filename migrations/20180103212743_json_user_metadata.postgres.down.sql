ALTER TABLE {{ index .Options "Namespace" }}users
ALTER raw_app_meta_data TYPE text USING raw_app_meta_data #>> '{}',
ALTER raw_user_meta_data TYPE text USING raw_user_meta_data #>> '{}';
