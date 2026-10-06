# Peculium — The backend explained

> This document explains how the Peculium backend works: the program that
> manages financial data and makes it available to the website.
> No programming knowledge is required: server, database and language concepts
> are explained as we go. If you have never opened a code file, read chapter 2
> ("Basic concepts") and chapter 18 ("Reading Go code") first, then the rest
> will be clear.

---

## 1. What Peculium does

Peculium is an application for keeping track of investments: you record what
you buy and what you sell, and the app shows you what your securities are
worth, how much you have gained or lost, and how your portfolio has performed
over time.

To work, it needs two "worlds" that cooperate:

- The **frontend**: the web page you see in the browser (charts, forms,
  buttons).
- The **backend**: the program "behind the scenes" that reads and writes the
  data, performs the financial calculations and answers the frontend's
  requests.

This document is about the backend.

---

## 2. Basic concepts

Before going into details, a small glossary. If you already know these
concepts, feel free to skip this chapter.

- **Server / process**: a running program that waits for requests. The backend
  is a Go program compiled into an executable file.
- **API / endpoint**: a "phone number" that the backend exposes. The frontend
  calls an endpoint (e.g. `POST /auth/login`) and receives a response. In
  jargon we talk about *request* and *response*.
- **HTTP**: the "language" in which the frontend and the backend talk. The
  main verbs are `GET` (ask for data), `POST` (create something),
  `PATCH` (modify), `DELETE` (delete).
- **JSON**: the text format in which data travels inside a request. It is made
  of "name: value" pairs inside curly braces, for example
  `{"email": "mario@example.com", "name": "Mario"}`. You can read it like a
  filled-in form.
- **Database**: a program that stores data in an orderly way on disk. Peculium
  uses **PostgreSQL**.
- **Table**: inside the database, data is organized into tables (like
  spreadsheets) with rows and columns. The `users` table, for example,
  contains one row per user.
- **SQL**: the language for querying the database
  (`SELECT ... FROM ... WHERE ...`).
- **Primary key (PRIMARY KEY)**: the column that uniquely identifies each row
  (like a personal ID number).
- **Foreign key (FOREIGN KEY)**: a column that "points" to another table to
  link rows together (e.g. a transaction points to the portfolio it belongs
  to).
- **Migration**: a versioned SQL file that builds or changes the database
  schema. Migrations are applied in order.
- **Cache**: a copy of data that has already been computed or downloaded, kept
  aside to avoid repeating the same work.
- **Rate limit / throttle**: limiting the number of calls made to an external
  service in a unit of time, so you don't get blocked.
- **Container**: an isolated environment in which a program runs together with
  everything it needs. Peculium uses Docker.
- **Redis**: an in-memory database (very fast) that here works as a "shared
  counter" to keep Yahoo calls under control.

An analogy to tie it all together: the **backend is the kitchen of a
restaurant**. The frontend (the waiter) brings the order (the HTTP request),
the kitchen prepares the dish (reads the data and does the calculations) and
delivers it (the JSON response). The database is the pantry with the raw
ingredients.

---

## 3. Big picture

```mermaid
flowchart LR
    FE["Frontend — web page (port 3000)"]
    subgraph BE["Backend (Go)"]
        direction TB
        API["cmd/server/main.go — port 8080"]
        CHAIN["chi router → handler → service → repository"]
        WK["cmd/worker/main.go (separate process)<br/>updates prices in the background"]
        API --> CHAIN
    end
    PG[("PostgreSQL 16 — real data + materialized daily series")]
    RD[("Redis 7 — counter for limiting the calls to Yahoo")]
    YH["Yahoo Finance (HTTP) — prices, history, currencies (FX), stock splits"]

    FE -- HTTP --> API
    CHAIN --> PG
    CHAIN --> RD
    CHAIN --> YH
    WK --> YH
    WK --> PG
```

The pieces that run (defined in `docker-compose.dev.yml` for local development
and in `docker-compose.yml` for the pull-only release stack):

- **secrets-init** — generates the database password and the JWT signing key on
  first boot and stores them in the `peculium_secrets` volume; it then keeps
  running as the heartbeat that gates the database and the backend.
- **postgres** — the actual database: users, portfolios, transactions, prices,
  exchange rates, daily series.
- **redis** — an in-memory store used as a shared counter to avoid exceeding
  the allowed number of calls to Yahoo.
- **backend** — the REST API that talks to the frontend.
- **worker** — a separate process that updates prices in the background.
- **python-service** — a small FastAPI service (`python-service/`) that fetches
  ETF metadata (countries/regions and GICS sectors) and resolves ISINs from
  tickers. Resolution is **market-aware**: tickers with a recognized exchange
  suffix (e.g. `XMME.MI`) resolve via Morningstar on that specific market
  (ISIN can differ by listing), while bare tickers use JustETF. The Go backend
  calls it through `PECULIUM_PYTHON_SERVICE_URL`.
- **frontend** — the web page.

The backend is **a single Go program** that, depending on the argument passed
when it is started, does two different things (`cmd/server/main.go`):

- `server migrate` — applies the SQL migrations (builds the tables) and then
  terminates;
- `server` — starts the HTTP API and keeps listening.

Inside the container the command is `sh -c "/server migrate && /server"`, so
migrations always run before the API starts. The release stack gates the backend
on Postgres being healthy (`depends_on: condition: service_healthy`); the dev
compose prefixes the same command with `sleep 3`.

---

## 4. How the backend starts

The server startup sequence is:

1. **Configuration** — `config.Load()` reads the environment variables (with a
   default value if they are missing). Environment variables are configuration
   values passed to the program at startup (for example the database address),
   without having to write them in the code. The two secret values, the
   database password and the JWT signing secret, also accept a `*_FILE`
   variant (`PECULIUM_DB_PASSWORD_FILE`, `PECULIUM_JWT_SECRET_FILE`): the
   value is then read from the file it points at (trailing whitespace is
   ignored). Precedence is explicit environment value > file contents >
   default.
2. **Database connection** — `pgxpool.New(ctx, cfg.DSN())`. `pgxpool` is a
   **connection pool**: it keeps a few database connections ready, you take
   one when needed and give it back, so a connection is not opened from
   scratch for every request.
3. **Redis** — it tries to connect to Redis to activate the global counter of
   Yahoo calls. If Redis does not respond, the server uses a "fake" counter
   that never blocks and **keeps working anyway**.
4. **HTTP router** — the list of endpoints is prepared (the restaurant "menu")
   and common behaviors are added to all requests: logging, error recovery,
   an ID to trace every request, a 30-second timeout and CORS handling (the
   rules that allow the frontend to call the API).
 5. **Routes** — the endpoints are connected to the functions that handle
    them. Some endpoints are public (`/auth/*`), the others require
    authentication with a valid token belonging to an active account (see the
    chapter on authentication), and the `/api/v1/admin/*` ones also require an
    admin role. Outside `/api/v1`,
    `GET /healthz` is a public liveness probe: it answers
    `{"status":"ok"}` with HTTP 200 and does not touch the database or
    Redis. `GET /api/v1/version` is public too: it answers
    `{"version":"...","commit":"...","built_at":"..."}` from the build
    metadata injected as environment variables (`PECULIUM_VERSION`,
    `PECULIUM_COMMIT`, `PECULIUM_BUILT_AT`), with no login required.
6. **Series backfill** — after startup, a goroutine (a "thread" of execution
   that works in parallel with the rest of the program, so the API can answer
   requests while the calculation continues) rebuilds the daily series of all
   portfolios (chapter 9).

On shutdown the server waits for the `SIGINT`/`SIGTERM` signal and then
closes connections gracefully, without interrupting requests halfway through.

---

## 5. Layered architecture

The backend code is divided into **four layers**, each with a precise job.
The golden rule is that each layer can only use the one below it, never the
one above.

```
HTTP (router)
   │  JSON requests and responses
   ▼
handler   →  receives the HTTP request, understands what the frontend wants
   │           and sends the response. Never touches the database.
   ▼
service   →  business logic: login, average cost calculation,
   │           currency conversion, orchestrating the steps
   ▼
repository →  talks to PostgreSQL with plain, parameterized SQL
```

- **handler** (`internal/handler/`) — the "front door". It knows about HTTP
  requests but nothing about SQL.
- **service** (`internal/service/service.go`) — the "brain". It does the
  calculations and coordinates. It knows nothing about HTTP.
- **repository** (`internal/repository/`) — the "storekeeper". It writes the
  SQL queries and converts database rows into Go objects.

The three layers are **wired by hand** (manual dependency injection): first
the repositories are created, then the services (which receive the
repositories), then the handlers (which receive the services). No magic or
injection frameworks:

```go
repos := repository.New(dbPool)                                   // data layer
fetcher := price.NewYahooFetcher(repos, cfg.PriceFetchInterval,
    price.WithMinInterval(cfg.YahooMinInterval),                  // 400ms queue
    price.WithRateBudget(budget),                                 // Redis counter
)                                                                 // Yahoo client
svc := service.New(repos, jwtAuth, fetcher, cfg.LookupCacheTTL)   // logic
h := handler.New(svc, jwtAuth)                                    // HTTP
```

---

## 6. The database

The migrations (`backend/migrations/`, files numbered from `000001` to
`000019`) build the schema. The main tables:

| Table | Contains | Explanation |
|---|---|---|
| `users` | the users | email, name, password hash, role, account status (`active`, `pending`, `disabled`), base currency (`base_currency`, default EUR) |
| `assets` | the securities | ticker, name, type (stock, ETF, crypto...), investment class, price source, currency, exchange, sector, industry |
| `portfolios` | the portfolios | a portfolio belongs to a user and has a currency |
| `portfolio_shares` | the sharing | who else can see a portfolio (and with what role) |
| `transactions` | the operations | buy/sell/dividend/split/fee, quantity, price, date |
| `prices` | the daily prices | for each security and each day: open, close, volume |
| `fx_rates` | the exchange rates | how much 1 dollar is worth in every other currency |
| `splits` | the stock splits | e.g. a stock goes from 1 share to 4 shares |
| `asset_region_weights` | the geographic exposure | for each security, the weight of each macro-region |
| `asset_country_weights` | the per-country exposure | for each security, the weight of each ISO-3166 country |
| `asset_sector_weights` | the sector exposure | for each security, the weight of each GICS sector |
| `supported_currencies` | the list of currencies | which currencies can be used (chapter 11) |
| `lookup_cache` | the ticker-search cache | results already downloaded from Yahoo for autocomplete |
| `portfolio_series` | per-portfolio series | portfolio value and cost for each day |
| `asset_series` | per-security series | value and cost of each security for each day |
| `health_events` | the price-health events | records of stale/failed price updates on the health page |
| `server_settings` | the server options | a single row: registration auto-approval (chapter 14) |

Two fundamental ideas about the database:

1. **Rows are linked together.** A transaction has a foreign key to the
   portfolio and one to the security. So, given a portfolio, you can find all
   of its transactions.
2. **Primary keys are UUIDs.** A UUID is a long random identifier (e.g.
   `a3f2...`). Unlike sequential numbers, a UUID can be generated without
   asking the database, which makes working in parallel easier.

---

## 7. A typical request: the dashboard

Let's take `GET /api/v1/dashboard`, the richest endpoint: it serves the main
page by showing all portfolios, securities and gains.

What happens, step by step:

1. **Router**: the request arrives and is routed to the `GetDashboard`
   function.
2. **JWT middleware**: the user's token (chapter 14) is checked and the
   user's data is placed "in the context" of the request.
3. **Handler**: extracts the user's data, calls the service, and sends the
   JSON response.
4. **Service `GetDashboard`**: orchestration — it loads the user (and their
   **base currency**, see chapter 10), the user's portfolios, the detailed
   holdings (with the average cost, chapter 8) and the exchange rates for the
   currencies involved. The response carries `base_currency` and a `summary`
   roll-up converted into the base currency; `by_currency`, `portfolios` and
   `assets` stay expressed in their own currency. The `summary` and each
   `portfolios` entry split the numbers into nested `active` and `closed`
     objects (see chapter 8): `active` reports `invested`, `value`,
     `gain_loss` and `gain_loss_pct` of the lots still held plus the
     `dividends` of the positions that are still open (even if partially
     sold); open positions of assets that have no price at all are excluded
     from `invested` and `value` (their cost has no market value to be
     compared with, and counting it would fake a -100% loss), while their
     dividends still count. `closed` reports `invested` (AVCO cost of the
     sold lots),
     `proceeds` (net sale proceeds plus the dividends of the fully closed
     positions), `realized` (proceeds − invested) and `realized_pct`
     (realized / invested × 100). The response also carries
     `invested_assets`: the single consolidated list of the currently invested
     assets, aggregated by asset across all the user's portfolios (the same
     asset held in several portfolios is one row) and converted into the base
     currency. Each row carries `asset_id`, `ticker`, `name`, `currency` (the
     asset's own), `invested` (cost of the open quantity), `value` (market
     value, at cost when the asset has no price — `has_price: false` — or when
     its FX rate to the base currency is missing, so no fake loss appears),
     `gain_loss` and `gain_loss_pct`; rows are ordered by `value` descending
     and closed positions (qty ≤ 0) are excluded.
5. **Repository**: runs the SQL queries, for example the query that loads the
   portfolios with a `LEFT JOIN` on the sharing table.

### The time-weighted return chart (`GET /dashboard/performance`)

`GET /api/v1/dashboard/performance?granularity=month|year` returns one chart
for the whole wealth (all portfolios aggregated, converted to the user's base
currency) instead of one series per portfolio. `granularity` defaults to
`month`; any other value is rejected with 400. The response is
`{currency, granularity, buckets[]}`, where each bucket carries `period`
(`YYYY-MM` for months, `YYYY` for years), `return`, `twr`, `invested` and
`value`:

- the **`return`** is the bucket's **true time-weighted return (TWR)** in
  percentage: the returns are measured **daily** and linked geometrically,
  `return = (Π (1 + r(d)) − 1) × 100` over the days `d` of the bucket, where
  each day carries `r(d) = (V(d) − V(d−1) − flow(d)) / V(d−1)` when
  `V(d−1) > 0` and is skipped (factor 1) otherwise — the first day and the
  gaps of a fully liquidated wealth measure no return, so liquidating and
  reopening a position never distorts the chart. `V(d)` is the **market
  value only**: `V(d) = mv_priced(d) + bond_at_cost(d)`, the FX-converted
  market value of the priced assets plus the cost basis of the unpriced
  ones (a bond carried at cost while held; already zero once fully sold).
  Realized P&L never enters `V`: it is already captured by the sell flow;
- the **`twr`** is the cumulative time-weighted return compounded over all
  days up to the bucket end: `(Π (1 + r) − 1) × 100`, i.e. the geometric
  linking of the bucket returns too;
- **`invested`** is the net capital deployed: the cumulative sum of the
  base-currency *capital* flows up to the bucket end — the same flows but
  with dividends counted as 0 (distributions are income, not capital; a
  sell above cost can make it temporarily negative);
- **`value`** is the market value `V` at the bucket's last date (priced
  assets at market value, unpriced ones at cost).

External cash flows per transaction — placed at the **end of the day** —
converted to the base currency at the FX rate of the transaction date (the
flow is skipped when that rate is missing): `buy → +(qty·price + fees)`
(deposit), `sell → −(qty·price − fees)` (withdrawal), standalone
`fee → +feeAmount` where `feeAmount = qty·price`, or the price alone when no
quantity is set; `dividend → −divAmount` (same amount rule; income taken
out, so it yields a positive return contribution without touching
`invested`) and `split → 0`.

The daily points come from the materialized per-asset series
`asset_series` (via `Series.FindPortfolio`) in the portfolio currency and
are converted to the base currency per date through the USD-pivoted FX
history: while an asset's rate is missing, its last converted value is
forward-filled (zero before the first successful conversion), and every
asset keeps carrying its last converted value on the dates where it has no
new point. The market values are aggregated per date into `V`, the flows are
bucketed per day, and the walk runs over the union of the series dates and
the flow dates in ascending order (the materialized series covers every
calendar day from each asset's first transaction, so in practice flow days
always carry a market observation). Bucket periods are monotonic over the
dates, so the last date of each month/year seals its bucket and the buckets
are emitted in ascending order — buckets with no data are simply omitted.
`return` and `twr` are rounded to 4 decimals, `invested` and `value` to 8.

### The paginated transactions list (`GET /portfolios/{id}/transactions`)

`GET /api/v1/portfolios/{id}/transactions?limit=&offset=` returns a
pagination envelope — `{transactions[], total, limit, offset}`. `limit`
defaults to 20 and is clamped to a
maximum of 100 (a larger value comes back as `limit: 100`), `offset`
defaults to 0; a non-numeric value on either parameter is a 400, and so is a
negative one (surfaced by the service as `ErrInvalidInput`). The page is
ordered by `date DESC, created_at DESC, id DESC` — a fully deterministic
tie-break, so two transactions sharing a date can never straddle pages or
repeat. Ownership is enforced by the service before querying: a portfolio
that belongs to someone else is a 403 and a missing one a 404. `total` is the
full number of transactions in the portfolio, not the page size, so a page past
the end simply yields an empty `transactions` array with the correct
`total`.

The list also accepts four optional, freely combinable filters (all absent =
the behaviour above):

| Param | Meaning | Invalid value |
|-------|---------|---------------|
| `type` | exact transaction type: `buy`, `sell`, `dividend`, `split`, `fee` | 400 `invalid type` |
| `asset_id` | exact asset UUID | 400 `invalid asset_id` |
| `from` | inclusive start date, strict `YYYY-MM-DD` | 400 `invalid from: date must be YYYY-MM-DD` |
| `to` | inclusive end date, strict `YYYY-MM-DD` | 400 `invalid to: date must be YYYY-MM-DD` |

The query is parsed into a `model.TransactionFilter` (typed: `Type string`,
`AssetID *uuid.UUID`, `From/To *time.Time` UTC midnights) by
`model.ParseTransactionFilter`; the service threads it into both
`FindByPortfolioPage` and `CountByPortfolio`, so **`total` reflects the
filtered count** — the frontend can render "1–20 of 42" over the filtered
set. The repository builds a dynamic parameterized `WHERE` from the filter
(placeholders numbered by a counter, values only ever bind parameters —
never interpolated); date bounds compare `t.date::date` against the parsed
days so the match is calendar-day based and inclusive on both ends (the
column is `TIMESTAMPTZ`). Ordering and pagination semantics follow the rules
above.
A non-empty `filter.Type` outside the allowed set is rejected by the service
with `ErrInvalidInput` even when the caller bypasses the HTTP parser.

### The per-portfolio TWR chart (`GET /portfolios/{id}/performance/buckets`)

`GET /api/v1/portfolios/{id}/performance/buckets?granularity=month|year`
applies exactly the same daily TWR model to a **single portfolio**, expressed
in the **portfolio's own currency**: same `granularity` default (`month`, any
other value → 400) and same `{currency, granularity, buckets[]}` response
shape as the wealth-wide chart. It is simpler than the dashboard version
because nothing is converted to a base currency: the materialized per-asset
series is already denominated in the portfolio currency, so `V(d)` needs no
FX leg at all. Only the cash flows — recorded in the asset currency — are
converted to the portfolio currency at the FX rate of the transaction date
(via the USD-pivoted history), and a transaction whose rate or asset
currency is unknown is skipped, exactly like on the dashboard. Ownership is
enforced before the cache lookup (403 for someone else's portfolio, 404 for
a missing one), and the result is cached under the `pf-perf` key as
`{portfolioID}:{granularity}`.

### The portfolio summary (`GET /portfolios/{id}/summary`)

The portfolio detail page shows the same "Investments" card the dashboard
uses: alongside the flat fields (`total_cost`, `total_value`, `gain_loss`,
`gain_loss_pct`, `realized_gl`, `unrealized_gl`), the response carries the
nested `active` and
`closed` objects with exactly the same rules as the per-portfolio entries of
the dashboard (see above): `active` sums the cost and market value of the
open lots (`invested`, `value`, `gain_loss`, `gain_loss_pct`) plus the
`dividends` of the positions that are still open (even partially sold), while
open positions of unpriced assets stay out of `invested`/`value` (their cost
has no market value to compare with) but keep counting their dividends;
`closed` reports `invested` (AVCO cost of the sold lots), `proceeds` (net
sale proceeds plus the dividends of the fully closed positions),
`realized` and `realized_pct`. Everything is expressed in the **portfolio
currency**: no base-currency conversion is applied and only the market value
goes through the asset→portfolio FX factor, so a value whose rate is missing
is skipped and surfaced by the `fx_missing_count` /
`fx_missing_value` data-quality fields. The figures are rounded by the same
`finalizeBreakdowns` helper the dashboard uses, and the `holdings` list is
also returned.

The typical Go pattern for reading multiple rows is:

```go
for rows.Next() {
    rows.Scan(&x, &y)   // reads one row and puts it into variables
}
```

The program goes through the rows one at a time (`for rows.Next()`) and copies
them into variables (`Scan`). Go requires you to **map every column by hand**
into a variable, with no magic shortcuts: for each row, you explicitly decide
which value goes into which variable.

---

## 8. AVCO — average cost

AVCO (Average Cost) is the **logic for calculating how much a security
position is worth**, just like your broker does. It is the financial heart of
the application, in `internal/position/position.go`.

Imagine buying 10 shares at 100 € and then another 10 at 120 €. How much did
you spend in total? 2,200 €. Your **average cost** is
2,200 / 20 = 110 € per share. AVCO keeps track of this number as operations
happen.

### The `State`

For each security, the engine keeps a "state" of the position:

```go
type State struct {
    Qty      decimal.Decimal  // number of shares you own
    Avg      decimal.Decimal  // average cost
    Cost     decimal.Decimal  // total invested
    Realized decimal.Decimal  // realized gain/loss

    ClosedCost decimal.Decimal // AVCO cost of the sold lots
    Proceeds   decimal.Decimal // net sale proceeds
    Dividends  decimal.Decimal // dividends received
}
```

> What is a `struct`? In Go a `struct` is a container that groups several
> named values together: it is like a "card" with several fields. Here the
> "position state" card has the open-lot fields (quantity, average cost,
> total cost, realized gain/loss) plus the cumulative metrics of the sold
> lots and of the distributions (cost of sold lots, net proceeds, dividends)
> that feed the active/closed dashboard breakdown. Each cumulative
> field also has a
> `*CCY` twin expressed in the asset currency. `decimal.Decimal` is the type
> of the numbers (precise decimal numbers, suitable for money, with no
> rounding errors).

The idea is simple: operations don't modify data randomly, they update the
card in an orderly way, operation after operation.

### `Apply` — how the state changes with each operation

- **Buy**: add `quantity × price + fees` to the cost, then recompute the
  average cost: `Avg = Cost / newQuantity`.
- **Sell**: the cost of the sold part is `Avg × quantity`. You subtract it
  from the total cost; the difference between that and the proceeds becomes
  `Realized` (the gain or loss already "banked"). The sold-lot cost and the
  net proceeds are also accumulated into `ClosedCost` and `Proceeds`, so a
  partially sold position keeps the closed portion separate from the still
  open lots.
- **Split**: the quantity is multiplied by the ratio (e.g. 1 becomes 4), but
  the average cost is **divided** by the same ratio: the total cost does not
  change.
- **Fee**: added to the cost.
- **Dividend**: added to `Realized` and, separately, to `Dividends` (the
  dashboard then classifies them by the position state: dividends of a still
  open position — even partially sold — join the `active` group, while those
  of a fully closed one are included in the `closed` proceeds).

### `Walk`

`Walk` takes all transactions, groups them by portfolio and by security,
sorts them by date and **injects the splits** into the timeline. A split is a
fact about the security, so it applies to every portfolio that holds that
security.

---

## 9. Materialized daily series

To draw the charts, each portfolio needs a value for every day from the first
operation to today. This series is not recomputed on every request: it is
**saved (materialized)** in the database in the `portfolio_series` and
`asset_series` tables (`internal/series/series.go`).

The calculation logic for a single day is:

```go
for i, d := range dates {
    // apply the transactions that happened up to date d
    // apply the splits that happened up to date d
    // take the last available price ≤ d
    mv := st.Qty.Mul(priceByAsset[aid][priceDates[pricePos-1]]).Mul(rawFactor).Mul(factor)
    agg[i].MarketValue = mv   // added to the portfolio total
}
```

Two details in the market value calculation:

- `rawFactor` — Yahoo historical prices are **adjusted for splits**, and this
  factor brings them back to current reality;
- `factor` — the conversion from the security's currency to the portfolio's.

When the series is recomputed:

- on every change to transactions or the portfolio (`AddTransaction`,
  `UpdateTransaction`, `DeleteTransaction`, `UpdatePortfolio`, import);
- after every price update (both from the worker and from a manual request);
- at server startup (the backfill mentioned in chapter 4).

The rebuild is driven by `Recompute(portfolioID)` (one portfolio) or
`RecomputeAll` (all of them). Reading the charts then becomes a simple read
from the database, without redoing all the calculations every time a page is
opened.

---

## 10. Currencies and exchange rates (FX)

Portfolios and securities can be in different currencies. To add or compare
values, you need to convert.

1. The backend downloads from Yahoo the **exchange rates from the dollar (USD)
   to the other currencies** (e.g. `USDCHF=X` for the franc) and saves them in
   the `fx_rates` table.
2. At calculation time, `series.LoadRates` loads the rates of the currencies
   that appear in the holdings.
3. To convert from currency A to currency B, `series.FxFactor` uses the
   **double step through the dollar**: `(USD→B) / (USD→A)`.

> There is no need for a table with every possible pair: one row per currency
> (the rate against the dollar) is enough, and the rest is computed.

If a rate is missing, the conversion is not available and the application
signals it (the model shows the `fx_missing` field).

**The base currency.** Every user has a preferred currency stored
in `users.base_currency` (default EUR) and editable via `PATCH /users/me`
with the `base_currency` field (an omitted/empty value keeps the stored one;
a non-empty value must be an enabled currency from the whitelist, chapter 11,
otherwise the request is rejected with 400). All wealth-wide dashboard
aggregations are converted into it: `GET /dashboard` returns `base_currency`,
a `summary` roll-up in the base currency — nested `active` (invested, value,
gain/loss of the lots still held, plus the dividends of the open positions;
unpriced open positions stay out of invested/value, see chapter 7) and
`closed` (invested = AVCO cost of the sold lots, proceeds = sale proceeds
+ dividends of the fully closed positions, realized = proceeds − invested,
realized_pct) — where
amounts whose rate is missing are excluded from the totals and reported by
`fx_missing_count`/`fx_missing_value` (only non-zero amounts are flagged);
`GET /dashboard/performance` charts the wealth-wide percentage time-weighted
return (true TWR with daily geometric linking, with the invested/value
capital series) by month or year in the base currency (chapter 7);
`GET /portfolios/{id}/performance/buckets` charts the same buckets for a
single portfolio, staying in that portfolio's own currency (chapter 7);
`GET /dashboard/allocation` is expressed in the base currency too and
aggregates every portfolio into the wealth-wide `classes`, `regions`,
`countries` and `sectors` breakdowns. The
per-currency (`by_currency`) and per-portfolio
(`portfolios`, `assets`) sections of the dashboard keep their own currency.
The `invested_assets` list is in the base currency too: the user's open
positions aggregated per asset across all their portfolios (chapter 7).

---

## 11. The list of supported currencies

Not every currency in the world can be managed. The application keeps a
**whitelist** (an allowed list) in the `supported_currencies` table, which
already contains two base currencies: USD and EUR.

Currencies can be added and removed from `/settings/currencies`
(GET = list, POST = add, DELETE = remove):

- **Adding** a currency: first it checks that Yahoo knows the conversion
  USD → that currency. If it does not, the currency cannot be managed and the
  request is rejected.
- **Removing** a currency: you cannot remove the dollar (it is the basis of
  everything) nor a currency that is still used by some security or
  portfolio.

---

## 12. Prices and the connection to Yahoo

The `internal/price/` package is in charge of talking to **Yahoo Finance** to
fetch prices, history, exchange rates, splits and security search results.

### The Yahoo client (`YahooFetcher`)

It has an HTTP client with a 15-second timeout, two "cooldown maps" per
security (history and splits) and a **`sync.Mutex`** protecting those maps.

> What is a `Mutex`? If several parts of the program (the goroutines) work in
> parallel and share the same "maps" (the lists of what has already been
> updated), they could write at the same moment and corrupt the data. The
> `Mutex` (from "mutual exclusion") ensures that only one goroutine can access
> the maps at a time: it is like a door with a single key. Other programs
> often call it a "lock".

### Smart refresh (`RefreshStale`)

To avoid calling Yahoo unnecessarily, before updating a price it checks three
criteria:

1. there has never been a price → download;
2. the last update is very recent → skip (throttle);
3. the last closing price is already the one of the expected business day →
   skip.

Only assets with `price_source = 'yahoo'` (or empty, for safety) are ever
sent to Yahoo — for current quotes, history and splits; assets with
`price_source = 'manual'` or `'none'` are skipped entirely and never reported
as stale.

When updates are needed, current quotes and exchange rates are fetched **in
batch** using Yahoo's `spark` endpoint: a single call for groups of 50
securities. Securities missing from the response are requested one by one.

### Keeping the calls under control

Yahoo limits the number of calls it accepts. Every Yahoo call therefore passes
through two "brakes":

1. a **FIFO queue** (first in, first out) that guarantees a **minimum
   interval** between one call and the next (default 400ms, configurable with
   `PECULIUM_YAHOO_MIN_INTERVAL`);
2. a **shared global counter** (`RateBudget`), implemented in Redis: a cap of
   requests per time window (default 8 requests per second, configurable with
   `PECULIUM_YAHOO_GLOBAL_RATE` and `PECULIUM_YAHOO_GLOBAL_WINDOW`). The counter is
   shared between the server and the worker, so the two processes together do
   not exceed the limit.

If Redis is unreachable, the counter is disabled and the app proceeds anyway
(it takes more risk, but it does not get stuck).

### Long fetching runs as a queued job

The operations that talk to an external site and can take a long time
(a price refresh, a full history backfill, the global metadata backfill, the
per-app-load asset sync) do **not** run inside the HTTP request. The request
only **queues a job** and answers `202 Accepted` with
`{"job_id": …, "status": "queued"}`; the worker drains the queue with its own
context, so a slow provider or a large database can never exceed the request
timeout.

- `POST /prices/refresh` (optionally `?portfolio_id=`) queues a `price_refresh`.
- `POST /assets/{id}/backfill-history` queues a `history_backfill` (full price
  history **plus** the asset's split metadata).
- `POST /assets/sync` queues an `asset_sync`: the incremental splits + history
  update for every Yahoo-priced asset (run once per app load).
- `POST /assets/backfill-meta` queues a `meta_backfill`.
- Creating an asset queues a `history_backfill` for it.
- A second request for the same job while one is still queued or running
  returns that open job instead of stacking a duplicate (deduplicated by
  type + target).
- The client follows progress by polling `GET /jobs/{id}` (and `GET /jobs` for
  the recent list). A job is `queued`/`running` while open and ends
  `done`/`failed`/`partial`; `processed`/`total` report per-item progress.

The provider-facing calls queued this way are the same ones described below;
the queue only moves *where* they run, not what they do. Reads never fetch:
`GET /portfolios/{id}/history` recomputes the stored series and serves it, and
the market data is brought up to date by the jobs above. Read-only previews
(`fetch-exposure`, `fetch-etf-exposure`, `fetch-morningstar-exposure`) stay
synchronous because the caller needs their result inline.

### Reporting problems

Price-update outcomes are recorded in the **health log** (`health_events`):
per-asset issues carry a stable code — `rate_limited` (Yahoo refused because
of too many calls), `http_<status>` (a specific HTTP error) or `error` — each
with a real duration in `duration_ms`. Events raised while a job runs are
tagged with that job's id, so the Admin → Data & Sync page can correlate a
failure with the queue row that produced it.

A queued job also emits its own **lifecycle** rows (event type `job`): one
`job_started` when the worker claims it and one terminal
`job_completed`/`job_partial`/`job_failed` with the run duration and the
processed/total counts. These meta rows are kept out of the provider success
rate (`GET /health/prices`) and from a job's own item rollup. Terminal events
are written on a fresh short-lived context, so the outcome of a job that ran
out of its deadline is still recorded (a timeout is not swallowed). Failed
per-bar saves and split saves are aggregated into one event per asset and
operation (count + first error), never one row per date. A **successful**
history backfill and split check each record one success row per asset (with
the bar/split count), so the page lists every backfill call, not just the
failures; assets skipped by cooldown or already up to date leave no row.

`GET /jobs/{id}` (and `GET /jobs`) expose the derived `duration_ms`, a readable
`target_label` (asset ticker / portfolio name, resolved in one batched query
per page; empty for global or unresolved targets) and a `summary` with the
`ok`/`failed` item counts of each job.

A job and its health events form one unit: the `health_events.job_id` foreign
key cascades, so deleting a job removes its events. Retention uses a single
window, `PECULIUM_HEALTH_RETENTION` (default 90 days): the worker prunes jobs
older than that window regardless of status (aged by
`COALESCE(finished_at, created_at)`), taking their events with them, plus
standalone events (no job) older than the window. It runs at startup and on
every price-refresh tick. An admin can also wipe everything on demand with
`DELETE /admin/health/events`, which clears every job and event and answers
`{"deleted": N}` (the number of events removed).

### History and splits

- **History (`EnsureHistory`)**: brings the prices back from the first
  operation to today, resuming from where it stopped (downloads everything if
  missing, otherwise only from the last available date). The save is
  idempotent per date: rewriting the same day does not create duplicates.
- **Full history (`HistoryAsset.Full`)**: when an asset has never been
  backfilled (`assets.history_backfilled = FALSE`), the first sync downloads
  the **complete** price history from Yahoo (from 1970) in one pass, then
  flips the flag to `TRUE`. From then on, the sync is incremental. The asset
  detail page exposes this as **"Backfill storico completo"**
  (`POST /assets/{id}/backfill-history`): it forces a full re-download and
  invalidates the cached prices.
- **Splits (`EnsureSplits`)**: downloads the split events from Yahoo and saves
  them (also idempotently on `asset_id, date`).

### Asset profile and exposure

For the asset detail page the backend also talks to Yahoo's `quoteSummary`
endpoint (which requires a short-lived **crumb** + session cookie handshake,
see `meta.go`):

- **Profile (`FetchAssetProfile`)**: the `assetProfile` module gives the GICS
  `sector` and `industry` of a single stock.
- **Sector exposure (`FetchAssetExposure`)**: for an ETF, the `topHoldings`
  module exposes `sectorWeightings` (a fraction per sector key), which the
  backend converts to our canonical 11 GICS sectors (percentage). This fetch is
  a **read-only preview**: neither the sector weights nor the profile fields
  (sector/industry/country) are persisted — saving happens only through
  `PUT /assets/{id}/exposure`.
- **ETF exposure via JustETF (`FetchETFExposure`)**: the backend can fetch
  the **complete** country/region and sector exposure of an ETF from the
  `python-service` microservice (`POST /assets/{id}/fetch-etf-exposure`). The
  python service reads JustETF (full tables via its "Show more" AJAX), and the
  Go `geo` package maps each country to a macro-region and normalizes sectors
  to the GICS set. If the asset has no ISIN, it is **resolved automatically
  from the ticker** (the `.MI/.DE/.L...` suffix is stripped, results are ranked
  by name similarity against the asset) and persisted on the asset — that ISIN
  write is the only thing the fetch stores. The fetch is otherwise a **read-only
  preview**: the raw countries (normalized to ISO-3166 alpha-2 codes), the
  derived regions and the GICS sectors come back in the three dimensions —
  `countries`, `regions` and `sectors` — but are NOT persisted; they reach the
  database only when the user saves with `PUT /assets/{id}/exposure`.
- **ETF exposure via Morningstar (`FetchMorningstarExposure`)**: a second
  source is available: `POST /assets/{id}/fetch-morningstar-exposure`
  (ETF-only; when the ISIN is missing it is auto-resolved via Morningstar on the
  ticker's market). The python-service
  endpoint `GET /api/v1/etf/{isin}/morningstar-exposure` uses a **custom
  resolver** (`app/morningstar.py`, no mstarpy): the `global.morningstar.com`
  SAL endpoints mstarpy targets are blocked (403), so the resolver runs a
  **headless Chromium bootstrap** (container packages `chromium` + `chromium-driver`
  + `xvfb` via apt) to clear the AWS WAF challenge on `www.morningstar.com` and
  obtain the **Bearer JWT** from `/api/v2/stores/maas/token` (~1h cache); the
  data calls then go through `requests` with bearer+cookies to
  `www.us-api.morningstar.com/sal/sal-service/etf/...` (sectors
  `portfolio/v2/sector/{sid}/data`, countries
  `portfolio/regionalSectorIncludeCountries/{sid}/data`, official **regions**
  `portfolio/regionalSector/{sid}/data`, ISIN→securityId via
  `www.morningstar.com/api/v2/search?q={isin}`). Country weights are kept as
  reported: Morningstar returns the full country list (51 entries, many zero; the 10×6
  paging is only client-side UI), with a residual share not exposed as a
  country, so the weights sum to ~95% (no forced scaling to 100).
  The Morningstar region keys (`northAmerica`, `unitedKingdom`, `japan`,
  `australasia`, ...) are mapped 1:1 onto the canonical Peculium taxonomy and
  returned as the `regions` dimension. Like the other fetches this endpoint is a
  **read-only preview**: countries, sectors and the official regions (when
  present) are returned but NOT persisted — the only write is an auto-resolved
  ISIN; saving happens through `PUT /assets/{id}/exposure`. Otherwise (or for
  JustETF) the preview derives the regions from the countries server-side via
  `price.AggregateRegions` as an explicit region dimension (this derivation
  belongs to the provider prefill only: the manual
  `PUT` with `{countries}` never rewrites the regions). The residual
  (100 − country sum) lands in the `Other / Not Classified` region,
  so the previewed regions always sum to 100. The canonical regions are
  **10 + `Other`**: North America, Latin America, United Kingdom,
  Europe Developed, Europe Emerging, Africa / Middle East, Japan, Australasia,
  Asia Developed, Asia Emerging, Other / Not Classified (UK/Japan/Australasia
  are standalone; TW/KR are Asia Developed).
- **Asset class (asset-info refresh / `GET /assets/meta`)**: Yahoo does not
  expose `assetClass` (the `quote` quoteSummary module does not exist; v7
  `/quote` does not return it). The class is derived in `FetchMeta` via
  `geo.ClassifyAssetClass` (Morningstar-style fund category
  `defaultKeyStatistics.category`/`fundProfile.categoryName` fetched by
  `FetchFundCategory`, plus a name heuristic; type-based default). It is
  coupled to the asset-info refresh (used at creation and by "Aggiorna da
  Yahoo") and **not** to the sector read: `fetch-exposure` does not update it.
  A manual override in the asset editor always wins (only applied when empty or
  `other`).
- **Geographic exposure**: for a single **stock**, the country (from the asset
  profile) is mapped to a macro-region at 100% (`geo.RegionForCountry`).

A note on **ISIN**: Yahoo does **not** expose the ISIN in any module. For ETFs
the value is resolved automatically from the ticker through the JustETF
service; the field remains editable by hand on the asset page as a
fallback. The exposure responses (`GET/PUT /assets/{id}/exposure`,
`fetch-exposure`, `fetch-etf-exposure`, `fetch-morningstar-exposure`) include
the persisted `isin` field (`AssetExposure.ISIN`), so the frontend can sync it
after a fetch. The `GET /assets/{id}/exposure` response exposes the
**countries** dimension zero-filled across the full canonical ISO list, and
`PUT /assets/{id}/exposure` accepts an optional `countries` array: it keeps
only canonical ISO codes and **rejects a country sum above 100** (sums below
100 are legitimate — provider coverage is often ~92–95%, and there is no
minimum). Saving countries does **not** touch the regions dimension: each
dimension in the `PUT` body is saved independently, so a `{countries}` body
leaves the stored regions exactly as they are. Regions are recomputed from
countries only through the explicit derive endpoint (or updated explicitly via
`{regions}`). Users can add/remove countries from the canonical
list and edit their individual weights. A companion endpoint
`POST /assets/{id}/exposure/derive` computes the regions from a `{countries}`
body **without persisting** (used by the "Calcola da paesi" button in the UI).
**Region save validation**: the explicit `regions` array accepts a total
**≤ 100** (below 100 is valid; above 100 is rejected). The UI never shows or
edits "Other / Not Classified", so when the client sends regions summing below
100 with no Other row, the backend **injects the residual into
`Other / Not Classified` before persisting**, keeping the stored invariant
"regions sum to 100" that portfolio geography aggregation relies on. Sectors
keep the exact 100 ± 0.5 rule.
**Exposure provenance**: every saved dimension also records *where it came
from*. `PUT /assets/{id}/exposure` accepts optional `countries_source`,
`regions_source` and `sectors_source` strings (e.g. `morningstar`, `justetf`,
`yahoo`); when a dimension is present in the body but its source is missing or
empty it is recorded as `manual`. The `GET/PUT /assets/{id}/exposure` responses
then include a `provenance` map — only for the dimensions persisted at least
once, omitted when empty — with one `{ "source", "updated_at" }` entry per
dimension, e.g. `"countries": { "source": "morningstar", "updated_at":
"2026-09-05T10:00:00Z" }`, so the UI can render "da Morningstar (2026-09-05)"
badges that survive a reload (persisted in `asset_exposure_provenance`,
migration `000017`). The fetch/prefill responses (`fetch-exposure`,
`fetch-etf-exposure`, `fetch-morningstar-exposure`) are previews and
deliberately carry **no** provenance: nothing is written until the user saves.

### Provider exposure cache (`exposure:<source>:<ISIN>`)

The two prefill fetches that hit the python-service
(`POST /assets/{id}/fetch-etf-exposure` and
`POST /assets/{id}/fetch-morningstar-exposure`) cache the **raw provider
payload** in the Redis lookup cache (`s.repos.Lookup`) under the key
`exposure:<source>:<ISIN>` (source is `justetf` or `morningstar`, the ISIN
is uppercased; the full Redis key is `pc:lookup:exposure:...`). The first
request for an ISIN runs the heavy fetch (Morningstar needs a Chromium/SAL
session) and stores the result; later requests are served from the cache and
never call the provider. Each source keeps its own entry, so prefilling from
JustETF does not warm Morningstar and vice versa. The TTL is
`PECULIUM_EXPOSURE_CACHE_TTL` (default 7 days). Country-less results are never
cached, so a transient empty fetch cannot stick for a week. The cache only
speeds up the read: entries are provider payloads, never stored weights
(saving still happens exclusively through `PUT /assets/{id}/exposure`).
`?refresh=1` (also `refresh=true`) on either endpoint forces a fresh provider
fetch that bypasses and rewrites the cache; it is a backend-only knob.
Yahoo's `fetch-exposure` is deliberately left uncached (it
is cheap and shares the Yahoo meta cache).

### Cache invalidation (`bumpRev`)

Every cached read (`cached()`, chapter 7) is keyed by a global **revision**
number. After any write (new prices, a backfill, an exposure update, ...) the
service calls `bumpRev` so the next read bypasses the stale cache. This is
also why `SyncAssetData` and the background sync after creating an asset
invalidate the price cache: otherwise a freshly backfilled asset would keep
showing an old, partial chart until the cache TTL expired.

### Security search (`lookup_cache`)

When the user searches for a security by ticker, the result is saved in the
`lookup_cache` table for a few days. So the next identical search does not
call Yahoo again.

---

## 13. The worker

The worker process (`cmd/worker/main.go`) is separate from the server. Every
interval (`PECULIUM_PRICE_FETCH_INTERVAL`, default 1 hour) it updates prices:

```go
ticker := time.NewTicker(interval)
// runs once immediately, then a loop that waits for:
//   - the next tick (update the prices)
//   - the shutdown signal (exit)
for { select { case <-ticker.C: FetchAll(); case <-ctx.Done(): return } }
```

Why a separate process and not a goroutine inside the server? **Isolation**:
if Yahoo goes down, you can restart the worker without touching the API. The
frontend does not notice anything.

At startup, the worker waits for the server to have applied the migrations (it
checks that the series tables exist) and only then starts. On every round,
after updating prices, it also recomputes the daily series of all portfolios
(chapter 9), so the data always stays in sync.

---

## 14. Authentication with JWT

Authentication uses **JWT** (JSON Web Token): a signed, "stateless" token,
meaning the server does not have to remember who is logged in, because the
token itself contains the data and a cryptographic signature (HMAC-SHA256)
that guarantees its authenticity. The signing secret lives in the
configuration.

At login, **two tokens** are issued:

- **access** (lasts 15 minutes): contains the user id, the email, the role
  and the type `token_type=access`. It is the ticket the frontend uses to
  call the APIs;
- **refresh** (lasts 72 hours): contains only the user id and the type
  `token_type=refresh`. It serves to obtain a new token pair when the access
  one expires (automatic rotation).

The middleware checks every protected request: if the token is missing,
invalid, or not of type `access`, the request is rejected.

Passwords are stored as **bcrypt hashes** (not in plain text).

> What is a *hash*? It is a "digital fingerprint" of the password: a string
> computed from the password with a one-way formula (you cannot go back from
> the fingerprint to the password). At login, the fingerprint of the entered
> password is recomputed and compared with the stored one. So even if someone
> steals the database, they cannot read the passwords.

### Account status and registration approval

Every account carries a **status** — `active`, `pending` (waiting for
approval) or `disabled` — and the status travels with the user object
returned by register, login and `GET /users/me`.

- The **first registrant** on an empty server becomes the active `admin`
  automatically. This happens inside one transaction that first locks the
  settings row, so two simultaneous signups can never both be treated as the
  first user.
- Following registrants get the `viewer` role. With the server setting
  `auto_approve_registrations` **on** (the default) they are `active` right
  away; with it **off** they are `pending`. Registration always succeeds —
  the status in the response is what tells the UI to show "waiting for
  approval".
- Login accepts only `active` accounts: a `pending` one is rejected with 403
  "account pending approval", a `disabled` one with 403 "account disabled"
  (the same rules apply to refreshing a token pair).

After the JWT check, a second middleware loads the account behind the token
id on **every** protected request and lets it through only while the stored
status is `active`: an unknown or deleted user gets 401, a non-active one
403. Approving or disabling an account therefore takes effect immediately,
even for tokens issued before the change.

### Admin area

Roles `owner` and `admin` are the administrator roles (an `owner` — the role
default of the schema — counts as admin everywhere); `editor` and `viewer`
are normal users. The admin endpoints sit under `/api/v1/admin`, behind the
JWT middleware, the active-status middleware and an admin gate that answers
403 for non-admins:

| Endpoint | Does |
|---|---|
| `GET /admin/users` | lists every account: id, email, name, role, status, created_at (the password hash is never exposed) |
| `PATCH /admin/users/{id}` | sets `role` and/or `status` (validated against the enums); refuses to demote or disable the **last active admin** with 409 |
| `POST /admin/users/{id}/reset-password` | sets a new password for the account (minimum 8 characters, no current password needed) |
| `GET /admin/settings` | reads the server settings (`auto_approve_registrations`, `updated_at`) |
| `PATCH /admin/settings` | updates `auto_approve_registrations` |
| `GET /admin/db/backup` | downloads a full database dump as a file attachment |
| `POST /admin/db/restore` | replaces the whole database from an uploaded dump (destructive, requires confirmation) |

### Database backup and restore (`GET /admin/db/backup`, `POST /admin/db/restore`)

These two endpoints work on the **entire PostgreSQL database** (every table,
every user) by running the `pg_dump` / `pg_restore` client tools, which the
backend server image ships (`postgresql16-client`, PostgreSQL 16); the worker
image does not carry them. The connection coordinates come from the same DB
configuration the app uses; the password reaches the tools through the
`PGPASSWORD` environment variable (never on the command line) and the TLS
setting through `PGSSLMODE`.

`GET /admin/db/backup` runs `pg_dump -Fc` — a full dump in the PostgreSQL
**custom format** (compressed, restorable selectively) — and pipes the tool's
output straight into the HTTP response, without ever holding the whole archive
in memory. The response is a download: `Content-Type: application/octet-stream`
and `Content-Disposition: attachment` with the file name
`peculium-db-<YYYY-MM-DD>.dump` (UTC date). The attachment headers are
committed before the tool runs, so a `pg_dump` failure surfaces as a truncated
download and the sanitized detail goes to the server log — tool output and
credentials are never leaked to the client.

`POST /admin/db/restore` accepts the archive as a **multipart form upload** in
the `dump` file field, together with the explicit confirmation
`confirm=replace` (as a form field or a query parameter): restoring **replaces
the entire database**, so the request is rejected with 400 without that exact
confirmation, as is a request with no dump file. The upload is written to a
temporary file (removed afterwards) and fed to
`pg_restore --clean --if-exists --no-owner --no-acl -d <db>`, which drops and
recreates the objects it finds in the archive. Success answers a JSON summary
(`dump_bytes`); failure answers 500 carrying the `pg_restore` error output with
the database password masked, so the admin can read what went wrong. Because
the archive contains the `users` table too, a restore rewrites the accounts:
existing sessions and tokens may reference rows that are gone, and the admin is
told to **log in again**.

---

## 15. Atomic operations — `WithTx`

When an operation involves several database writes, you want it to **either
succeed completely, or change nothing**. This is called an atomic
transaction.

```go
func (r *Repository) WithTx(ctx, fn func(*Repository) error) error {
    tx, _ := r.DB.Begin(ctx)
    defer func() { _ = tx.Rollback(ctx) }()   // if fn fails → undo everything
    rr := &Repository{ User: &userRepo{db: tx}, ... }
    if err := fn(rr); err != nil { return err }
    return tx.Commit(ctx)                      // if all is well → save everything
}
```

Two Go ideas worth noting:

1. **`DBTX` interface**: both the connection pool and a transaction expose the
   same methods (`Exec`, `Query`, `QueryRow`). The repositories therefore work
   identically both on the pool and inside a transaction.
2. The inner function receives a **"cloned" Repository** whose components are
   bound to the transaction. The `defer Rollback` after a successful
   `Commit` is a harmless no-op: if the function fails, however, it undoes
   everything.

An example use: importing a portfolio in "replace" mode deletes and recreates
the portfolio **atomically** — if a step fails, the old portfolio remains
intact.

### Portfolio export/import: versioning and backward compatibility

`GET /portfolios/{id}/export` produces a JSON document with a `version` field
(the format version, currently `1`); `POST /portfolios/import` consumes it.
The format is deliberately **additive**: the per-asset `price_source` and
`asset_class` fields the export writes are optional (`omitempty`), so
documents that omit them remain valid and recoverable.

When the importer creates an asset whose ticker does not exist yet, every
missing piece is filled with a default that satisfies the database
constraints: name falls back to the ticker, type to `stock`, currency to
`USD`, `asset_class` to the default class for the type, and `price_source` to
`yahoo`. An unknown `price_source` in the document does not fail the import:
it also falls back to `yahoo`. A document whose `version` is newer than what
the importer understands is rejected with a clear 400
(`unsupported export version N`) instead of a generic failure.

### Per-user backup and restore (`GET /backup`, `POST /backup/restore`)

`GET /backup` streams the whole data of the authenticated user as a single
downloadable JSON bundle (`peculium-backup-<email>-<date>.json`). Like the
portfolio export, the document is **versioned** (the `version` field,
currently `1`) and **additive**: optional fields may be missing on import,
and a bundle newer than what the server understands is rejected with a 400
(`unsupported backup version N`). The bundle carries:

- `user` — the email and name of the account it came from (informational)
  plus its **base currency**, the one user setting restore applies;
- `currencies` — the supported-currency entries in use: the base currency
  and every portfolio and asset currency;
- `portfolios` — one standard portfolio-export document per portfolio the
  user **owns** (portfolios shared into the account are not duplicated);
  each entry is self-contained and also importable on its own through the
  portfolio import endpoint;
- `assets` — the per-asset data no provider can refetch, keyed by ticker:
  the identity metadata, the **exposure** (country, region and sector rows
  plus, in the `*_source` fields, the provenance of every saved dimension)
  and the **manual price rows**.

Yahoo prices, FX history and health events are deliberately excluded: they
are provider data, refilled by the fetcher and the worker on the restored
server.

`POST /backup/restore?mode=add|replace` writes the bundle back onto the
authenticated account — the intended scenario is a server re-initialized
from scratch, where the user has recreated the account by hand. The body is
the bundle JSON itself. The mode decides what happens to existing data:

- `add` (the default when `mode` is omitted) is non-destructive: every
  portfolio of the bundle is imported as a new one and existing data is
  left untouched;
- `replace` first deletes the portfolios the user owns — CASCADE removes
  their transactions with them — and then imports the bundle. There is no
  server-side confirmation for `replace`: the client UI confirms before
  calling.

Assets are global and shared between users, so restore **never deletes**
any: an already-stored ticker is reused as-is — its shared identity
metadata is not overwritten — and a missing one is created from the bundle
with the same importer defaults the portfolio import applies. Exposure is
restored only for dimensions that actually carry rows (a dimension absent
from the bundle leaves the stored one untouched), and manual prices upsert
on `(asset, date)`. Currency entries add missing codes to the global
whitelist and never modify existing ones. The whole restore runs in a
single transaction, and once it commits the series of every created
portfolio is recomputed and a global **`asset_sync`** job is queued, so the
worker backfills the restored assets' price history and splits exactly as
after an import (the bundle carries no provider data).

The response is a summary of the operation: the `mode` applied and the
counts `portfolios_created`, `transactions_created`, `assets_created` and
`assets_reused`.

---

## 16. The complete data flow

Putting all the pieces together:

```
The user creates a security ──► POST /assets ──► CreateAsset
                                in background: downloads splits and history from Yahoo

The worker, every hour: FetchAll (quotes + currencies in batch) ──► RecomputeAll (series)

The user edits a transaction ──► Recompute(portfolio) → series updated

The administrator adds a currency ──► POST /settings/currencies
                                → verifies conversion on Yahoo → whitelist

The user opens the dashboard ──► GET /dashboard:
    holdings (AVCO) + exchange rates
    → summary converted into the user's base currency → JSON to the frontend

The user views the return chart ──► GET /dashboard/performance?granularity=month|year:
    daily per-asset points from the materialized series + per-date FX
    + daily external cash flows (priced assets at market value, unpriced at cost)
    → monthly/yearly true TWR (geometric linking of daily returns) + cumulative
      TWR buckets, in the base currency, with invested/value series → JSON to the frontend

The user opens a portfolio's return chart ──► GET /portfolios/{id}/performance/buckets?granularity=month|year:
    same daily TWR model for one portfolio, in the portfolio currency
    (series need no conversion; only the flows do) → JSON to the frontend

The user opens the asset detail page ──► GET /assets/{id}/quote (+ /prices?...&full=1):
    quote ranges + price history from the database → JSON to the frontend

The user edits the exposure ──► PUT /assets/{id}/exposure
    → saves asset_country_weights / asset_region_weights / asset_sector_weights
      (each dimension saved independently: countries never rewrite regions;
       country sum not enforced) → bumpRev

The user clicks "Calcola da paesi" ──► POST /assets/{id}/exposure/derive
    → {countries} → {regions} derived via AggregateRegions (no persistence) → fills the regions table

The user clicks "Aggiorna da Yahoo" ──► POST /assets/{id}/fetch-profile
    → quoteSummary (crumb) → sector/industry (+ issuer country) → saved on the asset

The user prefills the exposure ──► POST /assets/{id}/fetch-exposure
                                 | /fetch-etf-exposure | /fetch-morningstar-exposure
    → provider data (Yahoo sectors | python-service countries + sectors
      + official regions, GET /api/v1/etf/{isin}/[morningstar-]exposure)
      returned as a READ-ONLY preview — nothing persisted except an
      auto-resolved ISIN → bumpRev on that ISIN write only
    → the user confirms with the Save buttons → PUT /assets/{id}/exposure (persistence)
```

---

## 17. File map

```
backend/
├── cmd/
│   ├── server/main.go      # API startup, routing, migrations, series backfill
│   ├── worker/main.go      # background price updates
│   └── secrets/main.go     # secrets-init: generates secrets on first boot
├── internal/
│   ├── auth/jwt.go         # JWT: generation, validation, middleware
│   ├── config/config.go    # environment variables + connection DSN
│   ├── geo/geo.go          # macro-regions, GICS sectors, canonical ISO countries, country→region mapping
│   ├── handler/            # HTTP layer (auth.go, admin.go, admin_db.go, backup.go, portfolio.go, settings.go, ...)
│   ├── middleware/         # per-request account checks: active status + admin gate
│   ├── model/              # data structures with JSON tags
│   ├── position/           # AVCO engine (State, Apply, Walk)
│   ├── price/              # Yahoo client (yahoo.go, spark.go, meta.go, throttle.go, report.go, ...) + JustETF/Morningstar fetchers
│   ├── repository/         # SQL queries (repository.go = "hub" + asset.go + exposure.go + settings.go + WithTx + DBTX)
│   ├── secrets/            # secret materialization for the secrets-init image (Ensure, Health)
│   ├── series/             # materialized daily series (Recompute, LoadRates, FxFactor)
│   └── service/            # business logic (service.go, admin.go, backup.go, dbmaint.go)
├── migrations/             # versioned SQL (000001..000019)
└── go.mod
```

---

## 18. Reading Go code

The document contains fragments of Go code. Here are the basics for reading
them without ever having programmed.

- **Statements end at the end of the line.** Each line is an instruction: no
  parentheses or semicolons to hunt for. `//` marks a **comment**: the text
  after `//` is not code, it is a note for the reader.
- **`func` defines a function**: a block of code with a name, ready to be
  called. E.g. `func (s *Service) GetDashboard(...) { ... }`. The name before
  `(` indicates *which object it belongs to* (in this case the service).
- **The `{ }` braces delimit the body** of the function: everything between
  the braces is what the function does.
- **`:=` creates a variable** and assigns it a value in one step. E.g.
  `x := 5` means "create a slot called x and put 5 in it".
- **`struct` groups values**: a card with several named fields (like the
  "position state" card in chapter 8).
- **`package` and `import`**: `package` declares the "group" a file belongs
  to; `import` loads code already written by others (the libraries).
- **`context.Context`**: a "luggage carrier" that accompanies every request
  and carries background information (who the user is, when to stop). When
  the user closes the page, the context communicates it and the work in
  progress stops.
- **Functions return multiple values**, and the second one is almost always
  `err` (error). The Go rule: **errors are values to be checked**.
  `if err != nil { return err }` reads as: "if there was an error, stop and
  return the error". It is how Go signals problems: it does not crash the
  program suddenly, it lets the code author decide, line by line.

With these few rules, the code fragments in this document read like
sentences: "create the connection, if something goes wrong stop and signal it,
otherwise continue".
