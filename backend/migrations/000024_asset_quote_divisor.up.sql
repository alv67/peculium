-- UP
-- EPIC J.4 (option A1): asset-level quote divisor for instruments quoted per
-- 100 (or 1000) nominal, e.g. bonds. Cash amounts become
-- quantity × price / quote_divisor. Additive: the default 1 leaves every
-- existing asset at its current behaviour.
ALTER TABLE assets
    ADD COLUMN quote_divisor NUMERIC(12,4) NOT NULL DEFAULT 1 CHECK (quote_divisor > 0);
