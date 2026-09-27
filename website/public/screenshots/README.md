# Screenshots

Feature-area screenshots of Peculium, captured from the **isolated test stack**
(`docker-compose.test.yml` — never the dev/prod one) at **2× DPR of a
1280×800 viewport** (2560×1600 px). The pages reference them through
`withBase('/screenshots/<locale>/<file>')`, laid out at 1280×800.

One directory per locale — each page language must use the matching directory:

- `en/` — screenshots with the English UI (used by `/features`)
- `it/` — screenshots with the Italian UI (used by `/it/features`)

Both directories carry the same six filenames, and that pairing is the
contract with the pages:

- `portfolio-dashboard.png` — Portfolio & dashboard
- `allocations.png` — Allocations
- `transactions.png` — Transactions & history
- `asset-exposure.png` — Asset detail & exposure
- `price-health.png` — Data & sync / price health
- `multi-currency.png` — Multi-currency

To refresh them after a UI change, re-capture the matching area from the test
stack at the same viewport/DPR — in the **right locale** (switch the app's
language when capturing `it/`) — and replace the file in place.
