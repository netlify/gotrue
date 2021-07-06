-- Add phone, phone_confirmed_at columns to auth.users

ALTER TABLE auth.users 
DROP COLUMN phone,
DROP COLUMN phone_confirmed_at;