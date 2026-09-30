# Peculium — Development Plan

## 1. Vision & Architecture

**Peculium** is a self-hosted, multi-user web app for homelabs to track investments
and personal finances.

### Tech stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Backend** | Go 1.25 (with Chi) | Performance, single binary, minimal container — ideal for a homelab |
| **Frontend** | SvelteKit 2 + Svelte 5 + TypeScript + Vite | Modern SPA, runes API, file-based routing |
| **Database** | PostgreSQL 16 | Relational financial data, CTEs for statistics |
| **Cache/Jobs** | Redis 7 | Yahoo rate limiting, price caching |
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
- [x] Docker Compose with Postgres, Redis, backend, frontend, worker
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
- [ ] Periodic reports (monthly/quarterly)
- [x] Asset detail page (metadata, price history, geo/sector distributions) — **EPIC B.10 (#45)**
- [x] Python microservice for ETF metadata (JustETF scraping) — **EPIC B.5 (#11)**
- [x] Geographic allocation endpoint (weighted sum by region) — **EPIC B.6 (#12)**
- [x] Sector allocation endpoint (weighted sum by GICS) — **EPIC B.7 (#13)**
- [x] Dashboard/portfolio geography & sector charts (equity-only universe + coverage) — **EPIC B.8 (#14)**
- [x] FX rate history (per-date in the series engine) — **EPIC B.9 (#44)**
- [x] Assets with non-Yahoo tickers: `price_source` (yahoo / manual / none) — **EPIC G.7 (#53)**
- [x] Asset price chart: in-place zoom + YTD selector — **EPIC F.9 (#52)**
- [x] Per-country exposure storage + three dimensions (countries / regions / sectors) — **EPIC B.13 (#58)**
- [x] Morningstar exposure source (custom resolver) + backend route + UI prefill — **EPIC B.14 (#59)**
- [x] Provider exposure cache (Redis TTL + `?refresh=1`) and persisted provenance (source + date) — **PR #67**
- [x] Stock splits as markers on the asset price chart (`GET /assets/{id}/splits` + markLine)
- [x] Design system & dark mode: semantic tokens, 3-mode theme, `ui/` primitives, responsive AppShell — **EPIC D (#37)**
- [x] Domain pages & components rebuilt (dashboard, portfolio, assets, modals, login, tabbed settings) — **EPIC E (#38)**
- [x] Full Italian/English localisation — **EPIC F**
- [x] Dashboard & portfolio v2: user base currency, active/closed breakdown, time-weighted performance + capital charts, class/sector/country/macro-region allocations, consolidated invested-assets table, paginated transactions — **EPIC I (#98)**
- [x] UX/UI redesign: adaptive shell, tabbed entity pages, command palette, chart table view, CVD palette, allocation drill-down — **EPIC K (PR #115)**, released in v0.6.0
- [x] Admin area: user governance (approval, roles, enable/disable, password reset), server settings, per-user and server-wide backup & restore — **#57, #143, #145**
- [ ] New asset classes: bonds, certificates, supplementary pensions, cash deposits — **EPIC J (#113)**: manual price entry (J.1 #105), fixed-income metadata (J.2 #106), `cash`/`certificate` types (J.3 #107), fixed-income exposure (J.4 #108), deposit interest accrual (J.5 #109), bond metrics (J.6 #110), credit allocation (J.7 #111), pension wrappers (J.8 #112)

### Phase 3 — Multi-tenancy & family sharing

- [x] User roles and admin area — **#57**
- [ ] Portfolio sharing among family members
- [ ] Aggregated family views
- [ ] Per-user asset visibility — **#141**

### Phase 4 — Personal finance (future extensions)

- [ ] Expense tracking / categories
- [ ] Monthly budget
- [ ] Savings and goals
- [ ] Unified financial reporting

### Phase 5 — Homelab production

- [ ] Reverse proxy (Traefik / Caddy) with TLS
- [ ] Automatic database backup
- [x] Health checks and monitoring
- [x] Deployment documentation (project website and install guide)

---

## 3. Data model (draft)

### Core

```
User         → id, email, name, password_hash, role, status, base_currency, created_at
Portfolio    → id, user_id, name, description, currency, created_at
Asset        → id, isin, ticker, name, type, asset_class, price_source, country, exchange, currency, sector, industry
               + maturity_date, issuer, issuer_country, attributes JSONB (fixed income, EPIC J / J.2)
Transaction  → id, portfolio_id, asset_id, type (buy/sell/dividend/split/fee), quantity, price, date, fees, notes
Price        → id, asset_id, date, open, high, low, close, volume, source
FxHistory    → base_currency, quote_currency, date, rate, source
AssetRegion  → asset_id, region, weight
AssetSector  → asset_id, sector, weight
AssetCountry → asset_id, country, weight (ISO-3166 alpha-2, from B.13)
AssetExposureProvenance → asset_id, dimension, source, updated_at (where each dimension came from + last update, from B.14)
AssetCredit  → asset_id, rating, weight (credit exposure, post-MVP EPIC J / J.7)
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
│   │   ├── handler/     # HTTP handlers
│   │   ├── model/       # Structs / entities
│   │   ├── repository/  # DB queries
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
│   │   │   ├── components/   # Svelte 5 components
│   │   │   ├── stores/       # State management (runes)
│   │   │   ├── services/     # API client (fetch + JWT refresh)
│   │   │   └── format.ts     # Number/date formatters
│   │   └── routes/           # SvelteKit file-based routing
│   │       ├── +page.svelte  # Dashboard
│   │       ├── login/
│   │       ├── assets/
│   │       ├── portfolios/
│   │       ├── admin/
│   │       └── settings/
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
│   ├── test-epic-a.sh
│   ├── test-epic-b.sh
│   ├── test-epic-k.sh
│   └── test-portfolio-io.sh
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

## Current status (30 Sep 2026)

**Release v1.0.0** is published on `main`: the first official release under the
Peculium name, adding the admin area (user governance and server settings),
per-user and server-wide backup & restore, clear login messages for accounts
pending approval or disabled, and — from this release on — pull-only
installation from public GHCR images (linux/amd64 and linux/arm64). It also
brings the Dependabot security updates and moves the backend to Go 1.25.

Phase 0, Phase 1 (including EPIC A) and EPIC B are complete and released. The
dashboard/portfolio v2 (EPIC I) shipped in v0.5.0 and the UX/UI redesign
(EPIC K) in v0.6.0.

Active development proceeds on `develop`. The next planned work is **EPIC J —
new asset classes** (#113): manual price entry (J.1 #105), fixed-income metadata
(J.2 #106) and the `cash`/`certificate` types (J.3 #107) as the recommended
first PR, then fixed-income exposure (J.4 #108), deposit interest accrual
(J.5 #109), bond metrics (J.6 #110), credit allocation (J.7 #111) and pension
wrappers (J.8 #112). The other candidate is **EPIC C — risk metrics**.
Available-cash management (deposits/withdrawals, brokerage account) is tracked
separately in issue #101. See STATUS.md for the detailed state.
