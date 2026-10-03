-- Local demo role. Production uses Secret Manager and a non-default password.
DO $$
BEGIN
  CREATE ROLE finguard_app LOGIN PASSWORD 'finguard_app';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

GRANT CONNECT ON DATABASE finguard TO finguard_app;
