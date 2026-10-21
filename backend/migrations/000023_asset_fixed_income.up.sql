-- UP
-- EPIC J.2/J.3: fixed-income metadata and the `certificate` asset type.
-- Additive columns only, so every existing row keeps its defaults.
ALTER TABLE assets
    ADD COLUMN maturity_date DATE,
    ADD COLUMN issuer TEXT NOT NULL DEFAULT '',
    ADD COLUMN issuer_country TEXT NOT NULL DEFAULT '',
    ADD COLUMN attributes JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Investment certificates (Phoenix, Bonus Cap, Autocallable, ...) join the
-- selectable types; `cash` was already accepted by the constraint.
ALTER TABLE assets DROP CONSTRAINT IF EXISTS assets_type_check;
ALTER TABLE assets ADD CONSTRAINT assets_type_check
    CHECK (type IN ('stock', 'etf', 'bond', 'mutual_fund', 'crypto', 'commodity', 'cash', 'certificate'));

-- Bonds and deposits already carry their geography in `country`: reuse it as
-- the issuer country, which is what feeds the country exposure.
UPDATE assets
   SET issuer_country = country
 WHERE type IN ('bond', 'cash') AND country <> '';

-- Give legacy bonds/deposits without an explicit breakdown a single-country
-- exposure. Existing weights remain untouched.
WITH seeded AS (
    INSERT INTO asset_country_weights (asset_id, country, weight)
    SELECT a.id, a.issuer_country, 100
      FROM assets a
     WHERE a.type IN ('bond', 'cash')
       AND a.issuer_country <> ''
       AND NOT EXISTS (SELECT 1 FROM asset_country_weights w WHERE w.asset_id = a.id)
    RETURNING asset_id
)
INSERT INTO asset_exposure_provenance (asset_id, dimension, source)
SELECT asset_id, 'countries', 'issuer_country' FROM seeded
ON CONFLICT (asset_id, dimension) DO NOTHING;
