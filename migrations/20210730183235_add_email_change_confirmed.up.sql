-- adds email_change_confirmed

ALTER TABLE auth.users
ADD COLUMN IF NOT EXISTS email_change_token_current varchar(255) null DEFAULT '', 
ADD COLUMN IF NOT EXISTS email_change_confirm_status smallint DEFAULT 0 CHECK (email_change_confirm_status >= 0 AND email_change_confirm_status <= 2);

DO $$
BEGIN
  IF NOT EXISTS(SELECT *
    FROM information_schema.columns
    WHERE table_schema = 'auth' and table_name='users' and column_name='email_change_token_new')
  THEN
      ALTER TABLE "auth"."users" RENAME COLUMN "email_change_token" TO "email_change_token_new";
  END IF;
END $$;
