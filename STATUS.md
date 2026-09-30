# Peculium — Project Status (30 Sep 2026)

## Infrastructure

| Service | Stack | Notes |
|---------|-------|-------|
| Backend | Go 1.25 + Chi + pgx + golang-migrate | Containerised |
| Frontend | SvelteKit 5 + TypeScript + Tailwind + ECharts | Containerised (nginx) |
| Database | PostgreSQL 16 | With a Docker volume |
| Cache | Redis 7 | Dashboard/series caching, Yahoo rate limiting |
| Worker | Go (prices) | Separate container |
| Python Service | FastAPI + uvicorn + requests + bs4 + selenium (headless Chromium) | ETF metadata from JustETF (country + sector exposure) and Morningstar (countries + official regions + sectors via a custom resolver; market-aware ticker→ISIN auto-resolve) — EPIC B.5, B.14 |
| Container | podman + podman-compose on macOS | |

## Releases

- **v0.1.0** — first official release on `main` (25 Aug 2026).
- **v0.2.0** — EPIC A (data correctness & security) and EPIC B (geographic/sector
  distribution, asset classes, FX history, charts) (30 Aug 2026).
- **v0.3.0** — asset editing overhaul (`price_source` Yahoo/Manual/None, in-place price
  chart with YTD and split markers, exposure modal) and per-country exposure editing with
  Morningstar/JustETF source, cache and provenance (11 Sep 2026).
- **v0.4.0** — design system & dark mode (EPIC D) and rebuilt domain pages/components
  (EPIC E: assets/portfolios/modals/login/tabbed settings); clearer Health page and GitHub
  Actions CI (13 Sep 2026).
- **v0.5.0** — **EPIC I complete** (dashboard & portfolio v2): user base currency with FX
  aggregation, active/closed breakdown, time-weighted performance and capital charts,
  class/sector/country/macro-region allocations, consolidated invested-assets table,
  per-portfolio KPIs aligned with the dashboard, paginated transactions; includes the
  old-export import fix and the fake −100% P/L on closed positions fix (17 Sep 2026).
- **v0.6.0** — **EPIC K complete** (UX/UI redesign): foundations (tokens, self-hosted
  fonts, system theme, IT/EN i18n, primitives), adaptive shell (bottom nav + FAB on
  phones, icon rail on tablets, condensing header), dashboard hero with period chips,
  tabbed portfolio/asset pages, Activity filters with undo and sheet form, ⌘K command
  palette, chart table view, CVD palette and allocation drill-down; includes the backend
  transaction filters and drill-down endpoint (21 Sep 2026).
- **v0.6.1** — bug release: price-refresh status moved to the global header (clickable
  stamp + data-quality strip on every page), full IT/EN translation of the remaining
  screens, and mobile fixes for entity menus, allocation drill-down, chart tooltips and
  the command palette (23 Sep 2026).
- **v1.0.0** — first official release under the **Peculium** name (30 Sep 2026): **admin
  area** (user management — roles, registration approval, enable/disable, password reset —
  and server settings), **backup & restore** per user (JSON, Add or Replace mode) and
  server-wide (full database dump with destructive confirmation), clear login messages for
  accounts pending approval or disabled, and the first release distributed as **pull-only
  GHCR images** (linux/amd64 and arm64); includes the Dependabot security updates
  (golang.org/x/crypto, pgx, go-redis, chi, js-yaml, brace-expansion, devalue) and the move
  of the backend to **Go 1.25**.

Flow: branch → PR to `develop` → merge → tag `vX.Y.Z` on `main`.

## Implemented

### Foundation (Phase 0)

Monorepo (Go backend + SvelteKit frontend + Docker), Docker Compose (Postgres, Redis,
backend, frontend, worker, python-service, secrets-init), REST API (auth, assets,
portfolios, transactions, prices, settings, dashboard, health, admin), DB schema, and a
Makefile for podman-compose.

### Core (Phase 1 + EPIC A)

Multi-user registration/login (JWT access + refresh with rotation), portfolio and asset
CRUD with Yahoo autocomplete and sync, transactions (buy/sell/dividend/split/fee), base
dashboard (value, gain/loss, allocation, performance, per-asset ROI), multi-currency
support with a configurable whitelist, Redis-cached Yahoo prices with rate-limit/backoff
and materialized series, and price-sync health. **EPIC A** added FX-missing-safe
aggregation, ownership enforcement on every analytical endpoint, and data-quality metrics.

### Statistics & exposure (EPIC B, with F/G)

Asset detail page (editable metadata, full price history with backfill, split markers,
in-place zoom + YTD), ETF exposure from **JustETF** and **Morningstar** via the Python
microservice with market-aware ticker→ISIN auto-resolve, three exposure dimensions
(countries/regions/sectors) with persisted provenance and a Redis cache, asset classes
with a class allocation endpoint, geographic/sector/class allocation endpoints at portfolio
and dashboard level (equity-only universe with coverage metadata), per-date FX history in
the series engine, and `price_source` (yahoo/manual/none) for non-Yahoo assets. Full
Italian/English localisation (EPIC F).

### Design system (EPIC D/E)

Semantic tokens, 3-mode theme (light/dark/system) with dark default, self-hosted fonts, a
`ui/` primitive library, a responsive AppShell, and rebuilt domain pages/components
(dashboard, portfolio detail, assets/portfolios with modals, login, tabbed settings).

### Dashboard & portfolio v2 (EPIC I)

User base currency with FX aggregation, active vs closed breakdown, time-weighted return
and capital charts, class/sector/country/macro-region allocations, a consolidated
invested-assets table, aligned per-portfolio KPIs and allocations, and paginated
transaction lists.

### UX/UI redesign (EPIC K)

Adaptive shell (bottom nav + FAB on phones, icon rail on tablets, condensing header),
tabbed portfolio/asset pages with deep-linkable tabs, URL-persisted Activity filters,
transaction form as modal/sheet with undo toast, ⌘K command palette, chart table view,
colour-blind-friendly palette, first-run checklist, freshness stamp and allocation
drill-down.

### Admin & backup (v1.0.0)

Admin area reserved for `owner`/`admin`: user list with role and status, approve pending
registrations, enable/disable accounts, change roles, reset a password, and a server
settings page (auto-approve registrations). On a fresh server the first registered account
becomes the administrator, and account status is checked on every request so disabling a
user invalidates their existing tokens. Per-user backup/restore (JSON, Add or Replace) and
a server-wide database dump/restore with destructive confirmation.

### Infrastructure & quality (EPIC H)

GitHub Actions CI (Go build/vet/test, frontend check/lint, Python pytest, Compose
validation) with branch protection and required checks; multi-arch GHCR publishing and
GitHub Release automation on `v*` tags; GitHub Pages website deployment; price-sync health
with a selectable window (Today / Last 24h / Last 100 events), paginated events and
per-asset `asset_id`; Dependabot security and version updates.

## Open problems

### Yahoo Finance rate-limited (429)

Mitigated with rate limiting/backoff and throttling, but Yahoo can still block the podman
container. From the macOS host it works. If the worker stays blocked, consider an external
macOS script (curl/cron) that POSTs prices to the backend, or an alternative API (Finnhub,
Alpha Vantage with an API key).

### European tickers

European assets on Yahoo use an exchange suffix (e.g. `VWCE.DE`, `VWCE.AS`). The user must
know the correct ticker; this should be documented or the autocomplete should offer an
exchange selector.

### Health events coverage and retention

Health events do not yet cover every external call (meta/profile, JustETF/Morningstar,
successes) and there is no retention policy for the `health_events` table.

## Not yet implemented

- **EPIC J — new asset classes** (#113): manual price entry (J.1 #105), fixed-income
  metadata (J.2 #106), `cash`/`certificate` types (J.3 #107), fixed-income exposure
  (J.4 #108), deposit interest accrual (J.5 #109), bond metrics (J.6 #110), credit
  allocation (J.7 #111), pension wrappers (J.8 #112).
- **EPIC C — risk metrics** (#39): Sharpe, max drawdown, volatility, regression, Monte Carlo.
- Portfolio/cash management (#101): available capital, deposits/withdrawals, brokerage account.
- Portfolio sharing among family members, per-user asset visibility (#141) and aggregated
  family views (Phase 3).
- Expense tracking, budget and goals (Phase 4).
- Periodic (monthly/quarterly) reports.

## Useful commands

```bash
make up              # Start everything with podman-compose (dev stack)
make down            # Stop all services
make reset           # Stop services and delete data volumes (fresh start)
make logs            # Follow logs
make migrate         # Run DB migrations
make test            # Go tests (in container)
make test-e2e        # End-to-end tests on the isolated stack
make frontend-dev    # Frontend development with hot reload
# Manual API testing (VS Code REST Client) on the test stack:
#   tests/api-test.http — requests in order against http://localhost:8081/api/v1
# EPIC B allocation smoke test (test stack, port 8081):
#   tests/test-epic-b.sh [--step | --no-seed]
# EPIC K filters/drill-down smoke test (test stack, port 8081):
#   tests/test-epic-k.sh
# Recreate a container after changes (dev stack):
podman-compose -f docker-compose.dev.yml stop <service>
podman rm <container>
podman-compose -f docker-compose.dev.yml up -d --build <service>
```
