CREATE TABLE outbox_events(sequence bigserial UNIQUE,id uuid PRIMARY KEY,aggregate_id text NOT NULL,event_type text NOT NULL,payload jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),published_at timestamptz);
CREATE INDEX outbox_pending ON outbox_events(sequence) WHERE published_at IS NULL;
CREATE TABLE processed_events(event_id uuid NOT NULL,consumer text NOT NULL,processed_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(event_id,consumer));
CREATE TABLE http_commands(id text PRIMARY KEY,fingerprint text NOT NULL,response jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
