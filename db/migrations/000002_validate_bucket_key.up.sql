ALTER TABLE buckets ADD CONSTRAINT chk_bucket_key CHECK (bucket_key REGEXP '^[a-zA-Z0-9-]+$');
