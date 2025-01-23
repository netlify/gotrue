CREATE TABLE IF NOT EXISTS {{ index .Options "Namespace" }}instances (
    id uuid PRIMARY KEY,
    uuid uuid,
    raw_base_config text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);
