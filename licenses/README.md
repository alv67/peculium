# Third-party licenses

This folder documents the open-source components used by Peculium and the
licenses they are distributed under. Peculium itself is released under the
[MIT License](../LICENSE).

## Scope

The inventory below lists **direct runtime dependencies** (the components
actually shipped and executed). Development-only tooling (linters, test
runners, TypeScript, Vite plugins) and transitive/indirect dependencies are
intentionally excluded from the table: they are not redistributed and their
licenses are tracked upstream.

The `License` column reports the license declared by each upstream package.
The authoritative license text always lives inside the package itself
(e.g. `node_modules/<pkg>/LICENSE`); no license texts are vendored here on
purpose, so they stay in sync with the pinned versions.

## Inventory

### Go backend (`backend/go.mod`)

| Module | Version | License |
|--------|---------|---------|
| github.com/go-chi/chi/v5 | v5.2.1 | MIT |
| github.com/go-chi/cors | v1.2.1 | MIT |
| github.com/golang-migrate/migrate/v4 | v4.18.2 | MIT |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause |
| github.com/jackc/pgx/v5 | v5.7.4 | MIT |
| github.com/lestrrat-go/jwx/v2 | v2.1.4 | MIT |
| github.com/redis/go-redis/v9 | v9.7.1 | BSD-2-Clause |
| github.com/rs/zerolog | v1.33.0 | MIT |
| github.com/shopspring/decimal | v1.4.0 | MIT |
| golang.org/x/crypto | v0.36.0 | BSD-3-Clause |

### Frontend (`frontend/package.json`, production dependencies)

| Package | Version | License |
|---------|---------|---------|
| echarts | 5.6.0 | Apache-2.0 |
| lucide-svelte | 1.0.1 | ISC |
| svelte-echarts | 1.0.0 | MIT |

### Python microservice (`python-service/requirements.txt`)

| Package | Constraint | License |
|---------|------------|---------|
| fastapi | >=0.115,<0.116 | MIT |
| uvicorn[standard] | >=0.30,<0.31 | BSD-3-Clause |
| requests | >=2.32,<3 | Apache-2.0 |
| beautifulsoup4 | >=4.12,<5 | MIT |
| selenium | >=4.25,<5 | Apache-2.0 |

## Regenerating this inventory

When dependencies change, update the tables above and verify the licenses with
the following tools.

Go (installs `go-licenses`, then reports all modules):

```bash
go install github.com/google/go-licenses@latest
cd backend && go-licenses report ./... > /tmp/go-licenses.csv
```

Frontend (lists production dependencies with their license):

```bash
cd frontend && npx license-checker --production --summary
```

Python (lists installed packages with their license):

```bash
cd python-service && pip install pip-licenses && pip-licenses --format=markdown --with-urls
```

## Notes

- The tooling configuration in `.opencode/` is development-only and is not
  part of the distributed application, so it is not listed here.
- Apache-2.0 components require preserving their `NOTICE` files if any are
  present; this is handled automatically because the dependencies are installed
  from their published packages and not modified.
