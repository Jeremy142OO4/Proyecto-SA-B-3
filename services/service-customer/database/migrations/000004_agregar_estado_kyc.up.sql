ALTER TABLE customers
    ADD COLUMN kyc_status VARCHAR(20) NOT NULL DEFAULT 'PENDING';

UPDATE customers
SET kyc_status = 'VERIFIED'
WHERE status = 'ACTIVO';

ALTER TABLE customers
    ADD CONSTRAINT customers_kyc_status_check
    CHECK (kyc_status IN ('PENDING', 'VERIFIED', 'REJECTED'));
