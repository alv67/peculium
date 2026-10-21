-- DOWN
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM assets WHERE type = 'certificate') THEN
        RAISE EXCEPTION 'cannot roll back asset metadata migration while certificate assets exist';
    END IF;
END $$;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS assets_type_check;
ALTER TABLE assets ADD CONSTRAINT assets_type_check
    CHECK (type IN ('stock', 'etf', 'bond', 'mutual_fund', 'crypto', 'commodity', 'cash'));
ALTER TABLE assets
    DROP COLUMN attributes,
    DROP COLUMN issuer_country,
    DROP COLUMN issuer,
    DROP COLUMN maturity_date;
