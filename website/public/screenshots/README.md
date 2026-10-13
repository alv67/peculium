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

## How to capture (reproducible)

Use `scripts/capture-screenshots.mjs`. It drives **headless Chrome over the
DevTools Protocol**, the only way to pin the exact spec (a 1280×800 viewport at
2× DPR, light theme); it logs in through the API and injects the tokens into
`localStorage`, so no manual UI login is needed.

Prerequisites — all on the **isolated test stack**, never the dev/prod one:

1. Test backend on `:8081` with a seeded user named **Demo Investor**
   (register/login via `POST /api/v1/auth/register` + `/login`, see
   `tests/api-test.http`) — the sidebar shows the account name, so every
   screenshot must use the same one. Start the stack with
   `podman-compose -p peculium-test -f docker-compose.test.yml up -d --build backend`.
   The test compose has no worker: to have activity rows on a page, run a worker
   image on the same network by hand.
2. Frontend dev server whose `/api` proxy points at `:8081`: temporarily set
   `server.proxy['/api'].target` to `http://localhost:8081` in
   `frontend/vite.config.ts`, then `cd frontend && npm run dev -- --host`.
   **Revert the proxy afterwards.**
3. Headless Chrome with a debugging port:

   ```bash
   "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
     --headless=new --disable-gpu --hide-scrollbars --no-first-run \
     --remote-debugging-port=9222 --user-data-dir=/tmp/pec-chrome about:blank &
   ```

Then one command per file (credentials/URLs are overridable via
`PECULIUM_SHOT_*` env vars, see the script header):

```bash
node scripts/capture-screenshots.mjs website/public/screenshots/en/price-health.png en /admin/health
node scripts/capture-screenshots.mjs website/public/screenshots/it/price-health.png it /admin/health
```

Always capture **both locales** and replace the files in place. The manual
screenshots under `manual/` are captured the same way (different route/state).
