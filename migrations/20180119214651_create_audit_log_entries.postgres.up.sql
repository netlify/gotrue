CREATE TABLE IF NOT EXISTS {{ index .Options "Namespace" }}audit_log_entries (
    instance_id uuid,
    id uuid PRIMARY KEY,
    payload json,
    created_at timestamp with time zone
);

CREATE INDEX audit_logs_instance_id_idx ON {{ index .Options "Namespace" }}audit_log_entries USING btree (instance_id);
