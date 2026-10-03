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
| Backend | Go 1.25, Chi router, pgx (PostgreSQL), golang-migrate |
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

Published image tags:

- `latest` — the most recent release (from a `v*` tag on `main`);
- the release tag (`v1.0.0`, …);
- `develop` — a rolling tag for the latest build dispatched from the
  `develop` branch (unstable, moved on every dispatch);
- `rc-<date>` — a named test/RC build (via the dispatch `version` input);
- `sha-<commit>` — the exact commit of any build, always published for pinning.

None of the test tags (`develop`, `rc-*`) ever move `latest`. Set
`PECULIUM_VERSION` to one of them to run it on an existing database.

To publish a test build from `develop`:

```bash
make rc                        # dispatch from develop; publishes the rolling :develop tag
make rc REF=develop            # explicit ref (this is the default)
make rc VERSION=rc-2026-10-03  # also publish a named tag (rc-<date>) for pinning
```

`gh workflow run` uses the repository default branch unless `--ref` is given, so
`make rc` passes `--ref develop` for you. Without it a manual dispatch would
build `main`, not `develop`. The `VERSION` value is only an extra image tag,
never the ref to build.

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

The current release is **v1.0.0** (30 Sep 2026): the first official release under the
Peculium name, adding the admin area (user governance and server settings), per-user and
server-wide backup & restore, and — from this release on — pull-only installation from the
public GHCR images.

Active development on the [`develop`](https://github.com/alv67/peculium/tree/develop) branch — see [STATUS.md](STATUS.md) and [PLAN.md](PLAN.md) for the roadmap. The full release history is on the [project website](https://peculium.dev/releases) and in [GitHub Releases](https://github.com/alv67/peculium/releases).

## License

Released under the [MIT License](LICENSE). See [`licenses/`](licenses/) for the
third-party components and their licenses.
