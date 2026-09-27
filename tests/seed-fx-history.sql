-- Peculium — FX history seed for the ISOLATED test stack only
-- (project peculium-test, DB peculium_test, port 8081, Yahoo finance disabled).
--
-- Used by tests/test-fx-history.sh. It plants a stable EUR price for the
-- VLABEUR1 smoke asset on two dates and a USD->EUR FX history that moves
-- materially between those dates (0.90 on 2024-01-15, 1.10 on 2024-03-15), so
-- the portfolio history must show the resulting per-date FX distortion.
--
-- Run it against the test stack ONLY, NEVER against dev/prod (real data):
--
--   docker compose -p peculium-test -f docker-compose.test.yml exec -T postgres psql -U peculium -d peculium_test < tests/seed-fx-history.sql
--
-- Idempotent: ON CONFLICT DO UPDATE makes reruns safe.

INSERT INTO prices (asset_id, date, open, high, low, close, volume, source)
SELECT id, DATE '2024-01-15', 100.00, 100.00, 100.00, 100.00, 0, 'manual'
FROM assets WHERE ticker = 'VLABEUR1'
ON CONFLICT (asset_id, date) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    source = EXCLUDED.source;

INSERT INTO prices (asset_id, date, open, high, low, close, volume, source)
SELECT id, DATE '2024-03-15', 100.00, 100.00, 100.00, 100.00, 0, 'manual'
FROM assets WHERE ticker = 'VLABEUR1'
ON CONFLICT (asset_id, date) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    source = EXCLUDED.source;

INSERT INTO fx_history (base_currency, quote_currency, date, rate, source)
VALUES ('USD', 'EUR', DATE '2024-01-15', 0.90, 'manual')
ON CONFLICT (base_currency, quote_currency, date) DO UPDATE SET
    rate = EXCLUDED.rate,
    source = EXCLUDED.source;

INSERT INTO fx_history (base_currency, quote_currency, date, rate, source)
VALUES ('USD', 'EUR', DATE '2024-03-15', 1.10, 'manual')
ON CONFLICT (base_currency, quote_currency, date) DO UPDATE SET
    rate = EXCLUDED.rate,
    source = EXCLUDED.source;