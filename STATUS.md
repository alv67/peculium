# Peculium — Project Status (20 Oct 2026)

## Infrastructure

| Service | Stack | Notes |
|---------|-------|-------|
| Backend | Go 1.27 + Chi + pgx + golang-migrate | Containerised |
| Frontend | SvelteKit (Svelte 5) + TypeScript + Tailwind + ECharts | Containerised (nginx) |
| Database | PostgreSQL 16 | With a Docker volume |
| Cache | Redis 7 | Dashboard/series caching, Yahoo rate limiting |
| Job queue | PostgreSQL 16 (`jobs`) | One row per sync job (price, history, splits, metadata) with progress and checkpoint, claimed by the worker |
| Worker | Go | Separate container; runs the jobs queued in PostgreSQL |
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
- **v1.1.0** — **external data & sync** (18 Oct 2026): price, history, splits and metadata
  work runs through a PostgreSQL-backed job queue in the background, so the pages return
  immediately and update as the data arrives; the **Data & Sync** page merges jobs and
  provider calls into one activity list (progress, real durations, filters by type and
  outcome, 24h/7d/all window, per-job detail panel, admin clear, 90-day retention); the
  asset page reports its price-history completeness and the date of its latest close; and
  the running build (version, commit, build time) is shown in the sidebar, on the login
  page and in Settings → Preferences.

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

GitHub Actions CI (Go 1.27 build/vet/test, frontend check/lint, Python pytest, Compose
validation) with branch protection and required checks; multi-arch GHCR publishing and
GitHub Release automation on `v*` tags, plus a rolling `develop` tag published on manual
dispatch and a retention step at the end of every publish pruning old test tags (releases,
`latest` and `develop` are never deleted); GitHub Pages website deployment; Dependabot security
and version updates.

### External data & sync

Price, history, splits and metadata work runs through a PostgreSQL-backed job queue: each job
carries its type, target, status (queued / running / done / failed / partial), progress and a
checkpoint, and is claimed by the worker. On the **Data & Sync** page the jobs and the external
calls they produce form a single activity list, newest first, with a real duration and the error
when there is one and refreshed while something runs; the list filters by call type and outcome,
and a selectable window (24h / 7d / all) scopes the summary tiles together with the list.
Opening a job shows its own log lines in a side panel, an admin can clear the whole activity
with one confirmed action, and activity older than 90 days is removed automatically. Every
external call is covered — prices, FX, splits, asset metadata and the ETF/Morningstar exposure
lookups, with the ticker beside the ISIN — and the pages read from the cache and update as the
queued work completes, so a slow source no longer blocks the interface. A user backup restore
also queues the asset sync when it commits, so restored assets get their market data right
away.

The asset page reports the completeness of its price history (still downloading, absent, or
partial, each with the action to complete it) and labels the date of its latest close, which the
backend resolves from a single source across installations. The running build (version, commit
and build time) is visible in the sidebar, on the login page and in Settings → Preferences
("About this app").

### Website

Astro static site on GitHub Pages (homepage, features, install guide, release notes), deployed
on every `v*` tag together with the GHCR images and the GitHub Release. It carries the **user
manual** in English and Italian (quickstart, concepts, daily use, dashboard, data & sync,
settings, backup, administration, FAQ) under `/manual` and `/it/manual`, linked from the site
menu (#135), and ships with v1.1.0.

## Open problems

### Yahoo Finance rate-limited (429)

Mitigated with rate limiting/backoff and throttling, and the sync work now runs in the
background queue, so a blocked worker no longer blocks the interface. Yahoo can still block the
podman container (from the macOS host it works) and the remaining mitigation is tracked in #34.
If the worker stays blocked, consider an external macOS script (curl/cron) that POSTs prices to
the backend, or an alternative API (Finnhub, Alpha Vantage with an API key).

### JustETF legacy sector taxonomy (#250)

JustETF serves a legacy Morningstar sector taxonomy for many ETFs. The labels
are now all mapped into the canonical sector list (with an "Other / Not
Classified" residual bucket), but the feed's classification vintage still
distorts the weights: pre-2018 holdings such as Alphabet/Meta count as
Technology instead of Communication Services, and REITs are merged into
Financials. The upgrade path (prefer the Morningstar source for sectors when
available) is tracked in #250.

### European tickers

European assets on Yahoo use an exchange suffix (e.g. `VWCE.DE`, `VWCE.AS`). The user must
know the correct ticker; this should be documented or the autocomplete should offer an
exchange selector.

## Not yet implemented

- **EPIC J — new asset classes** (#113): manual price entry (J.1 #105), fixed-income
  metadata (J.2 #106), `cash`/`certificate` types (J.3 #107), fixed-income exposure
  (J.4 #108), deposit interest accrual (J.5 #109), bond metrics (J.6 #110), credit
  allocation (J.7 #111), pension wrappers (J.8 #112).
- **EPIC C — risk metrics** (#39): Sharpe, max drawdown and volatility (C.1 #22), benchmark
  regression (C.2 #23), Monte Carlo (C.3 #24) and investment strategy & rebalancing (C.4 #114).
- Securities-account model (#101): available capital, deposits/withdrawals and wealth-based
  performance.
- Portfolio sharing among family members, per-user asset visibility (#141) and aggregated
  family views (G.5 #30).
- Transaction CSV import (G.1 #26), watchlist and price-target alerts (G.2 #27), periodic
  reports (G.3 #28) and the annual tax report (G.4 #29).
- Domicile country and GICS sector auto-identified from the exposure lists on creation (#202).
- PWA / offline (G.6 #31) and in-app contextual help (#149).
- Audit log for shared data, admin-only view (#207).
- Expense tracking, budget and goals (Phase 4).
- Test coverage extension (H.2 #33), Yahoo 429 mitigation (H.3 #34), slimmer python-service
  image (#205), portfolios page summary cards (#104) and real per-bar backfill progress (#219).

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
make publish         # Dispatch a GHCR image build from a branch (RC_REF=develop, RC_VERSION=extra tag)
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
