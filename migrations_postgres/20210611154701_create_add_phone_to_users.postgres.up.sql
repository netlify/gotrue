-- Add phone, phone_confirmed_at columns to auth.users

ALTER TABLE auth.users 
ADD COLUMN phone VARCHAR(15) UNIQUE,
ADD COLUMN phone_confirmed_at timestamptz;