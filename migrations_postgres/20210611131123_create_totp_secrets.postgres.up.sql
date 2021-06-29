-- auth.totp_secrets definition

CREATE TABLE IF NOT EXISTS auth.totp_auth(
    instance_id uuid NULL,
    id bigserial NOT NULL,
    user_id uuid NOT NULL UNIQUE,
    encrypted_url bytea NULL,
    otp_last_requested_at timestamptz NULL,
    created_at timestamptz NULL,
	updated_at timestamptz NULL,
    CONSTRAINT totp_secrets_pkey PRIMARY KEY (id)
);
CREATE INDEX totp_auth_instance_id_idx ON auth.totp_auth USING btree (instance_id);
CREATE INDEX totp_auth_instance_id_user_id_idx ON auth.totp_auth USING btree (instance_id, user_id);
comment on table auth.totp_auth is 'Auth: Store of totp secrets used to generate otp once they expire.';