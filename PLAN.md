# Peculium — Development Plan

## 1. Vision & Architecture

**Peculium** is a self-hosted, multi-user web app for homelabs to track investments
and personal finances.

### Tech stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Backend** | Go 1.27 (with Chi) | Performance, single binary, minimal container — ideal for a homelab |
| **Frontend** | SvelteKit (Svelte 5) + TypeScript + Vite | Modern SPA, runes API, file-based routing |
| **Database** | PostgreSQL 16 | Relational financial data, CTEs for statistics |
| **Cache** | Redis 7 | Yahoo price cache and API rate limiting |
| **Job queue** | PostgreSQL 16 (`jobs`) | Price / history / splits / metadata sync, progress and health |
| **Container** | Docker/Podman + Compose | Homelab standard |
| **Auth** | JWT + refresh token | Self-hosted, no external dependencies |
| **Charts** | ECharts 5 | ROI, trends, distribution |
| **API design** | RESTful | |

### Why Go?

- Single binary, low RAM/CPU usage (ideal for a homelab)
- Fast builds, simple deployment
- Strong concurrency support for multi-source price fetching

---

## 2. Roadmap (phases)

### Phase 0 — Project setup

- [x] Repository structure (monorepo: Go backend + SvelteKit frontend + Docker)
- [x] Docker Compose with Postgres, Redis, backend, worker, python-service, frontend and secrets-init
- [x] Base CI/CD (GitHub Actions: Go build + vet + test, frontend check + lint) — **EPIC H.1 (#32)**
- [x] Release automation: multi-arch image publishing to GHCR + GitHub Release on `v*` tags — **#131**
- [x] Task runner / Makefile

### Phase 1 — Core: auth & investment management

- [x] Data model: User, Portfolio, Asset, Transaction
- [x] Multi-user registration/login (JWT)
- [x] Portfolio and transaction CRUD (buy / sell / dividend / split / fee)
- [x] Price integration via external APIs (Yahoo Finance)
- [x] Base dashboard: portfolio value, gain/loss
- [x] **EPIC A — data correctness & security** (#3 #4 #5 #6)

### Phase 2 — Statistics & visualisations

- [x] ROI per asset and per portfolio
- [x] Geographic distribution (by asset domicile) and sector distribution (GICS)
- [x] Charts: historical trend, portfolio composition
- [ ] Periodic reports (monthly/quarterly) — **G.3 (#28)**
- [ ] Risk metrics: Sharpe, max drawdown, annualised volatility, benchmark regression (beta/alpha/r²) and Monte Carlo — **EPIC C (#39)**: C.1 (#22), C.2 (#23), C.3 (#24)
- [ ] Investment strategy & rebalancing: target allocation, drift and rebalancing signals — **C.4 (#114)**
- [ ] Transaction CSV import — **G.1 (#26)**
- [ ] Watchlist and price-target alerts — **G.2 (#27)**
- [x] Asset detail page (metadata, price history, geo/sector distributions) — **EPIC B.10 (#45)**
- [x] Python microservice for ETF metadata (JustETF scraping) — **EPIC B.5 (#11)**
- [x] Geographic allocation endpoint (weighted sum by region) — **EPIC B.6 (#12)**
- [x] Sector allocation endpoint (weighted sum by GICS) — **EPIC B.7 (#13)**
- [x] Dashboard/portfolio geography & sector charts (equity-only universe + coverage) — **EPIC B.8 (#14)**
- [x] FX rate history (per-date in the series engine) — **EPIC B.9 (#44)**
- [x] Assets with non-Yahoo tickers: `price_source` (yahoo / manual / none) — **#53**
- [x] Asset price chart: in-place zoom + YTD selector — **EPIC F.9 (#52)**
- [x] Per-country exposure storage + three dimensions (countries / regions / sectors) — **EPIC B.13 (#58)**
- [x] Morningstar exposure source (custom resolver) + backend route + UI prefill — **EPIC B.14 (#59)**
- [ ] Auto-identify the domicile country and GICS sector of an equity as exposure lists on creation — **#202**
- [x] Provider exposure cache (Redis TTL + `?refresh=1`) and persisted provenance (source + date) — **PR #67**
- [x] Stock splits as markers on the asset price chart (`GET /assets/{id}/splits` + markLine)
- [x] Design system & dark mode: semantic tokens, 3-mode theme, `ui/` primitives, responsive AppShell — **EPIC D (#37)**
- [x] Domain pages & components rebuilt (dashboard, portfolio, assets, modals, login, tabbed settings) — **EPIC E (#38)**
- [x] Full Italian/English localisation — **EPIC F**
- [x] Dashboard & portfolio v2: user base currency, active/closed breakdown, time-weighted performance + capital charts, class/sector/country/macro-region allocations, consolidated invested-assets table, paginated transactions — **EPIC I (#83)**
- [x] UX/UI redesign: adaptive shell, tabbed entity pages, command palette, chart table view, CVD palette, allocation drill-down — **EPIC K (PR #115)**, released in v0.6.0
- [x] Admin area: user governance (approval, roles, enable/disable, password reset), server settings, per-user and server-wide backup & restore — **#57, #143, #145**
- [ ] New asset classes: bonds, certificates, supplementary pensions, cash deposits — **EPIC J (#113)**: manual price entry (J.1 #105), fixed-income metadata (J.2 #106), `cash`/`certificate` types (J.3 #107), fixed-income exposure (J.4 #108), deposit interest accrual (J.5 #109), bond metrics (J.6 #110), credit allocation (J.7 #111), pension wrappers (J.8 #112)

### Phase 3 — Multi-tenancy & family sharing

- [x] User roles and admin area — **#57**
- [ ] Portfolio sharing among family members
- [ ] Aggregated family views — **G.5 (#30)**
- [ ] Per-user asset visibility — **#141**

### Phase 4 — Personal finance (future extensions)

- [ ] Expense tracking / categories
- [ ] Monthly budget
- [ ] Savings and goals
- [ ] Unified financial reporting
- [ ] Annual tax report: realized gains/losses and dividends by fiscal year — **G.4 (#29)**

### Phase 5 — Homelab production

- [ ] Reverse proxy (Traefik / Caddy) with TLS
- [ ] Automatic database backup
- [x] Health checks and monitoring
- [x] Deployment documentation (project website and install guide)
- [ ] User manual (EN/IT) published on the website — **#135**
- [ ] In-app contextual help — **#149**
- [ ] PWA / offline — **G.6 (#31)**

### Backlog (open, not tied to a phase)

- [ ] Securities-account model: deposits/withdrawals, available capital and wealth-based performance — **#101**
- [ ] Test coverage extension (backend unit tests + first frontend tests) — **H.2 (#33)**
- [ ] Yahoo 429 rate-limit mitigation (backoff/retry, optional multi-source) — **H.3 (#34)**
- [ ] Portfolios page: full-width cards with portfolio summary info — **#104**
- [ ] Slim the python-service image (Chromium headless-shell) — **#205**
- [ ] Audit log for shared data, admin-only view — **#207**
- [ ] Real per-bar/per-asset backfill progress — **#219**
- [ ] Sector mapping: JustETF labels that do not reach the app sectors (their weight falls outside the classified share) — **#236**
- [ ] ETF exposure partial coverage: accept and flag weights that do not sum to 100 (unmapped JustETF labels, #236) instead of rejecting the whole save (`weights must sum to 100`, WARN in `tests/test-epic-b.sh`)

---

## 3. Data model (draft)

### Core

```
User         → id, email, name, password_hash, role, status, base_currency, created_at
Portfolio    → id, user_id, name, description, currency, created_at
Asset        → id, isin, ticker, name, type, asset_class, price_source, country, exchange, currency, sector, industry
               + maturity_date, issuer, issuer_country, attributes JSONB (fixed income, EPIC J / J.2)
Transaction  → id, portfolio_id, asset_id, type (buy/sell/dividend/split/fee; deposit/withdrawal planned — #101),
               quantity, price, date, fees, notes
Price        → id, asset_id, date, open, high, low, close, volume, source
FxHistory    → base_currency, quote_currency, date, rate, source
Job          → id, type, target_type, target_id, status (queued/running/done/failed/partial), total, processed,
               checkpoint, error, requested_by, created_at, started_at, finished_at
HealthEvent  → id, asset_id, job_id, event_type, status, code, message, duration_ms, created_at
AssetRegionWeight  → asset_id, region, weight (asset_region_weights)
AssetSectorWeight  → asset_id, sector, weight (asset_sector_weights)
AssetCountryWeight → asset_id, country, weight (asset_country_weights, ISO-3166 alpha-2, from B.13)
AssetExposureProvenance → asset_id, dimension, source, updated_at (asset_exposure_provenance: where each
               dimension came from + last update, from B.14)
AssetCredit  → asset_id, rating, weight (asset_credit_weights, planned; post-MVP EPIC J / J.7)
```

### Personal finance (Phase 4)

```
Expense      → id, user_id, category_id, amount, date, description, recurring
Budget       → id, user_id, category_id, amount, period (monthly/yearly)
Goal         → id, user_id, name, target_amount, current_amount, deadline
```

---

## 4. Design principles

1. **Privacy-first**: everything stays in the homelab, no data leaves it
2. **API-first**: every backend feature is reachable through the API
3. **Fully containerised**: `docker compose up` starts everything
4. **Minimal dependencies**: few external libraries, easy to maintain
5. **Offline-resilient**: graceful handling when price sources do not respond
6. **Mobile-friendly**: responsive interface (PWA optional)

---

## 5. Directory structure

```
peculium/
├── docker-compose.yml          # release, pull-only (images from GHCR)
├── docker-compose.dev.yml      # local development (builds from source)
├── docker-compose.test.yml     # isolated e2e stack
├── .env.example                # optional release overrides
├── Makefile
├── backend/
│   ├── cmd/
│   │   ├── server/main.go
│   │   ├── worker/main.go
│   │   └── secrets/main.go   # secrets-init: generates secrets on first boot
│   ├── internal/
│   │   ├── auth/        # JWT, middleware
│   │   ├── cache/       # Redis-backed caching (prices, series, exposure)
│   │   ├── config/      # Configuration from environment
│   │   ├── handler/     # HTTP handlers
│   │   ├── middleware/  # HTTP middleware
│   │   ├── model/       # Structs / entities
│   │   ├── repository/  # DB queries
│   │   ├── secrets/     # Secret generation and loading
│   │   ├── service/     # Business logic
│   │   ├── price/       # Price fetcher (Yahoo, etc.)
│   │   ├── geo/         # Macro-regions, GICS sectors, country/region mappings
│   │   ├── position/    # AVCO engine
│   │   └── series/      # Materialized daily series
│   ├── migrations/      # SQL migrations
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── app.html
│   │   ├── app.css
│   │   ├── lib/
│   │   │   ├── components/   # Svelte 5 components (incl. ui/ primitives)
│   │   │   ├── i18n/         # IT/EN strings and locale store
│   │   │   ├── stores/       # State management (runes)
│   │   │   ├── services/     # API client (fetch + JWT refresh)
│   │   │   ├── format.ts     # Number/date formatters
│   │   │   └── currencies.ts, countryNames.ts, chart*.ts   # shared lookup and chart helpers
│   │   └── routes/           # SvelteKit file-based routing
│   │       ├── +page.svelte  # Dashboard
│   │       ├── login/
│   │       ├── assets/       # list, [id], [id]/data, [id]/exposure
│   │       ├── portfolios/   # list, [id], [id]/positions, [id]/activity, [id]/allocation
│   │       ├── admin/        # users, settings, health, backup
│   │       └── settings/     # preferences, currencies, password, backup
│   ├── svelte.config.js
│   ├── vite.config.ts
│   ├── Dockerfile
│   └── package.json
├── python-service/            # FastAPI: ETF metadata from JustETF/Morningstar (B.5/B.14)
│   ├── app/                   # main.py, scraper.py, morningstar.py, schemas.py
│   ├── tests/                 # pytest
│   ├── Dockerfile
│   └── requirements.txt
├── tests/                     # e2e tests on the isolated stack
│   ├── api-test.http          # REST Client collection (VS Code)
│   ├── seed-prices.sql        # price seed (Yahoo is disabled on that stack)
│   ├── seed-fx-history.sql
│   ├── test-epic-a.sh
│   ├── test-epic-b.sh
│   ├── test-epic-k.sh
│   ├── test-fx-history.sh
│   └── test-portfolio-io.sh
├── scripts/                   # repo utilities (e.g. capture-screenshots.mjs)
├── website/                   # Astro static site (GitHub Pages)
└── docs/
    ├── BACKEND-GUIDE.en.md
    ├── BACKEND-GUIDE.it.md
    ├── DATABASE-GUIDE.en.md
    ├── DATABASE-GUIDE.it.md
    ├── FRONTEND-GUIDE.en.md
    ├── FRONTEND-GUIDE.it.md
    ├── UX-REDESIGN.en.md
    ├── UX-REDESIGN.it.md
    └── RELEASE-NOTES.en.md / RELEASE-NOTES.it.md
```

---

## Current status (18 Oct 2026)

**Release v1.1.0** is the current release: **external data & sync**. Price,
history, splits and metadata work runs through the PostgreSQL-backed job queue,
so the pages return immediately and update as the data arrives; the Data & Sync
page presents those jobs and the provider calls they generate as one activity
list with progress, real durations, filters by type and outcome, a 24h/7d/all
window, a per-job detail panel, an admin clear action and 90-day retention. The
asset page reports the completeness of its price history and the date of its
latest close, the running build (version, commit, build time) is shown in the
interface, and a user backup restore queues the asset sync on commit. It also
carries the user manual (EN/IT) on the website, which publishes with the
release deployment from `main`.

On `main`, **v1.0.0** remains the last published release: the first official
release under the Peculium name, adding the admin area (user governance and
server settings), per-user and server-wide backup & restore, clear login
messages for accounts pending approval or disabled, and — from that release on
— pull-only installation from public GHCR images (linux/amd64 and linux/arm64).

Phase 0, Phase 1 (including EPIC A) and EPIC B are complete and released. The
dashboard/portfolio v2 (EPIC I) shipped in v0.5.0 and the UX/UI redesign
(EPIC K) in v0.6.0.

The next planned work, in order: **J.1 — manual price entry** (#105, the
recommended first PR of **EPIC J — new asset classes** (#113)), **E.10 —
portfolios page full-width cards with summary info** (#104) and **C.1 — risk
metrics endpoint** (#22, the entry point of **EPIC C** (#39)). The rest of
EPIC J (fixed-income metadata (J.2 #106) through pension wrappers (J.8 #112)),
**EPIC G — new features** (#41) and the residual **EPIC H — infrastructure &
quality** (#42) follow as listed in the phases above. See STATUS.md for the
detailed state.
