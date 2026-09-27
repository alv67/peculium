# Screenshots

Feature-area screenshots of Peculium, captured from the **isolated test stack**
(`docker-compose.test.yml` — never the dev/prod one) at **2× DPR of a
1280×800 viewport** (2560×1600 px). The Features pages reference them through
`withBase('/screenshots/<file>')`, laid out at 1280×800:

- `portfolio-dashboard.png` — Portfolio & dashboard
- `allocations.png` — Allocations
- `transactions.png` — Transactions & history
- `asset-exposure.png` — Asset detail & exposure
- `price-health.png` — Data & sync / price health
- `multi-currency.png` — Multi-currency

To refresh them after a UI change, re-capture the matching area from the test
stack at the same viewport/DPR and replace the file in place — the names are
the contract with the pages.
