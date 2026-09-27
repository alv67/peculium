# Screenshots

Slot for the feature-area screenshots (website issue #102, slice 3). Files are
captured from the **isolated test stack** (`make test-e2e` stack — never the
dev/prod one) and dropped here with exactly these names, which the Features
pages already reference through `withBase('/screenshots/<file>')`:

- `portfolio-dashboard.png`
- `allocations.png`
- `transactions.png`
- `asset-exposure.png`
- `price-health.png`
- `multi-currency.png`

Recommended size: 1280×800 (16:10). Until a file lands, the page renders the
"screenshot coming soon" placeholder.
