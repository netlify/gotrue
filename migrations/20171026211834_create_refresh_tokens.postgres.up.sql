CREATE TABLE IF NOT EXISTS {{ index .Options "Namespace" }}refresh_tokens (
    instance_id uuid,
    id bigserial PRIMARY KEY,
    token character varying(255),
    user_id character varying(255),
    revoked boolean,
    created_at timestamp with time zone
);

CREATE INDEX refresh_tokens_instance_id_idx ON {{ index .Options "Namespace" }}refresh_tokens USING btree (instance_id);
CREATE INDEX refresh_tokens_instance_id_user_id_idx ON {{ index .Options "Namespace" }}refresh_tokens USING btree (instance_id, user_id);
CREATE INDEX refresh_tokens_token_idx ON {{ index .Options "Namespace" }}refresh_tokens USING btree (token);
