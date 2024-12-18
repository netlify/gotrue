ALTER TABLE {{ index .Options "Namespace" }}instances ADD deleted_at timestamp with time zone;
