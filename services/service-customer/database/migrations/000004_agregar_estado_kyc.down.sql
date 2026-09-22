ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_kyc_status_check;
ALTER TABLE customers DROP COLUMN IF EXISTS kyc_status;
