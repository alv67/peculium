# Peculium

**Your wealth, self-hosted.**

Self-hosted, multi-user personal finance and investment suite for homelabs.

Track your investments, monitor asset performance, and gain insights into your financial portfolio — all from your own infrastructure.

## Features

- **Multi-user** — Family-friendly with role-based access (owner, admin, editor, viewer)
- **Portfolio management** — Multiple portfolios per user, export/import, ownership enforced on all endpoints
- **Asset tracking** — Stocks, ETFs, bonds, crypto, commodities with auto-complete and sync via Yahoo Finance
- **Asset detail page** — Editable metadata (exchange, ISIN), full price history with backfill, and editable geographic/sector exposure with charts
- **ETF exposure from JustETF and Morningstar** — a Python microservice resolves the ISIN from a ticker and fetches full country/region + GICS sector weights from JustETF (`POST /assets/{id}/fetch-etf-exposure`) and official regions from Morningstar (`POST /assets/{id}/fetch-morningstar-exposure`); each distribution keeps its source and last-update date
- **Transaction history** — Buy, sell, dividends, splits, fees with multi-currency support
- **Dashboard** — Portfolio value, gain/loss, allocation, performance charts, ROI by asset
- **Market prices** — Yahoo Finance with Redis caching, rate-limit/backoff, series materialization, price health dashboard
- **Data quality** — Summary exposes staleness / missing-country / missing-sector / missing-FX metrics
- **Self-contained** — Everything runs via Docker/Podman Compose

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.23, Chi router, pgx (PostgreSQL), golang-migrate |
| Frontend | SvelteKit 5, TypeScript, Tailwind CSS, ECharts |
| ETF metadata | Python microservice (FastAPI, requests, BeautifulSoup) — JustETF scraping |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Container | Docker / Podman + Compose |
| Auth | JWT (access + refresh tokens, rotation) |

## Quick Start

### From source (local build)

```bash
make up
```

Then open http://localhost:3000. This builds the images from the working tree
(`docker-compose.dev.yml`).

### Pull-only install (no local build)

The release stack (`docker-compose.yml`) runs pre-built images published to
GitHub Container Registry (`ghcr.io/alv67/peculium-*`, available from the
v1.0.0 release onward):

```bash
curl -O https://raw.githubusercontent.com/alv67/peculium/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/alv67/peculium/main/.env.example
cp .env.example .env   # optional: pin a version or provide your own secrets
podman-compose up -d
```

Secrets are generated on first boot and stored in the `peculium_secrets` volume
(back it up). Pin `PECULIUM_VERSION` in `.env` to keep the stack on a fixed
release.

> Requires Podman (or Docker) with Compose support.

## Testing

```bash
make test-e2e        # End-to-end API tests on an isolated stack (no data pollution)
make test            # Go unit tests (inside the backend container)
cd python-service && python3 -m pytest   # ETF microservice tests
```

Manual API testing against the isolated test stack (port 8081): open
[`tests/api-test.http`](tests/api-test.http) with the VS Code **REST Client** extension
and send the requests in order.

## Project Status

Releases on [`main`](https://github.com/alv67/peculium/tree/main): **v0.1.0** (25 Aug 2026,
first official release), **v0.2.0** (30 Aug 2026, EPIC A — data correctness & security —
and EPIC B — geographic/sector distribution, asset classes, FX history, charts) and
**v0.3.0** (11 Sep 2026, asset editing overhaul and per-country exposure editing with
Morningstar/JustETF prefill).

Active development on the [`develop`](https://github.com/alv67/peculium/tree/develop) branch — see [STATUS.md](STATUS.md) and [PLAN.md](PLAN.md) for the roadmap.

## License

Released under the [MIT License](LICENSE). See [`licenses/`](licenses/) for the
third-party components and their licenses.
