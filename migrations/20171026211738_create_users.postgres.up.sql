CREATE TABLE IF NOT EXISTS {{ index .Options "Namespace" }}users (
    instance_id uuid,
    id uuid PRIMARY KEY,
    aud character varying(255),
    role character varying(255),
    email character varying(255),
    encrypted_password character varying(255),
    confirmed_at timestamp with time zone,
    invited_at timestamp with time zone,
    confirmation_token character varying(255),
    confirmation_sent_at timestamp with time zone,
    recovery_token character varying(255),
    recovery_sent_at timestamp with time zone,
    email_change_token character varying(255),
    email_change character varying(255),
    email_change_sent_at timestamp with time zone,
    last_sign_in_at timestamp with time zone,
    raw_app_meta_data text,
    raw_user_meta_data text,
    is_super_admin boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);
CREATE INDEX users_instance_id_idx ON {{ index .Options "Namespace" }}users USING btree (instance_id);
CREATE INDEX users_instance_id_email_idx ON {{ index .Options "Namespace" }}users USING btree (instance_id, email);
