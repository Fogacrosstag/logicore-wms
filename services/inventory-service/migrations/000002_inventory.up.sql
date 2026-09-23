CREATE TABLE stocks(
 id uuid PRIMARY KEY,product_id uuid NOT NULL,warehouse_id uuid NOT NULL,location_id uuid,quantity bigint NOT NULL CHECK(quantity>=0),reserved_quantity bigint NOT NULL DEFAULT 0 CHECK(reserved_quantity>=0 AND reserved_quantity<=quantity),
 batch_number text NOT NULL,expiration_date date,unit_weight float8 NOT NULL CHECK(unit_weight>0),unit_volume float8 NOT NULL CHECK(unit_volume>0),storage_type text NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX stocks_allocation ON stocks(warehouse_id,product_id,expiration_date) WHERE quantity>0;
CREATE TABLE capacities(location_id uuid PRIMARY KEY,warehouse_id uuid NOT NULL,current_weight float8 NOT NULL DEFAULT 0 CHECK(current_weight>=0),current_volume float8 NOT NULL DEFAULT 0 CHECK(current_volume>=0),revision bigint NOT NULL DEFAULT 0);
CREATE TABLE reservations(id uuid PRIMARY KEY,warehouse_id uuid NOT NULL,status text NOT NULL,expires_at timestamptz NOT NULL,shipment_id uuid,created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX reservation_expiry ON reservations(expires_at) WHERE status='CONFIRMED';
CREATE TABLE allocations(reservation_id uuid REFERENCES reservations(id),stock_id uuid REFERENCES stocks(id),quantity bigint NOT NULL CHECK(quantity>0),PRIMARY KEY(reservation_id,stock_id));
CREATE VIEW stock_view AS SELECT id,product_id AS "productId",warehouse_id AS "warehouseId",location_id AS "locationId",quantity,reserved_quantity AS "reservedQuantity",quantity-reserved_quantity AS "availableQuantity",batch_number AS "batchNumber",expiration_date AS "expirationDate",unit_weight AS "unitWeight",unit_volume AS "unitVolume",storage_type AS "storageType",created_at AS "createdAt",updated_at AS "updatedAt" FROM stocks;
