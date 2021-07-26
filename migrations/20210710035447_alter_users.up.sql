-- alter user schema

ALTER TABLE auth.users 
ADD COLUMN IF NOT EXISTS phone VARCHAR(15) NULL UNIQUE DEFAULT NULL,
ADD COLUMN IF NOT EXISTS phone_confirmed_at timestamptz NULL DEFAULT NULL,
ADD COLUMN IF NOT EXISTS phone_change VARCHAR(15) NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS phone_change_token VARCHAR(255) NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS phone_change_sent_at timestamptz NULL DEFAULT NULL;

ALTER TABLE auth.users
RENAME COLUMN confirmed_at TO email_confirmed_at;
