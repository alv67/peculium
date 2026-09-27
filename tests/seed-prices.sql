-- Peculium — manual price seed for the ISOLATED test stack only
-- (project peculium-test, DB peculium_test, port 8081, Yahoo finance disabled).
--
-- The test stack exposes no HTTP writer for prices (the only writer,
-- Price.Create, is used by the Yahoo fetcher which is disabled here), so the
-- portfolio allocation weights stay 0 until prices are seeded directly in the DB.
--
-- Run it against the test stack ONLY, NEVER against dev/prod (real data):
--
--   docker compose -p peculium-test -f docker-compose.test.yml exec -T postgres psql -U peculium -d peculium_test < tests/seed-prices.sql
--
-- Idempotent: re-running overwrites the same (asset_id, date) rows via
-- ON CONFLICT DO UPDATE, so reruns are safe.

INSERT INTO prices (asset_id, date, open, high, low, close, volume, source)
SELECT id, CURRENT_DATE, 105.00, 105.00, 105.00, 105.00, 0, 'manual'
FROM assets WHERE ticker = 'SMEA.MI'
ON CONFLICT (asset_id, date) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    source = EXCLUDED.source;

INSERT INTO prices (asset_id, date, open, high, low, close, volume, source)
SELECT id, CURRENT_DATE, 310.00, 310.00, 310.00, 310.00, 0, 'manual'
FROM assets WHERE ticker = 'SXR8.DE'
ON CONFLICT (asset_id, date) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    source = EXCLUDED.source;