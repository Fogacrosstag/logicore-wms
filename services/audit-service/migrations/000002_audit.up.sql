CREATE TABLE audit_events(id uuid PRIMARY KEY,event_id uuid NOT NULL UNIQUE,event_type text NOT NULL,aggregate_type text NOT NULL,aggregate_id text NOT NULL,payload jsonb NOT NULL,timestamp timestamptz NOT NULL,correlation_id text NOT NULL,source text NOT NULL);
CREATE INDEX audit_warehouse ON audit_events((payload->>'warehouseId'),timestamp);
CREATE INDEX audit_product ON audit_events((payload->>'productId'),timestamp);
CREATE INDEX audit_payload ON audit_events USING gin(payload);
CREATE INDEX audit_timestamp ON audit_events(timestamp,id);
