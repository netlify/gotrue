-- auth.audit_log_entries definition

CREATE TABLE IF NOT EXISTS auth.audit_log_entries (
	instance_id uuid NULL,
	id uuid NOT NULL,
	payload json NULL,
	created_at timestamptz NULL,
	CONSTRAINT audit_log_entries_pkey PRIMARY KEY (id)
);
CREATE INDEX audit_logs_instance_id_idx ON auth.audit_log_entries USING btree (instance_id);
comment on table auth.audit_log_entries is 'Auth: Audit trail for user actions.';