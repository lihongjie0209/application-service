CREATE INDEX application_outbox_retention_idx ON application_outbox_events (published_at, id) WHERE published_at IS NOT NULL;
