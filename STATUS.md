# Peculium — Stato Progetto (30 Set 2026)

## Infrastruttura

| Servizio | Stack | Note |
|----------|-------|------|
| Backend | Go 1.25 + Chi + pgx + golang-migrate | Containerizzato |
| Frontend | SvelteKit 5 + TypeScript + Tailwind + ECharts | Containerizzato (nginx) |
| Database | PostgreSQL 16 | Con docker volume |
| Cache | Redis 7 | Caching dashboard/series, rate-limit Yahoo |
| Worker | Go (prezzi) | Container separato |
| Python Service | FastAPI + uvicorn + requests + bs4 + selenium (chromium headless) | ETF metadata da JustETF (esposizione paesi + settori) + Morningstar (esposizione paesi + regioni ufficiali + settori via resolver custom; auto-resolve ticker→ISIN per mercato) — EPIC B.5, B.14 |
| Container | podman + podman-compose su macOS | |

## Release

**v0.1.0** — prima release ufficiale su `main` (25 Ago 2026).
**v0.2.0** — seconda release su `main` (30 Ago 2026): EPIC A (data correctness & security)
e EPIC B completo (distribuzione geo/settoriale, asset class, FX history, charts).
**v0.3.0** — terza release su `main` (11 Set 2026): asset editing overhaul (price_source
Yahoo/Manual/None, chart in-place con YTD e marcatori split, modale esposizione) ed editing
dell'esposizione per-paese con fonte Morningstar/JustETF, cache e provenienza.
**v0.4.0** — quarta release su `main` (13 Set 2026): design system & dark mode (EPIC D),
rebuild delle pagine e dei componenti di dominio (EPIC E: asset/portafogli/modali/login/
impostazioni a tab), dashboard e dettaglio portafoglio rinnovati, Health più chiaro
(periodo Today/24h/100, paginazione, fix N/A) e CI GitHub Actions.
**v0.5.0** — quinta release su `main` (17 Set 2026): **EPIC I completa** (dashboard & portfolio
v2) — valuta base utente con aggregazione FX, riepilogo attivo/chiuso, grafico performance
time-weighted (barre mensili/annuali + linea cumulata) e grafico del capitale, allocazioni per
classe/settore/paese/macro-regione, tabella asset investiti consolidata, KPI e allocazioni del
dettaglio portafoglio allineati alla dashboard, performance a barre e transazioni paginate;
inclusi i fix import di export vecchi (#99) e P/L fittizio -100% sulle posizioni chiuse (#100).
**v0.6.0** — sesta release su `main` (21 Set 2026): **EPIC K completa** (redesign UX/UI) —
fondazioni (token, font self-hosted, tema di sistema, i18n IT/EN, primitive), shell adattiva
(bottom nav + FAB su phone, rail su tablet, header condensante), dashboard con hero e chip
periodo, pagine portafoglio/asset a tab, filtri Attività con undo e form a sheet, command
palette ⌘K, vista tabella dei grafici, palette CVD e drill-down delle allocazioni; include i
filtri transazioni e l'endpoint di drill-down lato backend.
**v0.6.1** — settima release su `main` (23 Set 2026): bug release — stato aggiornamento prezzi
spostato nell'header globale (stampa cliccabile + fascia qualità dati su ogni pagina), traduzione
completa IT/EN di dashboard, dettaglio portafoglio, liste, modali, Impostazioni e pagina Health
(con etichette tipo/classe/fonte asset locale-aware), fix mobile di menu entità, drill-down
allocazioni, tooltip dei grafici e command palette.
**v1.0.0** — prima release ufficiale col nome **Peculium** (30 Set 2026): **area Admin**
(gestione utenti — ruoli, approvazione delle registrazioni, disattivazione/riattivazione, reset
password — e impostazioni server), **backup e ripristino** per-utente (JSON, modalità Aggiungi o
Sostituisci) e a livello server (dump completo del database con conferma distruttiva), messaggi di
login chiari per account in approvazione o disattivati, e prima release distribuita come
**immagini pull-only su GHCR** (anche per linux/arm64); include le patch di sicurezza Dependabot
(golang.org/x/crypto, pgx, go-redis, chi, js-yaml, brace-expansion, devalue) e l'allineamento del
backend a **Go 1.25**.
Flusso: branch → PR su `develop` → merge → tag `v0.1.0`…`v0.6.1`/`v1.0.0` su `main`.

## Fase 0 — ✅ Completata

- [x] Struttura monorepo (backend + frontend + docker)
- [x] Docker Compose (Postgres, Redis, backend, frontend, worker)
- [x] Go module con tutte le dipendenze
- [x] Frontend migrato a SvelteKit (da React/Vite)
- [x] Auth JWT (access + refresh token, rotazione automatica)
- [x] API REST (auth, assets, portfolios, transactions, prices, settings, dashboard, health)
- [x] DB schema (users, assets, portfolios, portfolio_shares, transactions, prices, fx_rates, splits, series, health_events)
- [x] Makefile aggiornato per podman-compose

## Fase 1 — ✅ Completata

- [x] Registrazione/Login multi-utente
- [x] CRUD portafogli + export/import
- [x] CRUD asset con autocomplete e sync Yahoo Finance
- [x] Transazioni (buy/sell/dividend/split/fee) con JOIN asset
- [x] Dashboard: valore totale, gain/loss, allocazione, performance, ROI per asset
- [x] Valuta dinamica (€/$/£/CHF) e whitelist valute configurabile
- [x] Prezzi Yahoo con cache Redis, rate-limit/backoff, serie materializzate, health dashboard
- [x] `POST /api/v1/prices/refresh`
- [x] EPIC A — Data correctness & security (#3 #4 #5 #6):
  - Aggregazione coerente con FX mancante (bucket esplicito + forward-fill serie)
  - Enforce ownership su tutti gli endpoint analitici
  - Metric data-quality nel summary (`missing_country`, `missing_sector`, `stale_count`, `fx_missing_*`)
  - Rimosso codice SQL morto in `repository/portfolio.go`

### Da fare (Fase 1/2)
- [ ] Sharing portafogli tra utenti (portfolio_shares)
- [ ] Inserimento prezzo manuale nella UI
- [ ] Ricerca asset con autocomplete nel form transazioni
- [ ] Import CSV transazioni (l'import JSON esiste già)

## Problemi Aperti

### Yahoo Finance rate-limited (429)
Mitigato con rate-limit/backoff e throttling, ma Yahoo può comunque bloccare il container podman.
- Da macOS host → funziona
- Se il worker resta bloccato, valutare: script esterno su macOS (curl/cron) che POSTa i prezzi al backend,
  oppure API alternativa (Finnhub, Alpha Vantage con API key)

### Ticker europei
Asset europei su Yahoo usano suffisso exchange (es. VWCE.DE, VWCE.AS). L'utente deve sapere il
ticker corretto. Da documentare o aggiungere selezione exchange nell'autocomplete.

## Fase 2 — In corso

### EPIC B — Distribuzione geo/settore + FX history + Asset detail (#36) — 12 sub-issues
| Issue | Titolo | Componente | Stato |
|-------|--------|------------|-------|
| #7  | B.1 — Migration + region/sector weight model + `fx_history` + `exchange` | Backend | ✅ model + migrazioni (exchange, exposure, history) |
| #8  | B.2 — GICS/sector backfill + Yahoo v10 fetch-profile | Backend | ✅ chiusa da design change: `category_id`/`categories` rimossi (migrazione `000012`, PR #51); fetch-profile + `assets.sector` normalizzato coperti altrove; residuo (backfill `missing_sector`) accorpato a B.3 |
| #9  | B.3 — Country backfill (exposure via `assetProfile`) + sector backfill + ISO normalization + region mapping | Backend | ✅ implementata: country = domicilio emittente (fix cross-listing alla creazione), `geo.NormalizeCountry`/`RegionForCountry`, validazione ISO su create/update, `POST /assets/backfill-meta` che riempie E corregge i legacy (1AAPL.MI → US) — in PR #55 |
| #10 | B.4 — ETF weight editor (frontend): regions/sectors grid + "Try scrape" | Frontend | ✅ editor tabelle + pie chart sulla pagina asset; scrape differito a B.5 |
| #11 | B.5 — Python microservice: ETF metadata da JustETF | Python | ✅ implementata: `python-service` (FastAPI) con search ticker/ISIN + exposure paesi/regioni e settori; backend `POST /assets/{id}/fetch-etf-exposure` con auto-resolve ISIN — in PR #55 |
| #12 | B.6 — Endpoint /allocation/geography (weighted sum by region) | Backend | ✅ implementata: 8 macro-regioni + `Other` (poi allineate a Morningstar: 10 + `Other`), zero-filled — in PR #60 |
| #13 | B.7 — Endpoint /allocation/sector (weighted sum by GICS) | Backend | ✅ implementata: 11 settori GICS, zero-filled — in PR #60 |
| #14 | B.8 — Frontend GeographyChart + SectorChart + dashboard/portfolio widgets | Frontend | ✅ implementata (charts dashboard/portfolio, universo equity-only + coverage, pulsante JustETF + ISIN) — in PR #62 |
| #44 | B.9 — FX rate history + series engine per-date | Backend | ✅ implementata — in PR #61 |
| #45 | B.10 — Asset detail page (`/assets/[id]`) + exchange field | Full-stack | ✅ completa |
| #49 | B.11 — Asset class: colonna `asset_class` + auto-detect Yahoo + override manuale | Full-stack | ✅ implementata (da validare) |
| #50 | B.12 — Allocazione per classi: `GET /allocation/class` + donut | Full-stack | ✅ implementata (da validare) |
| #58 | B.13 — Per-country exposure storage: tabella `asset_country_weights`, 3 dimensioni | Backend + Frontend | ✅ implementata |
| #59 | B.14 — Morningstar come fonte esposizione: resolver custom (bootstrap Chromium headless per WAF+JWT), rotta backend, prefill frontend | Backend + Frontend | ✅ implementata |

**Ordine di implementazione**:
1. Data layer: B.1 → B.9 → B.3
2. Backend: B.3 (backfill) → B.5 → B.6/B.7 ✅  <small>(B.2 chiusa: superata in B.11/B.12, residuo in B.3)</small>
3. Frontend: B.10 → B.4 → B.8

### Completato in questa sessione (EPIC B, parte)
- **Pagina asset detail `/assets/[id]`** (B.10) su branch `feat/B.10-asset-detail`:
  - Caratteristiche editabili: Ticker, ISIN, Nome, Tipo, Valuta, Exchange (+ metadati `exchange`, `sector`, `industry` nel modello)
  - Card metriche + grafico prezzi ECharts con selettore periodo (1M/3M/1Y/MAX)
  - Tabelle distribuzione **geo** (8 macro-regioni, poi allineate a Morningstar in B.14 → 10 + `Other`) e **settore** (11 settori GICS) **modificabili**,
    con validazione somma=100% e grafici a ciambella affiancati
  - Pulsanti da menu hamburger: **"Aggiorna da Yahoo"** (meta: profile + sector weightings)
    e **"Backfill storico completo"** (storico prezzi completo da Yahoo)
  - Backend: migrazioni `000009` (asset meta), `000010` (exposure weights), `000011` (history_backfilled);
    package `geo` (macro-regioni + settori GICS + mappatura paese→regione);
    repository exposure; service `UpdateAsset`/`GetAssetQuote`/`FetchAssetProfile`/
    `GetAssetExposure`/`SaveAssetExposure`/`FetchAssetExposure`/`BackfillAssetHistory`;
    fix invalidazione cache (`bumpRev`) su sync dati
  - Endpoint: PATCH `/assets/{id}`, GET `/assets/{id}/quote`, POST `/assets/{id}/fetch-profile`,
    GET/PUT `/assets/{id}/exposure`, POST `/assets/{id}/fetch-exposure`,
    POST `/assets/{id}/backfill-history`

### B.3 + B.5 — Esposizione ETF da JustETF (branch `feat/B.3-B.5-meta-backfill`, PR #55 open)
- **B.3 (country backfill + ISO)** — `country` degli stock = domicilio emittente (fix cross-listing
  alla creazione, es. 1AAPL.MI → US), validazione ISO alpha-2 su create/update,
  `POST /assets/backfill-meta` che riempie e corregge i legacy. Package `geo` esteso:
  mappature paese→ISO (South Korea, Saudi Arabia, UAE, Thailand, Malaysia + EM comuni) e alias
  settori JustETF→GICS (Finance→Financials, Consumer Non-Cyclicals→Consumer Staples, etc.).
- **B.5 (python-service)** — microservizio `python-service/` (FastAPI + uvicorn + requests + bs4,
  immagine `python:3.12-slim`):
  - `GET /api/v1/etf/search?q={ticker|nome}` — ticker/ISIN via JustETF quick-search; rimuove il
    suffisso borsa (`.MI/.DE/.L/...`) come richiede il sito;
  - `GET /api/v1/etf/{isin}/exposure` — paesi/regioni e settori **completi** replicando gli AJAX
    Wicket "Show more" di JustETF (niente browser a runtime, Playwright usato solo per discovery);
  - `GET /api/v1/etf/{isin}/holdings` (stub) e `GET /healthz`; mai crash → errori `502` JSON.
  - **Morningstar** (`GET /api/v1/etf/{isin}/morningstar-exposure`) — stesso breakdown paesi/
    settori recuperato da **Morningstar** con un **resolver custom** (`app/morningstar.py`, senza
    mstarpy): il dominio `global.morningstar.com` usato da mstarpy è hard-blocked (403), quindi il
    resolver esegue un **bootstrap Chromium headless** (sotto Xvfb nel container, chromedriver
    esplicito per aarch64) che risolve il challenge AWS WAF su `www.morningstar.com` e ottiene il
    **Bearer JWT** da `/api/v2/stores/maas/token` (cache ~1h), poi le chiamate dati viaggiano via
    `requests` con token+cookie verso `www.us-api.morningstar.com/sal/sal-service/etf/...`
    (`portfolio/v2/sector/{sid}/data` → settori, `portfolio/regionalSectorIncludeCountries/{sid}/data`
    → paesi, `portfolio/regionalSector/{sid}/data` → regioni). ISIN→securityId via
    `www.morningstar.com/api/v2/search?q={isin}`. Parser difensivo: `fundPortfolio.countries`
    (name camelCase→nome leggibile, percent) e bucket `EQUITY`/`FIXEDINCOME` → settori GICS
    (scelta bucket col peso maggiore, chiavi non-GICS scartate), conversione 0-1→0-100,
    pesi paesi tenuti come riportati (Morningstar espone la lista paesi completa, 51 voci
    con molte a 0 e una quota residuale non esposta come paese; il residuo confluisce
    nella regione `Other / Not Classified` lato backend), ordinamento pesi desc.
- **Backend Go** — interfaccia `price.ETFFetcher` + `JustETFFetcher` (client del python-service),
  aggregazione `AggregateRegions`/`AggregateSectors` (paesi→macro-regioni + alias settori),
  nuova rotta **`POST /assets/{id}/fetch-etf-exposure`**: se l'asset non ha ISIN lo **auto-risolve
  dal ticker** (preferenza ticker esatto, poi similarità nome con `asset.Name`; il valore viene
  persistito sull'asset), poi scarica paesi/regioni + settori e li restituisce come
  **anteprima non persistente** (vedi «Redesign modale distribuzione geografica»: la
  persistenza avviene solo via `PUT /assets/{id}/exposure`). Config `PECULIUM_PYTHON_SERVICE_URL`;
  servizio `python-service` presente
  sia in `docker-compose.dev.yml` sia in `docker-compose.test.yml`.
- **Asset duplicato** — `POST /assets` con ticker già esistente ora risponde **409 Conflict** con
  messaggio chiaro e l'id dell'asset esistente (`asset_id` + `id`).
- **Verifica** — pytest (`python-service/tests`, 17 test), Go `build/vet/test green`; e2e su stack
  isolato `peculium-test`: XMME (14 paesi/13 settori, regioni sommano 100), VWCE e `SMEA.MI` senza
  ISIN auto-risolti correttamente (SMEA.MI → `IE00B4K48X80` iShares Core MSCI Europe).
  Test manuali via **`tests/api-test.http`** (estensione REST Client in VS Code) sullo stack test
  (porta 8081).

### Decisione ISIN
Verificato: **Yahoo non espone l'ISIN** (nessun campo in `assetProfile`/`fundProfile`/`price`).
`investing.com` è bloccato da Cloudflare (403) e Morningstar richiede API a pagamento
o scraping fragile token-gated. Decisione B.5: il campo `isin` resta **editabile a mano**
nella pagina asset, ma per gli ETF è ora **automatizzato** via JustETF: `POST /assets/{id}/fetch-etf-exposure`
(anche alla creazione, se l'ISIN è vuoto) risolve ticker→ISIN e lo persiste sull'asset.

Da B.14 l'auto-resolve è **market-aware**: quando il ticker porta un suffisso di
mercato riconosciuto (`.MI`, `.DE`, `.L`, `.SW`, ...), l'ISIN viene risolto via
**Morningstar** cercando la quotazione su quell'exchange (mappa suffisso→codice
Morningstar in `python-service/app/morningstar.py`); senza suffisso si usa
JustETF. È necessario perché lo stesso ticker può avere **ISIN diversi a seconda
del listino** (es. `EQQQ` = `IE0032077012` sul listino tedesco vs altre quotazioni),
e Morningstar permette di cercare sul mercato esatto.

### Asset class + allocazione per classi (B.11, B.12)
- **Rimossa la classificazione single-category** (`category_id` → tabella `categories`): inadatta agli ETF
  multi-settore; la distribuzione settoriale è già coperta da `asset_sector_weights`/`assets.sector`.
  Migrazione `000012_remove_category` (drop colonna + tabella).
- **Nuova `assets.asset_class`** (migrazione `000013`, check enum): `equity`, `bond`, `commodity`,
  `currency`, `crypto`, `real_estate`, `mixed`, `other`. Etichetta primaria esclusiva; per i
  multi-classe si usa `mixed`.
- **Auto-detect da Yahoo**: l'endpoint non espone più `assetClass` (modulo `quote`
  inesistente in quoteSummary v10), quindi la classe viene derivata da
  `geo.ClassifyAssetClass` (categoria fondo Morningstar `defaultKeyStatistics.category`/
  `fundProfile.categoryName` + euristica sul nome; BND→bond, GLD→commodity, VWCE.DE→equity),
  con default dal tipo (`stock`→equity, `bond`→bond, `commodity`→commodity, `cash`→currency,
  `crypto`→crypto). Il recupero della classe è **accorpato al recupero info asset**
  (`GET /assets/meta`, usato alla creazione e da "Aggiorna da Yahoo") e **non** alla lettura
  dei settori (`fetch-exposure` non la tocca più); **override manuale** nell'editor asset
  che vince sempre (aggiornato solo se vuota o `other`).
- Metrica data-quality: `missing_category` → **`missing_sector`** (asset con `sector` non valorizzato).
- **Endpoint** `GET /portfolios/{id}/allocation/class` (somma pesata sul valore in valuta portafoglio)
  e widget donut "Allocazione per classi" nella pagina portfolio.

### B.6 + B.7 — Allocazione geo/settoriale a livello portafoglio (branch `feat/B.6-B.7-allocation`, PR #60)
- **Endpoint** `GET /portfolios/{id}/allocation/geography` (somma pesata per **macro-regione**) e
  `GET /portfolios/{id}/allocation/sector` (somma pesata per **settore GICS**): stessi principi di
  `/allocation/class` (pesi per-asset da `asset_region_weights`/`asset_sector_weights` × valore in
  valuta portafoglio, conversioni FX incluse).
- Risultato con **10 macro-regioni (+ `Other`, allineate a Morningstar)** e **11 settori GICS (+ `Other`)**, zero-filled per
  evitare buchi: bucket sempre in ordine fisso, percentuali che sommano a 100. `Other` raccoglie
  pesi non mappabili (asset country/region o settore non riconosciuti); per i settori un peso
  complessivo 0 finisce tutto in `Other`.
- Porto vuoto o totale zero → righe a zero senza bucket `Other` (niente denominatori artificiali).
- **Verifica**: e2e-unit in `backend/internal/service/service_test.go` (ETF completo, fallback stock
  su domicilio, conversione FX, portafoglio vuoto, bucket `Other`); test manuale di smoke su stack
  isolato `peculium-test` con **`tests/test-epic-b.sh`** (20 check PASS) usando prezzi seminati da
  **`tests/seed-prices.sql`** (Yahoo è disabilitato sullo stack test, quindi i prezzi si scrivono
  solo via SQL) e la raccolta **`tests/api-test.http`** estesa.

### B.8 + B.9 — Chart dashboard/portfolio e FX history (branch `feat/B.8-allocation-charts` PR #62, `feat/B.9-fx-history` PR #61)
- **B.8 (issue #14, PR #62)** — widget di allocazione geo/settoriale:
  `GeographyChart` + `SectorChart` (donut a 12 colori + tabella righe complete)
  sulla pagina portafoglio e nella card "Allocazione complessiva" della
  dashboard, alimentati da `GET /portfolios/{id}/allocation/geography`,
  `/allocation/sector` e `GET /dashboard/allocation` (aggregato USD).
- **Universo equity-only (follow-up B.8)** — le allocazioni geo/settoriali
  coprono solo l'equity (`exposureEligible`: stock sempre; etf/mutual_fund solo
  con `asset_class` `equity`/`real_estate`). Bond, crypto, commodity, valute e
  fondi non classificati sono **esclusi** (mai in `Other`); geography/sector/
  dashboard espongono `covered_value`/`excluded_value` e i grafici mostrano la
  nota di copertura.
- **Editor asset (follow-up B.8)** — pulsante **"Carica da JustETF"**
  (`POST /assets/{id}/fetch-etf-exposure`) che scarica regioni+settori come
  **anteprima** (non persiste; salvataggio solo col PUT) e sincronizza l'**ISIN**
  risolto nel form; campo ISIN anche nel form di
  creazione asset; banner "solo asset azionari" quando l'asset non è
  azionabile.
- **B.9 (issue #44, PR #61)** — storico tassi di cambio per-data (`fx_history`)
  integrato nel series engine per conversioni storiche per-date.
- **Verifica** — Go build/vet/test green; smoke su stack isolato
  `peculium-test` con `tests/test-epic-b.sh` (20 PASS; gli ETF sono creati con
  `asset_class: equity` per rispettare l'universo strict) + esclusione bond
  verificata end-to-end (covered=2600, excluded=10000, pesi somma 100).

### B.13 + B.14 — Exposure 3 dimensioni + Morningstar (issues #58, #59)
- **B.13 — Per-country exposure storage** (#58): nuova tabella
  `asset_country_weights (asset_id, country, weight)` (migrazione
  `000016_country_weights`). L'esposizione ora ha 3 dimensioni:
`countries` (pesi per paese ISO-3166 alpha-2), `regions` (macro-regioni)
    e `sectors` (settori GICS). Il package `geo` espone `var Countries` (~89
    codici ISO canonici). Il repository `ExposureRepository` aggiunge
    `FindCountries`/`ReplaceCountries`/`FindCountriesByAssets`.
    - **Tassonomia regioni allineata a Morningstar**: 10 macro-regioni +
      `Other / Not Classified`. Rispetto alla vecchia lista a 8 regioni:
      **United Kingdom** (GB) separata da Europe Developed; **Japan** (JP)
      separata da Asia Developed; **Australasia** (AU, NZ) separata; **TW/KR**
      spostate da Asia Emerging a **Asia Developed**; **SI** spostata da
      Europe Emerging a Europe Developed (coerente con Morningstar).
      Le chiavi Morningstar (`northAmerica`, `unitedKingdom`, `japan`,
      `australasia`, ...) mappano 1:1 sui nomi canonici.
  - `GET /assets/{id}/exposure` restituisce countries zero-filled sulla lista
    canonica completa (più regions e sectors).
  - `PUT /assets/{id}/exposure` accetta `countries?` (dimensioni indipendenti),
    tenere solo codici ISO canonici; la somma dei paesi **non deve** essere
    esattamente 100 (i pesi paesi sono informativi): quando i countries sono
    forniti il backend ricalcola e persiste le regions dalla mappatura
    paese→regione, imputando il residuo (100 − somma) nella regione
    `Other / Not Classified` (coerenza automatica a somma 100 per le regioni).
    _(Auto-derivation al save poi **rimossa** dal redesign modale: vedi
    «Redesign modale distribuzione geografica (paesi-first)」.)_
  - JustETF ora espone i countries raw (normalizzati a codici ISO) invece di
    scartarli; il backend li include nella preview e li archivia in
    `asset_country_weights` solo al salvataggio (`PUT`).
- **B.14 — Morningstar come fonte esposizione** (#59):
  - **python-service**: nuovo endpoint `GET /api/v1/etf/{isin}/morningstar-exposure`
    (modulo `python-service/app/morningstar.py`, **resolver custom senza mstarpy**).
    Il dominio `global.morningstar.com` delle SAL API informali usate da mstarpy è
    hard-blocked (403 "Request blocked") dall'IP di questo ambiente; il resolver
    esegue un **bootstrap Chromium headless** (installato nell'immagine via apt:
    `chromium`, `chromium-driver`, `xvfb`, `fonts-liberation`; chromedriver passato
    esplicitamente perché Selenium Manager non supporta `linux/aarch64`). Il browser
    carica `www.morningstar.com` finché il challenge AWS WAF non si risolve, legge i
    cookie di sessione e il **Bearer JWT** da `/api/v2/stores/maas/token` (cache in
    memoria, scadenza ~1h da `exp` del JWT). Le chiamate dati avvengono poi via
    `requests` con Bearer+cookie verso
    `www.us-api.morningstar.com/sal/sal-service/etf/...`:
    `portfolio/v2/sector/{sid}/data` (settori), `portfolio/regionalSectorIncludeCountries/{sid}/data`
    (paesi), `portfolio/regionalSector/{sid}/data` (regioni ufficiali),
    ISIN→securityId via `www.morningstar.com/api/v2/search?q={isin}`.
    Parser: paesi camelCase→nomi leggibili, settori dal bucket col peso maggiore
    (EQUITY per fondi azionari, FIXEDINCOME per obbligazionari; chiavi non-GICS come
    `cashAndEquivalents`/`government` scartate), regioni mappate sui nomi canonici
    (`_MORNINGSTAR_REGION_NAMES`). I pesi paesi sono tenuti come
    riportati (Morningstar espone la lista paesi completa, 51 voci, ma con una
    quota residuale "Other" non esposta come paese → la somma è ~95%): il residuo
    confluisce nella regione `Other / Not Classified` lato backend. L'auto-resolve
    ticker→ISIN è **market-aware** (mappa suffisso→exchange, es. `.MI`→`XMIL`):
    `search_etf` usa Morningstar per i ticker con suffisso riconosciuto e JustETF
    per i ticker senza suffisso. Variabili env opzionali
    `MORNINGSTAR_BEARER`/`MORNINGSTAR_COOKIES` per bypassare il browser nei test.
  - **Backend Go**: interfaccia `ETFFetcher` estesa con `FetchMorningstarExposure`;
    nuovo metodo `Service.FetchMorningstarExposure`; nuovo handler + rotta
    `POST /assets/{id}/fetch-morningstar-exposure` (solo ETF, auto-risolve ISIN).
    Quando il python-service restituisce le regioni ufficiali Morningstar, queste
    vengono usate direttamente (ordine canonico); altrimenti si ricade sulla
    derivazione da paesi. Nuovo endpoint **`POST /assets/{id}/exposure/derive`**
    (`{countries}` → `{regions}`, senza persistenza): usa la stessa
    `AggregateRegions` con il residuo in `Other / Not Classified`.
  - **Frontend**: la pagina asset detail è riorganizzata in **due card** —
    "Distribuzione geografica" (top 15 paesi a barre orizzontali + donut
    regioni) e "Distribuzione settoriale" (donut settori) — e l'editing è
    separato in **due modali**: `ExposureGeoModal` (regioni + paesi) e
    `ExposureSectorModal` (settori, prefill JustETF/Yahoo). I pulsanti di prefill
    seguono la fonte dei dati:
    - box **paesi**: **"Prefill da JustETF"** (JustETF scarica i paesi) e
      **"Prefill da Morningstar"** (paesi [+ settori]);
    - box **regioni**: **"Calcola da paesi"** (`POST /assets/{id}/exposure/derive`,
      ricalcola le regioni dai paesi correnti senza salvare) e
      **"Prefill da Morningstar"** (regioni ufficiali Morningstar);
    - box **settori**: "Prefill da JustETF" e "Prefill da Yahoo".
    Modifica add/remove
    dei paesi dalla lista canonica supportata; display usa nomi paese amichevoli
    da `frontend/src/lib/countryNames.ts`.
- **Verifica**: Go build/vet/test green; python-service pytest (51 test) green;
  `svelte-check`/eslint clean; e2e su stack isolato `peculium-test` con
  `make test-e2e` (18 PASS, 0 FAIL). Morningstar verificato end-to-end sullo stack
  test: `POST /assets/{id}/fetch-morningstar-exposure` per SMEA restituisce paesi
  canonici (zero-fill, somma raw ~95%) + settori GICS (somma 100) + **regioni
  ufficiali** (es. United Kingdom 21.27, Europe Developed ex-UK 75.85 per un ETF
  europeo); `POST /assets/{id}/exposure/derive` verifica la nuova tassonomia
  (GB→United Kingdom, JP→Japan, TW+KR→Asia Developed, residuo→Other).
- **Resilienza WAF Morningstar (fix post-#59)**: la WAF può revocare la
  sessione cacheata prima della scadenza del JWT e rispondere con una pagina
  HTML di challenge (JSON non decodificabile → 502 opaco "upstream fetch
  failed"). Ora l'intero flow (ISIN→securityId + le tre chiamate SAL + parsing)
  viene **ritentato una volta** con credenziali fresche:
  `_invalidate_bootstrap_cache`/`_retry_on_waf_challenge` in
  `python-service/app/morningstar.py`, con nuovo errore `MorningstarWafError`.
  Se la challenge persiste, `morningstar-exposure` risponde 502 con messaggio
  esplicito ("Morningstar WAF challenge could not be passed for ISIN …; try
  again in a few seconds") e l'auto-resolve ticker→ISIN degrada a JustETF.
  Gli altri errori di trasporto restano mappati come prima (nessun retry).
  pytest aggiornato a 57 test, tutti verdi.
- **Resilienza bootstrap Chrome (fix post-#59, oltre al retry WAF)**: bootstrap
  falliti lasciano processi Chromium/chromedriver/crashpad orfani che bloccano
  ogni sessione successiva ("session not created: Chrome instance exited",
  zombie `<defunct>` persistenti nel container). Il ramo browser di
  `_session_credentials` ora ritenta **una sola volta** dopo pulizia best-effort
  dei processi orfani (`_kill_stale_browsers`, solo `pkill` — mai glob su /tmp);
  al secondo crash risponde 502 con messaggio chiaro ("Chrome headless could not
  start in the sandbox; try again in a few seconds"). pytest aggiornato a 62
  test, tutti verdi.
- **Fallback multi-listing Morningstar (fix #76)**: il resolver usava solo il
  **primo** securityID restituito da `/api/v2/search`; se quel listing non ha
  dati SAL (risposta `206`/`Can't get SecurityInfo`, es. la quotazione XAMS di
  `IE00B5L8K969`/CSEMAS) il JSON non era decodificabile e il caso veniva
  scambiato per challenge WAF → re-bootstrap del browser (con processi Chromium
  orfani lasciati indietro) e 502 fuorviante "WAF challenge could not be
  passed", nonostante gli altri listing dello stesso ISIN avessero dati validi.
  Ora `_search_security_ids` restituisce **tutti** i securityID dei fondi che
  matchano l'ISIN (in ordine di ricerca, deduplicati) e `_fetch_exposure_once`
  li prova in sequenza fermandosi al primo con paesi validi; il nuovo errore
  non ritentabile `MorningstarSecurityUnavailable` fa passare al candidato
  successivo **senza** re-bootstrap, e se nessun listing ha dati l'errore è un
  chiaro `MorningstarDataError` ("No SAL data found for ISIN … on any of N
  Morningstar listings"), mai `MorningstarWafError`. Solo una pagina HTML di
  challenge (JSON non decodificabile senza marcatore missing-data) resta
  soggetta al retry con sessione fresca. pytest aggiornato a 70 test, tutti
  verdi.
- **Cache esposizione provider (post-B.14)**: `FetchETFExposure` e
  `FetchMorningstarExposure` cachano il payload grezzo del provider in Redis
  (chiave `pc:lookup:exposure:<source>:<ISIN>`, TTL `PECULIUM_EXPOSURE_CACHE_TTL`
  default 7 giorni): la prima richiesta su un ISIN esegue il fetch pesante, le
  successive rispondono dalla cache; `?refresh=1` forza il refetch e riscrive
  la cache (risultati senza paesi mai cachati; Yahoo `fetch-exposure`
  invariato). 4 nuovi test service (`TestFetch*Cache*`/`EmptyResultNotCached`).

### Redesign modale distribuzione geografica (paesi-first)
- **Frontend** (`ExposureGeoModal`, `ExposurePie`, `ProvenanceBadge`): la modale
  passa a **due colonne** — **Paesi a sinistra**, **Regioni a destra** (impilati
  paesi-primo su mobile).
  - **Paesi**: lista a **partenza vuota** (non più 89 righe zero-filled); ogni
    riga è `codice · nome · barra orizzontale · input peso · cestino`, ordinata
    per peso desc (riordino su add/remove/blur, mai mentre digiti — la barra si
    anima live); select + "Aggiungi" per inserire codici canonici; footer con
    "Totale X%" + meter; **save disabilitato se somma > 100** (sotto 100 ok);
    **nessun donut**.
  - **Regioni**: **tabella fissa delle 10 regioni canoniche** (niente
    add/remove; riga "Other / Not Classified" **rimossa** — filtrata alla
    pagina, non entra mai in `regionsEdit`); **donut aperto** con la nuova prop
    `complete={false}` su `ExposurePie` (una fetta residua trasparente mantiene
    veritieri gli angoli quando la somma < 100, invece della fetta grigia
    Other); **save disabilitato se somma > 100** (sostituisce la vecchia regola
    100 ± 0,5).
  - **Badge di provenienza** (`ProvenanceBadge`): pillola con puntino colorato
    per fonte — `manuale`/`da JustETF`/`da Morningstar`(` regioni ufficiali`)/
    `calcolato dai paesi`/`da JustETF via paesi`; prefill/derive impostano il
    badge, ogni modifica manuale lo riporta a "manuale". Stato session-scoped
    (niente persistenza), badge nascosto a reload.
  - Coerenza: anche la **card geografica** della pagina usa `complete={false}`
    e filtra Other da donut/legenda regioni.
  - **Salvataggio paesi**: il backend **non ricalcola più le regioni** al save
    dei `countries` (auto-derivation rimossa); le regioni memorizzate restano
    invariate (badge di provenienza intatto) e si aggiornano solo manualmente
    con «Calcola da paesi».
- **Backend**: `SaveAssetExposure` accetta ora **regioni con somma ≤ 100** e,
  quando il client non invia Other, **inieietta il residuo in
  `Other / Not Classified`** prima di persistere (invariante «regioni=100»
  preservata per l'aggregazione portafoglio); **countries rifiutano somma
  > 100.5** (nessun minimo, sotto 100 ok); settori invariati (100 ± 0,5).
  Auto-derivation al save dei paesi **rimossa**: ogni dimensione del body `PUT`
  è salvata indipendentemente (logica estratta in `saveExposureDimensions`) e
  le regioni si ricalcolano solo via `POST /assets/{id}/exposure/derive` o
  salvando regioni esplicite.
  `derive` invariato (restituisce Other; filtra il frontend).
- **Prefill/fetch = anteprime non persistenti**: i tre endpoint di fetch
  (`POST /assets/{id}/fetch-exposure`, `/fetch-etf-exposure`,
  `/fetch-morningstar-exposure`) **non salvano più** le dimensioni di
  esposizione (countries/regions/sectors) né i campi profilo dell'asset:
  restituiscono una **preview canonica** costruita dai dati del provider
  (per `fetch-exposure` le dimensioni geo memorizzate restano lette dallo
  stato attuale). L'unica eccezione concordata resta l'**ISIN auto-risolto**,
  che continua a essere persistito sulla tabella `assets` (con `bumpRev`).
  La persistenza avviene **solo** con `PUT /assets/{id}/exposure` (pulsanti
  Salva); `POST /assets/{id}/exposure/derive` era e resta non persistente.
  Smoke e2e aggiornato: `tests/test-epic-b.sh` FASE 4 fa ora un PUT esplicito
  dei valori in preview dopo il fetch. Test:
  `TestFetchETFExposure_PreviewDoesNotPersist` / `...PersistsOnlyResolvedISIN`,
  `TestFetchMorningstarExposure_PreviewDoesNotPersist` /
  `...PersistsOnlyResolvedISIN`, `TestFetchAssetExposure_PreviewDoesNotPersist`.
- **Verifica**: Go build/vet/test green (nuovi `TestPrepareRegions`/
  `TestPrepareCountries`/`TestValidateExposureWeights`/
  `TestSaveExposureDimensions_*`); pytest 51;
  `svelte-check`/eslint/build clean; `make test-e2e` 18 PASS 0 FAIL. Test
  manuale PUT: regions somma 95 → 200 con Other=5 persistito (somma 100);
  regions 103 → 400; countries 120 → 400; countries 95 → 200 con regioni
  memorizzate invariate (nessuna riscrittura).

### Provenienza persistita dell'esposizione (follow-up B.13/B.14)
- **Backend**: nuova tabella `asset_exposure_provenance (asset_id, dimension, source, updated_at)`
  (migrazione `000017`). `PUT /assets/{id}/exposure` accetta `countries_source` /
  `regions_source` / `sectors_source` (default `manual`) e registra la provenienza di
  ogni dimensione salvata; `GET`/`PUT /assets/{id}/exposure` rispondono la mappa
  `provenance` (`omitempty`) così i badge "da Morningstar (2026-09-05)" dell'UI
  sopravvivono al reload. Gli fetch/prefill restano anteprime senza provenienza.
  3 nuovi test service + fake aggiornato; Go build/vet/test green.

### Altri EPIC Fase 2
- EPIC G.7 (#53) — Asset con ticker non-Yahoo: no richiesta prezzo e no errori
  - Campo `price_source` su `assets` (`yahoo`/`manual`/`none`, default `yahoo`):
    migrazione `000015`, modello, repo, validazione service, handler PATCH.
  - Il worker e `RefreshStale` filtrano solo asset `yahoo`; gli asset `manual`/
    `none` sono saltati del tutto (nessuna chiamata Yahoo, nessun errore di health).
  - Implementata: backend (model, repo, service, handler, price fetcher, docs).
- F.9 (#52) — Chart storico asset: zoom in-place + selettore YTD
  - `PriceChart` carica sempre tutto lo storico; i selettori 1M/3M/1Y/YTD/MAX
    fanno zoom in-place (coppia `start`/`end` percentuali) senza ricaricare dati.
  - Uno zoom/spostamento manuale deseleziona il pulsante attivo e preserva la vista.
- F.10 (#64) — Pagina asset: grafici + modale di modifica esposizione
  - **Ristrutturata dopo B.13/B.14**: l'unica card "Distribuzione" con una sola
    `ExposureModal` è diventata **due card** — "Distribuzione geografica" (paesi
    a barre + donut regioni) e "Distribuzione settoriale" (donut settori) — con
    **due modali separate**: `ExposureGeoModal.svelte` (regioni + paesi) e
    `ExposureSectorModal.svelte` (settori). I prefill vivono solo nelle modali e
    popolano una dimensione alla volta: JustETF → regioni; Morningstar → paesi
    (+ settori); Yahoo → settori.
- Splits sul chart asset — nuovi `GET /assets/{id}/splits` (service `AssetSplits`,
  handler) e `markLine` viola etichettati con il rapporto sul `PriceChart`,
  come nel `PositionChart` del portafoglio.
- EPIC C (#39) — Metric di rischio: Sharpe, max drawdown, volatilità, regressione, Monte Carlo
- EPIC E (#38) — Pagine e componenti dominio (rebuilt dashboard, tabelle, modali)

### EPIC D — Design system & dark mode (#37) — ✅ Completata
Branch unico `feat/D-design-system`, 5 commit:
- `feat(tokens)` — token semantici (CSS custom properties HSL in `app.css` mappate in `tailwind.config.js` con `<alpha-value>`), store tema a 3 modalità (light/dark/system, **default dark**), script anti-FOUC in `app.html`, `lib/chartTheme.ts` (temi ECharts `peculium-light`/`peculium-dark`) e `lib/chartPalette.ts` (palette risolta a runtime).
- `feat(ui)` — sweep dei colori hardcoded (~333 classi palette + hex) verso i token su tutte le pagine/componenti; grafici dark-aware.
- `feat(ui)` — primitive in `src/lib/components/ui/` (Button, Input, Field, Select, Card, Badge, Modal, ConfirmDialog, Spinner, Skeleton, EmptyState, Table, SegmentedControl, StatCard) + refactor Toaster/ProvenanceBadge; i 4 `confirm()` nativi sostituiti da `ConfirmDialog`.
- `feat(shell)` — `AppShell` responsive: sidebar collassabile (stato persistito in localStorage), header sticky, UserMenu, ThemeToggle a 3 modalità, MobileDrawer accessibile; `Layout.svelte` rimosso.
- `feat(theme)` — dark mode di default + rimozione della pagina dev `/settings/theme-tokens`.
- Fix: label dei donut leggibili in dark (le label ECharts non ereditavano il `textStyle` del tema → fill scuro + bordo bianco).
- Documentazione: `docs/FRONTEND-GUIDE.en/it.md` (styling/tema, chart, layout) e `docs/RELEASE-NOTES.en/it.md`.

### Ondata 0 — Fix & infrastruttura (13 Set 2026)

- H.1 (#32) — GitHub Actions: `.github/workflows/ci.yml` con due job (backend: `go build` + `go vet` + `go test`; frontend: `npm ci` + `npm run check` + `npm run lint`) su push e pull request verso `develop`/`main`, più un job `compose` che valida i tre file Compose. Release automation in `.github/workflows/publish-images.yml`: buildx multi-arch (`amd64` + `arm64`) delle cinque immagini su GHCR (`peculium-*`) e GitHub Release creata dai `docs/RELEASE-NOTES.*` sui tag `v*`. Fase 0 completa.
- #57 (fase A) — Governance utenti: colonna `users.status` (`active`/`pending`/`disabled`) e `server_settings.auto_approve_registrations` (default ON); il **primo utente** registrato diventa `admin/active`; i successivi sono `active` o `pending` secondo l'auto-approvazione; lo stato è verificato **a ogni richiesta** (disabilitare un utente invalida i token esistenti); area admin (elenco utenti, approva/disabilita, cambio ruolo, reset password, toggle auto-approve) riservata ai ruoli `owner`/`admin`. Backup/restore (#56) e permessi per-utente sugli asset (#141) restano separati.
- H.4 (#46) — Price sync health: campo `has_data` nel summary; la card Success Rate mostra **N/A** (niente più `NaN%`) quando non ci sono eventi nel periodo; `formatRate` robusto a null/undefined/NaN.
- H.5 (#47) — Logging API: `FetchIssue` con `request_type` (`chart`/`spark`/`search`/`fx`) e `asset_id`; `HealthEvent.AssetID` popolato sugli eventi per-asset; messaggi di successo con l'elenco dei ticker; registrazione degli eventi `search` (lookup) prima assenti.
- E.9 (#71) — Le allocazioni del portafoglio (classi, geo, settori) vengono rifetchate dopo create/update/delete di una transazione, senza reload.
- H.2 (#33, parziale) — Nuovi unit test per le allocazioni backend: `GetPortfolioAllocation` (multi-valuta, FX mancante, skip qty/prezzo) e `GetPortfolioClassAllocation` (raggruppamento/ordinamento, skip FX), più allocazione settoriale ETF (`backend/internal/service/allocation_test.go`).
- H.9 (#90) — Asset non-Yahoo (`manual`/`none`) non più interrogati per history/split: filtro `price_source` in `GetPortfolioHistory`, `SyncAssetData`/`syncAssets` e `BackfillAssetHistory` (no-op). Niente più eventi health `history_fetch`/`split_fetch` per questi asset.
- H.11 (#93) — Health summary calcolato dal DB (`health_events`) su finestra selezionabile **Today / Last 24h / Last 100 events** (rimossi i contatori Redis orari); `period` e `has_data` nella risposta, selettore nella pagina Health.
- H.10 (#91, parziale) — Paginazione della lista eventi Health (`limit`/`offset`, `events_total`; UI 50/pagina con Previous/Next). Restano: copertura di tutte le chiamate esterne (meta/profilo, JustETF/Morningstar, successi) e retention di `health_events`.

### Ondata 1 — EPIC E ✅ completata

- E.8 (#54) — Chart storico portafogli in dashboard: asse `time` con i punti di ogni portafoglio (niente più unione di date con `null` che spezzava le linee), `connectNulls` + `sampling: lttb`, `dataZoom` inside/slider come `PositionChart`. `PortfolioLineChart` non prende più la prop `data` (usato solo dalla dashboard).
- E.3 (#20) — Componenti di dominio riusabili: `AssetSearchAutocomplete`, `CurrencySelect`, `CreateAssetModal`, `CreatePortfolioModal`, `ImportPortfolioModal`. Pagine assets/portfolios ripulite dai form inline (Badge per il tipo, `ui/Button`/`ui/Card`/`ui/EmptyState`), login ridisegnato (logo, `SegmentedControl` Sign in/Register, `Field`/`Input`, validazione inline, conferma password in registrazione).
- E.1 (#18) — Dashboard ridisegnata: KPI `ui/StatCard` per valuta, `AllocationDonut` per portafoglio, portafogli come card cliccabili, `PositionTable` condiviso nell'accordion, `EmptyState`/`Spinner`. Backend: `finished_at` in `RefreshReport`, mostrato nell'header come "Prices updated".
- E.2 (#19) — Dettaglio portafoglio: `AssetCombobox`, `TransactionTable`, `AddTransactionModal` (form + validazione inline + totale live + delete con ConfirmDialog). KPI con `StatCard`, posizioni con `PositionTable` condiviso (con colonna Price), azioni in header sticky; refetch post-mutation (E.9) preservato.
- E.4 (#21) — Settings divisa in tab via subroute (`/settings` Profile, `/settings/password`, `/settings/currencies`, `/settings/health`) con `SettingsTabs`; cambio password con validazione inline e mappatura errori `401`/`400` sui campi. Sweep a11y/numerico: `ui/Th` con `scope="col"`, tabelle assets/health/valute con primitive `ui/Table`, `aria-label` sui bottoni icona, `tabular-nums`/allineamento a destra sui numeri.

### I.1 — Valuta base utente + aggregazione FX (branch `feat/I.1-base-currency`)
Enabler di EPIC I (sblocca I.2–I.5). La dashboard e le sue viste aggregate esprimono i
totali nella **valuta base dell'utente** (default `EUR`).
- **Backend**:
  - Migrazione `000018_users_base_currency`: colonna `users.base_currency` (default `EUR`).
  - `model.User`/`repository/user.go`: `base_currency` in SELECT/RETURNING/Update.
  - `UpdateProfile` accetta `base_currency` (opzionale, validata contro la whitelist
    `supported_currencies` enabled via `EnabledByCodes`; vuota = mantiene il valore).
  - `PATCH /users/me` accetta il campo `base_currency` (400 se non è una valuta abilitata).
  - `GET /dashboard` risponde ora `base_currency` + `summary` aggregato in valuta base
    (`invested`/`value`/`gain_loss`/`gain_loss_pct`/`realized`/`fx_missing_count`/
    `fx_missing_value`); le serie `history` sono convertite per-data in valuta base
    (`series.LoadDateRates` + `dateRates.Factor`, punti senza FX scartati). `by_currency`,
    `portfolios`, `assets` restano invariati (serviranno a I.2/I.5).
  - `GET /dashboard/allocation` ora esprime geo/settori nella valuta base (prima USD).
  - `series.LoadRates` esteso (variadic `extra ...string`) per includere le valute dei
    portafogli oltre a quelle degli asset + base + USD.
  - FX mancante gestito esplicitamente (escluso dai totali + conteggiato), coerente con
    `GetPortfolioSummary` (EPIC A).
- **Frontend**:
  - `api.ts`: `User.base_currency`, `Dashboard.base_currency`/`summary` (`DashboardSummary`),
    `authApi.updateProfile` con `base_currency?`.
  - `auth.svelte.ts`: `updateProfile(name, email, baseCurrency?)`.
  - Settings → Profile: `CurrencySelect` "Base currency" (popolata da
    `settingsApi.listCurrencies()`), salvata con `PATCH /users/me`.
  - Dashboard: KPI primari in valuta base (`summary`), ripartizione per-valuta mostrata
    solo quando ci sono più valute; donut "Allocation by portfolio" in valuta base.
- **Verifica**: Go build/vet/test green (7 nuovi test service: `UpdateProfile` base currency,
  `GetDashboard` summary multi-valuta + FX mancante + conversione history, `GetDashboardAllocation`
  in valuta base); `svelte-check`/eslint clean.

### I.2 — Dashboard attivo vs chiuso (branch `feat/I.1-base-currency`)
Il riepilogo dashboard (patrimonio e per-portafoglio) separa ora le quote di investimento
**attive** da quelle **chiuse**, in valuta base a livello di patrimonio.
- **Backend**:
  - `position.State`: aggiunti i cumulati dei lotti chiusi `ClosedCost`/`ClosedCostCCY`
    (costo AVCO dei venduto), `Proceeds`/`ProceedsCCY` (incasso netto) e
    `Dividends`/`DividendsCCY`; `TxSell` e `TxDividend` li accumulano senza toccare la
    logica `Realized` esistente.
  - `model.Holding`: propagati i sei campi da `HoldingsDetailed`.
  - `model`: nuovi `ActiveBreakdown` (`invested`/`value`/`gain_loss`/`gain_loss_pct`) e
    `ClosedBreakdown` (`invested`/`proceeds`/`realized` = proceeds − invested, `dividends`
    separate dal capitale). `DashboardSummary` e `PortfolioPerformanceSummary` sostituiscono
    i campi flat I.1 con gli oggetti annidati `active`/`closed` (breaking per la UI, frontend
    da adeguare); `by_currency` e `assets` invariati.
  - `GetDashboard`: granularità per porzione di lotto (quantità vendute → chiuso, quantità
    residue → attivo); conversione per-importo con gli stessi criteri FX-missing di I.1
    (importo non convertibile escluso dai totali e contato in `fx_missing_count`/
    `fx_missing_value`, solo importi nonnulli).
- **Verifica**: Go build/vet/test green; nuovo `position_test.go` (venduto totale/parziale,
  dividendi separati, `Walk`) + test service `TestGetDashboard_ActiveClosedBreakdown` e
  `TestGetDashboard_SummaryInBaseCurrency` aggiornati alla forma annidata.
- **Documentazione**: `docs/BACKEND-GUIDE.en/it.md` (cap. 7/8, paragrafo valuta base) e
  `docs/RELEASE-NOTES.en/it.md`.

### I.3 — Dashboard: grafico performance + capitale (branch `feat/I.1-base-currency`)
- **Backend**: nuovo `GET /dashboard/performance?granularity=month|year` → bucket
  `{period, return, twr, invested, value}`. Rendimento **time-weighted (TWR)** puro: `V(d)` =
  valore di mercato (asset prezzati + bond non quotati portati al costo), flussi esterni
  `buy/sell/dividend/fee`, rendimento giornaliero composto; `invested` = capitale netto,
  `value` = market value. Rimosso il vecchio `history` dal dashboard.
- **Frontend**: card **Performance** (barre `return` % + linea TWR cumulata) e card **Capital
  invested** (`invested` vs `value`), toggle Mensile/Annuale (`PerformanceChart`,
  `CapitalChart`); rimosso `PortfolioLineChart`. Rimosse anche le righe per-valuta
  (`by_currency`) dalla dashboard.
- **Verifica**: `service_test.go` (TWR, liquidazione/riapertura, bond al costo, dividendi,
  multi-valuta) + `svelte-check`/lint.

### I.4 — Allocazione dashboard estesa (branch `feat/I.1-base-currency`)
- **Backend**: `GET /dashboard/allocation` esteso con `classes` (per asset class) e `countries`
  (per paese, equity-only, non-zero, descending), in valuta base.
- **Frontend**: card "Allocazione complessiva" → classi (donut, `ClassDonut`) + regioni/settori/
  paesi a **barre orizzontali** (`ExposureBarChart`); paesi con nome completo e ~10 righe
  visibili + scroll.

### I.5 — Dashboard: tabella asset investiti consolidata (branch `feat/I.1-base-currency`)
- **Backend**: `GET /dashboard` espone `invested_assets` (per asset, aggregato su tutti i
  portafogli, valuta base, sole posizioni aperte, ordinato per valore; asset senza prezzo al
  costo con `has_price=false`).
- **Frontend**: card **Invested assets** che sostituisce gli accordion per-portafoglio.

### I.6 — Dettaglio portafoglio: KPI allineati alla dashboard (branch `feat/I.1-base-currency`)
- **Backend**: `GET /portfolios/{id}/summary` espone `active`/`closed` (stessa forma della
  dashboard, in valuta portafoglio).
- **Frontend**: card condivisa **`InvestmentsTable`** (estratto) usata da dashboard e dettaglio;
  la vecchia riga KPI Value/Realized/Open G/L/Assets è sostituita dalla card Active/Closed +
  riga asset.

### I.7 — Dettaglio portafoglio: allocazioni come la dashboard (branch `feat/I.1-base-currency`)
- **Backend**: `GET /portfolios/{id}/allocation/geography` esteso con `countries` (equity-only,
  descending, valuta portafoglio).
- **Frontend**: sezione allocazione = classi (donut) + regioni/settori/paesi a barre
  (`ClassDonut`/`ExposureBarChart`); rimossi i componenti `GeographyChart`/`SectorChart`.

### I.8 — Dettaglio portafoglio: performance a barre (branch `feat/I.1-base-currency`)
- **Backend**: `GET /portfolios/{id}/performance/buckets?granularity=month|year` (TWR, valuta
  portafoglio, ownership + cache).
- **Frontend**: card **Performance** (barre % + linea TWR, toggle Mensile/Annuale); il
  `PositionChart` "Performance history" resta come vista secondaria.

### I.9 — Transazioni paginate (branch `feat/I.1-base-currency`)
- **Backend**: `GET /portfolios/{id}/transactions?limit=&offset=` → `{transactions, total,
  limit, offset}` (default 20, max 100, ordine `date DESC, created_at DESC, id DESC`,
  ownership check); script e2e aggiornati.
- **Frontend**: paginatore sotto la tabella (range + prev/next), refetch della pagina corrente
  dopo le mutazioni.

### Fix nella stessa PR
- **#99 — import export vecchi**: l'import non fallisce più se il documento non ha `price_source`
  (default `yahoo`), export esteso con `price_source`/`asset_class`, versione documento gestita.
- **#100 — posizioni chiuse e residui**: la riga Active non conta più il costo residuo (AVCO) di
  posizioni chiuse; arrotondamento degli importi. Niente più P/L fittizio -100%.

### EPIC I — stato
Tutte le sub-issue **I.1–I.9 completate** e rilasciate in **v0.5.0** (PR #98 mergiata su
`develop`/`main`). Nota: la gestione del **capitale disponibile / versamenti-prelievi** (conto
titoli) è tracciata a parte nell'issue **#101** e sarà una PR separata.

## EPIC J — Nuove asset class: bond, certificati, fondi pensione, conti deposito (#113) — pianificata

Estendere Peculium agli **investimenti a reddito fisso e non quotati**: obbligazioni (tipo, cedola,
scadenza, esposizione geo/settoriale), certificati d'investimento (prodotti strutturati), piani
pensionistici complementari (fondi pensione/PIP) e conti deposito. L'analisi finanziaria
(30 Ago 2026) ha verificato che le quattro classi sono già tracciabili con il modello attuale
(`price_source` `manual`/`none`, TWR al costo), a patto di colmare due gap trasversali:
**inserimento prezzo manuale in UI** (assente) e tipo **`cash`** non selezionabile nel form di
creazione.

| Issue | Titolo | Componente | Priorità |
|-------|--------|------------|----------|
| #105 | J.1 — Inserimento prezzo manuale (endpoint + UI) | Backend + Frontend | MVP (sblocca tutte le classi) |
| #106 | J.2 — Metadati asset fixed income (scadenza, emittente, `attributes` JSONB) | Backend + Frontend | MVP |
| #107 | J.3 — Tipo `cash` in UI + nuovo tipo `certificate` | Backend + Frontend | MVP |
| #108 | J.4 — Esposizione geo/settoriale per fixed income (opt-in `exposure_kind`) | Backend | Post-MVP |
| #109 | J.5 — Maturazione interessi conti deposito | Backend + Frontend | Post-MVP |
| #110 | J.6 — Metriche bond: duration, YTM, current yield | Backend | Post-MVP |
| #111 | J.7 — Allocazione per merito di credito (`asset_credit_weights`) | Backend + Frontend | Post-MVP |
| #112 | J.8 — Wrapper piani pensionistici (comparti/sub-fondi) | Backend + Frontend | Post-MVP (bassa) |

**Prima PR consigliata**: J.1 + J.2 + J.3 insieme — set minimale e non regressivo che abilita
un'esperienza first-class per tutte e quattro le classi.

**Decisioni aperte** (da risolvere in implementazione): cedola vs dividendo come tipo a sé
(MVP: `coupon` = `TxDividend` + note); `asset_class` dei certificati (`other` vs nuovo
`structured`); policy di esposizione opt-in (`exposure_kind`) per non impattare i portafogli
esistenti; gestione prezzo clean/dirty per i bond (MVP: prezzo inserito usato as-is).

## EPIC K — Redesign UX/UI (in corso, branch separato)

Bozza progettuale di revisione completa dell'interfaccia basata **solo sulle funzionalità**
attuali (l'implementazione visiva è considerata sostituibile), per un'interfaccia moderna
usabile su PC, tablet e mobile. **Spec completa**: `docs/UX-REDESIGN.en.md` / `.it.md`
(chapter 1–12, wireframe ASCII, diagrammi mermaid, token, componenti, roadmap).

> **Branch isolato `feat/K-ux-redesign`**: tutto il redesign vive qui e verrà mergiato solo
> se il risultato convince; in caso contrario il branch si scarta senza impattare `develop`.

**Fase 0 — ✅ completata (questo branch)**: stesura della specifica di design + aggiornamento
STATUS/PLAN. Nessuna modifica al codice UI.

**Decisioni registrate** (D1–D11, riflesse in tutta la spec):

| # | Tema | Decisione |
|---|------|-----------|
| D1 | Lingua UI | i18n leggero IT+EN, **default IT**, fallback EN (introdotto in K.1) |
| D2 | Nav mobile | **Bottom nav** (4) + **FAB** + "More" sheet; hamburger declassato |
| D3 | Scope switcher | **Naviga** tra `/` (patrimonio) e `/portfolios/:id` (non filtra) |
| D4 | Ispezione righe | **Drawer** destro ≥ `lg`, **bottom sheet** < `lg` |
| D5 | Font | **Inter + mono** self-hosted (`@fontsource`, no CDN) |
| D6 | P/L a11y | Segno + ▲▼ sempre **+** toggle palette **CVD** (blu/arancio) in Preferenze |
| D7 | Health prezzi | Voce separata "Data & Sync" ora; in futuro spostabile nel menu **Amministrazione** (admin, debug/log) |
| D8 | Primo avvio | **Checklist guidata** portafoglio → asset → transazione |
| D9 | Tema | **Default = segui sistema** (non più dark forzato); light/dark pari |
| D10 | Range hero | **Bucket-driven** (mensile/annuale) ora; serie giornaliera del patrimonio come fast-follow |
| D11 | Undo | **Toast ⟲ Undo** (5s) sul delete transazione |

**Fasi di implementazione** (da delegare a `frontend`, richieste backend a `backend`):

| Fase | Contenuto | Backend ask |
|------|-----------|-------------|
| **K.1 Foundations** | Token (elevazione a 4 step, type scale, font D5, palette CVD), i18n (D1), tema→system (D9), primitive `DataTable`/`Drawer`/`Sheet`/`Tabs`/`AsyncCard`/`KpiStrip`/`PnlValue`/`PeriodChips` | — |
| **K.2 Shell adattiva** | BottomNav+FAB+QuickAction (D2), rail@sm–lg, header condensante, entry "Data & Sync" relocabile (D7); ScopeSwitcher (D3) e FreshnessStamp rinviati a K.3 | — |
| **K.3 Overview** | 🔄 *in corso — K.3a completata (hero zona A + chip bucket-driven D10, strip qualità, checklist D8, ScopeSwitcher D3, FreshnessStamp) e K.3b completata (sparkline valore nei card portafogli, zona C).* Restano in K.3: digest allocazioni (zone B sotto il hero), card-ificazione tabelle (K.4/K.5) | serie giornaliera del patrimonio (fast-follow); endpoint batchato per gli storici delle sparkline (fast-follow solo se il numero di portafogli cresce) |
| **K.4 Entità → tab** | ✅ *completata — K.4a (portafoglio: shell `+layout` con header sticky — identità, strip KPI, `[+ Transazione]`, menu `⋯` export/import/elimina — e tab nested-route Overview/Positions/Activity/Allocation con context condiviso), K.4b (asset: shell `+layout` con header sticky — identità + chip quotazione + menu `⋯` — e tab nested-route Panoramica/Esposizione/Dati con context condiviso; nuovo blocco "Dove è detenuto" nel tab Panoramica) e K.4c (filtri Attività persistiti nell'URL — tipo/asset/intervallo date — con refetch filtrato; form transazione responsive Modal/Sheet (D4); eliminazione transazione con toast undo (D11)).* | inventory holdings per-portafoglio (derivata client-side da `GET /dashboard` in K.4b, nessun endpoint nuovo) |
| **K.5 Power layer** | ✅ *completata — K.5c (toggle palette CVD opzionale, D6: store `palette.svelte.ts`, token `html.cvd`, mirror `chartPalette`, controllo in Preferenze), K.5b ("view as table" nei sei wrapper dati: toggle segmentato condiviso `ui/ChartTableToggle`, tabella accessibile con gli stessi dati, cap opt-out per i chiamanti che elencano già le righe), K.5a (⌘K command palette montata nella shell: sezioni Vai a/Asset/Azioni, matcher locale senza dipendenze, combobox+listbox ARIA completo), backend del drill-down (`GET /portfolios/{id}/allocation/drill` + `GET /dashboard/allocation/drill`: `dim`+`key` → contribuzioni per asset, totali identici ai bucket delle allocazioni) e K.5d (drill-down frontend: fette/barre cliccabili in `ClassDonut`/`ExposureBarChart` e `AllocationDrillPanel` drawer/sheet (D4) con gli asset contribuenti, montato una volta su dashboard e tab Allocazione).* | ✅ endpoint contribuzione drill-down (`dim+key` → asset) esposto; drawer frontend completato in K.5d |

> **K.1a — Fondamenta token/font/tema — ✅ completata (questo branch)**: scala
> di elevazione a 4 step (`--surface-0..3`, con gli alias `--surface`/
> `--surface-raised` mappati per compatibilità), token
> semantico `--info`, `--chart-grid` cablato sulle griglie ECharts a ~8% di
> opacità (mirror in `chartTheme.ts`), gradini tipografici `text-hero`/
> `text-micro`, token di motion (`duration-fast/base/slow`, `ease-standard`,
> neutralizzazione `prefers-reduced-motion`), font self-hosted **Inter +
> JetBrains Mono** via `@fontsource` (D5) e **tema di default → system** (D9).
> Nessun markup di componente toccato: restano da fare il resto di K.1
> (primitive → completate in K.1c, i18n D1 → K.1b, palette CVD).

> **K.1b — Livello i18n (D1) + pagina Preferenze — ✅ completata (questo branch)**:
> layer i18n leggero senza dipendenze esterne in `frontend/src/lib/i18n/`:
> runtime a rune `index.svelte.ts` (`SUPPORTED_LOCALES = ['it','en']`,
> `DEFAULT_LOCALE = 'it'`, `locale` reattivo, `setLocale()` persistente in
> `localStorage['peculium-locale']` + sync `<html lang>` + ascolto cross-tab,
> `t(key, params)` con interpolazione `{name}`), dizionari `en.ts`
> (canonico) / `it.ts` verificati con `satisfies Dictionary` (identità
> strutturale garantita alla compile-time), chiavi annidate a due livelli
> `group.key` esposte appiattite (`nav.dashboard`) e tipizzate come unione
> `MessageKey`; chiave sconosciuta → chiave stessa con fallback EN (D1) e
> warning solo in dev. `app.html` parte con `lang="it"` (valore statico =
> DEFAULT_LOCALE, sincronizzato a runtime). Nuova pagina **Settings →
> Preferenze** (`/settings/preferences`, tab tra Password e Valute come da
> spec §6.6): tema Chiaro/Scuro/Sistema sullo store esistente (default
> system, D9) e lingua IT/EN su `setLocale` (default italiano, D1), con
> apply immediato e persistenza locale. Prima passata di migrazione:
> navigation della shell tradotta (`SidebarNav`, `AppHeader`, `UserMenu`,
> `ThemeToggle`, `SettingsTabs`, `MobileDrawer`, skip-link `AppShell`); le
> altre pagine restano con la copia mista EN/IT fino alle rispettive fasi
> (migrazione progressiva).

> **K.1c — Primitive UI di base — ✅ completata (questo branch)**: sei nuove
> primitive accessibili (WCAG 2.2 AA) e theme-aware in
> `frontend/src/lib/components/ui/`, ancora non consumate da nessuna pagina
> (l'adozione avviene con K.2–K.5): `PnlValue` (segno + ▲▼ + colore semantico,
> zero neutro — D6), `AsyncCard` (stati loading/errore/vuoto/dati per singola
> card, con Retry isolato), `PeriodChips` (radiogroup compatta per i periodi
> dei grafici, navigazione con frecce), `Drawer` (drawer di ispezione ≥ `lg`,
> focus-trap + Esc/backdrop + ripristino — D4), `Sheet` (bottom sheet < `lg`,
> stessa API con handle decorativo) e `Tabs` (tablist ARIA legata alle route,
> focus roving). Helper condivisi estratti: `ui/focus-trap.ts` e
> `ui/transitions.ts` (transizioni sui token motion, `prefers-reduced-motion`
> rispettato anche nelle transizioni JS); `Modal`/`MobileDrawer` restano
> invariati. **`DataTable` e `KpiStrip` sono rinviati
> deliberatamente a K.4**, dove verranno progettate attorno ai reali call
> site. In K.1 resta solo la palette CVD (il toggle è pianificato per K.5);
> i18n (D1) è completata in K.1b.

> **K.2 — Shell adattiva — ✅ completata (questo branch)**: la shell
> (`frontend/src/lib/components/layout/`) ora è davvero adattiva sulle tre
> classi di dispositivi della spec §5.1–5.2, con desktop invariato. Nuovo
> helper reattivo `lib/stores/viewport.svelte.ts` (`matchMedia` su 640/1024,
> `isPhone`/`isTablet`/`isDesktop`, SSR-safe via `browser` + feature check,
> fallback desktop). **Tablet `sm`–`lg`**: la sidebar è forzata a rail di
> icone da 64px (`collapsed` forzato dalla shell; la preferenza persistita
> `peculium-sidebar` vale solo da `lg` in su), niente hamburger né bottom
> nav; menu utente nel footer del rail (in header resta solo il tema).
> **Telefono < `sm`**: nessuna sidebar — `BottomNav` fissa con 4 destinazioni
> (Panoramica · Portafogli · Asset · Altro, decisione D2, target ≥ 44px,
> `env(safe-area-inset-bottom)`, stato attivo con la regola del prefisso più
> lungo di `SidebarNav`) + `Fab` che apre la `QuickActionSheet` sul `ui/Sheet`
> K.1c (Aggiungi transazione → portfolio picker via `portfolioApi.list()`,
> Aggiungi asset → `/assets`, Aggiorna prezzi → `pricesApi.refresh()` con
> toast, *Inserisci prezzo* disabilitato "In arrivo" finché non arriva J.1).
> Il `MobileDrawer` è declassato a sheet "Altro" (focus trap/Esc/chiusura alla
> navigazione intatti, ora rende la navigazione `Sidebar`); padding inferiore
> extra in `<main>` sui telefoni. **Header condensante** (tutte le misure):
> prop `condensed` misurata sulla scroll container della shell (soglia 16px,
> listener passivo), altezza 56→44px con transizione CSS rispettosa di
> `prefers-reduced-motion`. **D7**: la voce Admin "Health" diventa **"Dati e
> sincronizzazione"** (`nav.dataSync`), definita in un unico punto di config
> (`adminItems` di `SidebarNav`), route `/admin/health` invariata. Nuove
> chiavi i18n EN/IT (shapes identici): `nav.overview`, `nav.more`,
> `nav.bottomNav`, `nav.dataSync`, `fab.open`, il gruppo `quickActions.*`
> (etichette, hint, "In arrivo", toast di refresh); rimosse le chiavi morte
> dell'hamburger (`header.openMenu`/`closeMenu`, `nav.drawer`, `nav.health`).
> **Rinviati a K.3** (come da task): `ScopeSwitcher`, `FreshnessStamp`,
> `DataTable`/`KpiStrip` e i form globali di creazione/transazione.

> **K.3a — Hero dell'Overview — ✅ completata (questo branch)**: la
> dashboard (`routes/+page.svelte`) è ricostruita attorno al modello hero
> (spec §6.1 zone A–B, decisioni D3/D8/D10) senza nuovi endpoint né dipendenze.
> **Zona A**: numero unico — valore netto `summary.active.value` in
> `base_currency` con il token `text-hero` + `tabular-nums` — riga P/L firmata
> con due `PnlValue` (importo + % dell'active breakdown), chip secondari muted
> (Realizzato via `PnlValue`, Dividendi, Investito) e **`FreshnessStamp`**
> ("Prezzi alle HH:MM" da `finished_at` del refresh di sessione, tono
> `--info`/muted, stato "in aggiornamento"; toast e semantica
> una-volta-per-sessione invariati). A destra (stacked su telefono) il grafico
> **valore vs investito**: `CapitalChart` nella nuova variante opt-in
> `compact` (canvas 240px, niente slider dataZoom) alimentato dagli **stessi**
> bucket `dashboardPerformance`; **chip periodo guidati dai bucket (D10)**:
> `PeriodChips` finestre i bucket lato client — mensili → 1Y (ultimi 12) /
> 3Y (ultimi 36) / TUTTO, annuali → solo TUTTO con chip nascosti; opzioni
> derivate dalla `granularity` del payload, scelta persistita in
> `localStorage['peculium-hero-period']`. L'`InvestmentsTable` Active/Closed
> si apre in un `<details>` "Dettaglio" a divulgazione progressiva; la vecchia
> card "Capital invested" è assorbita nell'hero. **Zona B**: card Performance
> invariata (toggle Monthly/Annual che guida anche l'hero) ora in griglia
> 2-colonne con il donut "Allocation by portfolio"; **strip qualità**
> (`DataQualityStrip`, chip-link solo quando azionabile: FX mancante da
> `summary.fx_missing_count/value` → `/settings/currencies`, esito refresh
> rate-limit/issues/failed → `/admin/health`; i contatori missing-sector/
> country/stale sono rinviati perché non esistono sul tipo `Dashboard` —
> richiesta backend futura); **checklist primo avvio** (`FirstRunChecklist`,
> D8: `<ol>` accessibile ①portafoglio ②asset ③transazione con stati
> done/current/pending derivati solo dal payload, sparisce con portafogli
> presenti); **`ScopeSwitcher`** (D3: `<select>` nativa da `dash.portfolios`,
> "Tutti i portafogli (Patrimonio)" + i portafogli, selezionarli **naviga** a
> `/portfolios/{id}`). Zone C–E (card portafogli, Allocazione complessiva,
> Invested assets) invariate; nessuna posizione fissa aggiunta (compatibile
> con la bottom nav K.2). Nuove chiavi i18n EN/IT (shape identici): `hero.*`,
> `period.*`, `quality.*`, `freshness.*`, `checklist.*`, `scope.*`. **Rinviati
> a K.3b+**: sparkline nei card portafoglio, digest allocazione, tabelle in
> card, `DataTable`/`KpiStrip`, skeletons `AsyncCard` per card.

> **K.3b — Sparkline nei card portafoglio — ✅ completata (questo branch)**:
> nuovo componente minimale `domain/Sparkline.svelte` — line chart ECharts
> senza assi/legenda/tooltip/zoom (tree-shaking: solo `LineChart` +
> `GridComponent` + `CanvasRenderer`), griglia a bordi zero e `yAxis scale`
> per usare tutta l'altezza, colore semantico `marketValue` di default (prop
> `color`), riempimento d'area discreto al 10% opzionale (`area`),
> `sampling: 'lttb'` (pattern spec §9.2), serie `silent` (nessun hover),
> strip d'altezza fissa via `heightClass` (default `h-10`); accetta numeri
> semplici (asse indice) o punti `{date, value}` (asse temporale hidden, i
> buchi di calendario restano veritieri) e **non renderizza nulla** sotto i
> 2 punti; wrapper `role="img"` con `aria-label`; re-init al flip di tema
> con il pattern `{#key resolved()}`. Le card portafoglio della dashboard
> (zona C §6.1) mostrano ora lo strip dello storico del valore: gli esiti
> arrivano da `portfolioApi.history(id)` (i `market_value` della serie
> convertiti in punti `{date, value}`), caricati **in parallelo e in background**
> dopo il payload della dashboard — le card si renderizzano subito e le
> sparkline si aggiungono quando le risposte landano; guard monotònico
> last-write-wins sul round (e store chiave-per-id) perché una risposta
> obsoleta non possa finire dietro una più recente o sulla card sbagliata;
> chiamate fallite silenziose (nessun toast, nessuno spazio riservato). A
> scala familiare N GET parallele sono accettabili (la cache GET da 60s
> deduplica il round post-refresh): l'endpoint batchato è il fast-follow
> backend se il numero di portafogli cresce. Il digest allocazione "See
> all" resta rinviato (non esiste ancora una vista allocazione dedicata;
> la card "Allocazione complessiva" è invariata). Nuove chiavi i18n EN/IT
> (shape identici): `sparkline.trend`, `sparkline.valueTrend`.

> **K.4a — Dettaglio portafoglio → tab annidati — ✅ completata (questo
> branch)**: la pagina unica `routes/portfolios/[id]/+page.svelte` (568 righe)
> è divisa in una shell `+layout.svelte` + quattro tab nested-route (spec
> §4.2/§6.2): **Overview** (`+page.svelte`: card `InvestmentsTable` + conteggio
> asset, card Performance I.8 col toggle Mensile/Annuale, "Performance
> history" `PositionChart`, digest allocazione `ClassDonut` + link),
> **Positions** (`positions/`: tabella holding completa), **Activity**
> (`activity/`: tabella transazioni paginata I.9, piè di pagina Previous/Next
> invariato), **Allocation** (`allocation/`: griglia I.7 ciambella classi +
> barre regioni/settori/paesi con stati d'errore isolati). Le tab sono URL
> reali (`ui/Tabs` K.1c, stato attivo derivato dalla route, scroll orizzontale
> su telefono): deep-link e pulsante indietro funzionano; le azioni del
> portafoglio (export, import — con *questo* portafoglio come unico target di
> sovrascrittura — ed elimina con conferma + redirect alla lista) vivono nel
> menu `⋯` dell'header, non in una quinta tab. **Condivisione dati**: il
> layout possiede ogni fetch (portfolio, summary, finestra transazioni, bucket
> TWR, storico, tre allocazioni, refresh prezzi una-volta-per-sessione +
> refill E.9 post-mutazione) e lo espone tramite **context** tipizzato Svelte 5
> (`context.ts`: `createContext`, membri in getter che fanno proxy al `$state`
> del layout); le tab non rifetchano nulla e il layout restando montato
> conserva dati e finestra di paginazione tra i cambi di tab. Header sticky:
> link indietro, nome + (valuta), descrizione, `[+ Transazione]`, strip KPI
> (valore + `PnlValue` + chip investito/realizzato/dividendi in valuta
> portafoglio, composizione dell'hero K.3a). Nessuna API o logica di business
> toccata; nessuna dipendenza nuova. Nuove chiavi i18n EN/IT (shape identici):
> gruppo `portfolio.*` (etichette tab, back, azioni header/menu, conferma
> eliminazione, link digest) e `common.delete`/`common.cancel`. **Restano a
> K.4**: filtri Attività in URL state + editing in
> sheet + undo toast D11 (K.4c); il tab dell'asset è completato in K.4b
> (nota sotto).

> **K.4b — Dettaglio asset → tab annidati + "Dove è detenuto" — ✅ completata
> (questo branch)**: la pagina unica `routes/assets/[id]/+page.svelte`
> (1137 righe) è divisa in una shell `+layout.svelte` + tre tab nested-route
> (spec §4.2/§6.3, stesso pattern e contesto tipizzato di K.4a):
> **Panoramica** (`+page.svelte`: card `PriceChart` con selettore
> 1M/3M/1Y/YTD/MAX, zoom in-place e marcatori di split invariati; **NUOVO**
> blocco "Dove è detenuto"; griglia sola-lettura "Dati principali"),
> **Esposizione** (`exposure/`: card Distribuzione geografica — barre top-15
> paesi + donut regioni aperto — e Distribuzione settoriale, banner
> equity-universe quando non applicabile) e **Dati** (`data/`: form
> "Caratteristiche" con dirty-save e selettore `price_source`, "Zona
> pericolosa", slot riservati EPIC J — prezzo manuale J.1 e attributi
> obbligazionari J.2 — muti e senza comportamento). Le tab sono URL reali
> (`ui/Tabs` K.1c, stato attivo dalla rotta, scroll orizzontale su telefono):
> deep-link e pulsante indietro funzionano. **Condivisione dati**: il layout
> possiede ogni fetch e mutazione — load iniziale combinato
> (asset+quote+prices+exposure+splits, 404 → redirect a `/assets`), refresh
> prezzi una-volta-per-sessione con refill di quote/prezzi, PATCH metadati
> sul `form` condiviso (il tab Data vi si lega in binding diretto; i prefill
> continuano a sincronizzare `form.isin`), e l'intera machinery exposure —
> salvataggi per dimensione con fonte di provenienza, prefill
> JustETF/Morningstar/Yahoo, derivazione regioni, guard sulle somme, badge di
> provenienza, re-hydration all'apertura — esposta via getter nel context
> (`context.ts`); le modali geo/sector e il `ConfirmDialog` di eliminazione
> sono montati una sola volta nella shell (le liste `$bindable` restano
> `$state` nativi del layout), la tab li apre con
> `openGeoModal`/`openSectorModal`. Gli helper puri di normalizzazione
> (`roundWeight`/`capAtHundred`/`positiveCountries`/`withoutOther`/
> `sectorsList`) si spostano invariati in `exposure-utils.ts`, condiviso da
> shell e tab. **Header sticky**: link indietro, ticker (mono) + nome, chip
> identità tipo·classe·valuta·exchange, chip "nessun sync automatico" per
> fonti non-Yahoo, strip quote (ultima chiusura + chip 1G/1S/1M/1Y/YTD via
> `PnlValue`, dai vecchi "Metriche quote", + data ultimo prezzo) e menu `⋯`
> (Aggiorna da Yahoo / Backfill storico completo / Elimina asset con conferma
> e redirect) — stesse azioni e busy-flag rispecchiate nella zona pericolosa;
> l'eliminazione asset sbarca così sulla pagina detail (prima solo in lista).
> **"Dove è detenuto"**: una riga per portafoglio che detiene l'asset (link
> al portafoglio, quantità, costo, valore, P&L firmato + ROI nella valuta
> dell'asset) derivata client-side da `portfolioApi.dashboard()`
> (`PortfolioAssets[] → AssetPerformance[]` filtrati sull'id corrente,
> posizioni chiuse escluse) — nessun endpoint nuovo; fetch isolato e non
> bloccante: in attesa il blocco non renderizza, errore → nota muted "non
> disponibili", vuoto → "Non è detenuto in nessun portafoglio". Nessuna API
> o logica di business toccata; nessuna dipendenza nuova. Nuove chiavi i18n
> EN/IT (shape identici): gruppo `asset.*` (tab, header/menu/back, chip
> quote, "Dove è detenuto", dati principali, zona pericolosa, slot
> riservati); etichette tipo centralizzate in `format.ts`
> (`ASSET_TYPE_LABELS`). **Restano a K.4**: filtri Attività in URL state +
> editing in sheet + undo toast D11 (K.4c).

> **K.4c — backend filtri elenco transazioni — ✅ completata (questo
> branch)**: `GET /portfolios/{id}/transactions` accetta ora i parametri
> opzionali e combinabili `type` (buy/sell/dividend/split/fee), `asset_id`
> (UUID) e `from`/`to` (date `YYYY-MM-DD` esatte, bordi inclusivi); valori
> non validi → 400 con il nome del parametro. Il `total` della risposta
> riflette il conteggio **filtrato** (la paginazione del frontend dice
> "1–20 di 42" riferito all'insieme filtrato); WHERE SQL dinamica e
> parametrizzata, ordine/paginazione invariati, chiamate esistenti non
> toccate. Il frontend K.4c è completato nella nota sotto →.

> **K.4c — frontend: filtri Attività + form sheet + undo — ✅ completata
> (questo branch)**: la tab Attività (spec §6.2/§6.4, decisioni D4/D11)
> guadagna la riga di filtri — chip `ui/PeriodChips` per il tipo (Tutte/
> Acquisto/Vendita/Dividendo/Split/Commissione), `ui/Select` sugli asset
> registrati nel portafoglio (da `summary.holdings`, chiuse incluse) e input
> nativi Dal/Al per l'intervallo — interamente **persistita nell'URL**
> (`?type=sell&asset=<uuid>&from=YYYY-MM-DD&to=YYYY-MM-DD`, codec puro in
> `routes/portfolios/[id]/tx-filters.ts`): l'URL è l'unica fonte di verità,
> il layout lo interpreta (getter `txFilters` nel context) e ogni fetch
> delle transazioni lo rispetta — deep link e reload partono già filtrati,
> indietro/avanti ripristina la vista esatta, il piè di pagina mostra il
> totale filtrato. `setTxFilters` scrive con `goto(..., { replaceState,
> keepFocus, noScroll })` e un watcher sulla firma dei filtri riporta la
> finestra alla prima pagina filtrata rifetchandola soltanto (via
> `loadTransactions`, guard monotònico invariato); "Cancella filtri" e uno
> stato vuoto dedicato (`EmptyState`, copy `activity.*`) chiudono il flusso.
> `transactionApi.list` passa i nuovi parametri `type`/`asset_id`/`from`/`to`
> (omessi quando non impostati; `TransactionPage` invariato). **Form
> transazione responsivo (D4)**: `AddTransactionModal` renderizza gli
> snippet condivisi `formBody`+`footer` dentro `ui/Modal` da `sm` in su e
> dentro `ui/Sheet` (bottom sheet) sui telefoni, scelti con lo store
> `viewport`; campi/validazione/totale live identici, `ui/Sheet` guadagna
> `closeLabel` opzionale. **Undo sul delete (D11)**: il toast store accetta
> ora `{ duration, action: { label, onclick } }` e `Toaster` renderizza
> l'azione come pulsante inline (raggiungibile da tastiera, al click chiude
> il toast); Elimina nel form agisce subito — niente più `ConfirmDialog` per
> le transazioni (resta per portafogli/asset/import) — e "Annulla" (5 s)
> ricrea la riga con `transactionApi.create` sul payload catturato: la
> ricostruzione produce un **nuovo id** (accettato su scala familiare, i
> dati economici — data/tipo/importi — sono identici); il refill post-
> mutazione passa da `reloadAfterMutation`, quindi lista (con filtri
> attivi), KPI, allocazioni e performance restano sincronizzati. Nuove
> chiavi i18n EN/IT (shape identici): gruppo `activity.*`, gruppo `tx.*`,
> `common.close`. Nessuna dipendenza nuova; desktop invariato.

> **K.5b — "View as table" nei grafici (spec §9.1) — ✅ completata (questo branch)**:
> nuova primitive `ui/ChartTableToggle.svelte`: disclosure segmentata
> Grafico ⇄ Tabella che avvolge la `SegmentedControl` esistente (tablist di
> veri pulsanti, quindi già raggiungibile da tastiera con `aria-selected`) e
> espone `bind:view` (`'chart' | 'table'`, `$bindable`) più un `name`
> opzionale che interpola il titolo del grafico nel nome accessibile della
> tablist (fallback generico). Il toggle è incorporato **dentro** i sei
> wrapper portatori di dati — `ExposureBarChart` (nome/valore/peso,
> etichette come nei tooltip: nome leggibile + codice grezzo tra parentesi
> via `labelFor`, e cap `maxVisibleRows` rispecchiato con viewport proprio),
> `ClassDonut` (classe via `ASSET_CLASS_LABELS`/valore/peso), `ExposurePie`
> (nome/peso), `PerformanceChart` (periodo/rendimento/TWR cumulativo, con
> `pnlColorClass` come le barre), `CapitalChart` (periodo/investito/valore,
> toggle attivo anche nella variante `compact` dell'hero) e
> `AllocationDonut` (nome/valore/peso, colonna importo omessa con
> `showValue={false}` come nel tooltip) — quindi ogni call site lo riceve
> gratis; la prop `showTableToggle={false}` (default: mostrato) disattiva
> solo dove sotto al grafico è già renderizzato l'elenco delle stesse righe:
> le due ciambelle del tab Esposizione dell'asset (legenda completa
> nome+peso) e le anteprime `mute` nelle due modali esposizione (la griglia
> di pesi editabile *è* quella tabella). Le tabelle riusano le primitive
> `ui/Table`/`Th`/`Td` con `<caption>` sr-only, `scope="col"`, celle
> numeriche destre in `tabular-nums` e gli stessi formattatori dei tooltip;
> il rendering di default resta il grafico (nessuna call-site rotation), in
> vista tabella il canvas viene smontato (esce dall'albero di accessibilità)
> e gli stati vuoti esistenti hanno precedenza sulla tabella (mai tabelle
> vuote, e senza dati il toggle non appare). Non hanno avuto il toggle:
> `PriceChart`/`PositionChart` (viste secondarie/storico prezzi, fuori
> perimetro K.5b) e le `Sparkline` (forma pura, numeri già nella card).
> **Rimandato** (poi arrivato): il drill-down paesi/regioni/settori → asset
> contribuenti (click sulla riga/barra) aspettava solo il frontend — l'
> endpoint di contribuzione (`dim`+`key` → asset) era già esposto (`GET
> /portfolios/{id}/allocation/drill` e `GET /dashboard/allocation/drill`) e
> il drawer è stato completato con K.5d, qui sotto.
> Nuove chiavi i18n
> EN/IT (shape identici): gruppo `chartView.*` (etichette del toggle, nomi
> accessibili con/senza `{name}`, caption e intestazioni di colonna, nomi
> dei grafici senza titolo proprio). Nessuna dipendenza nuova; dati e
> backend intatti.

> **K.5c — Toggle palette CVD (D6) — ✅ completata (questo branch)**:
> nuovo store `lib/stores/palette.svelte.ts` a specchio di quello del tema:
> `$state` reattivo `palette.cvd` (default `false`), persistenza in
> `localStorage['peculium-cvd']`, `setCvd()` che commuta la classe `cvd` su
> `<html>`, listener cross-tab e ri-assert al boot. `app.css` aggiunge gli
> override `html.cvd`/`html.cvd.dark` di `--positive`/`--negative` con una
> coppia blu/arancione derivata Okabe–Ito (tema chiaro `#0072b2`/`#c2410c`,
> tema scuro `#56b4e9`/`#fb923c`; contrasto testo ≥ 4.7:1 su ogni superficie
> dei rispettivi temi — AA) e selettore abbastanza specifico da battere sia
> `:root` sia `.dark`: tutto il testo P&L (`pnlColorClass`/`PnlValue`,
> varianti Badge/StatCard, progress bar) segue i token senza toccare un
> componente, e i glifi segno+▲▼ restano (l'indipendenza dal colore è già
> garantita dalla K.1c). Lato grafici: in `lib/chartPalette.ts` la tabella
> `CHART_SEMANTIC_COLORS_CVD` (solo `positive`/`negative`;
> `marketValue`/`costBasis`/`realized`/`splitMarkLine`/`other` invariati) e
> `chartSemanticColors()` che legge lo store, quindi i `$derived` dei
> chiamanti sono reattivi al flip; l'unico wrapper che dipinge la coppia è
> `PerformanceChart`, il cui `{#key}` ora include anche la palette —
> verificati gli altri (`PositionChart`, `Sparkline`, `CapitalChart`,
> `ClassDonut`, `PriceChart`): non consumano colori P/L, nessun cambio.
> Settings → Preferenze: seconda `SegmentedControl` «Colori utile/perdita»
> (Classica/Accessibile ai daltonici) collegata a `setCvd`/`palette.cvd`,
> apply immediato. Il bootstrap pre-paint di `app.html` applica anche la
> classe `cvd` (niente flash verde/rosso al reload). Nuove chiavi i18n
> EN/IT (shape identici): `preferences.colorGroup`,
> `preferences.paletteClassic`, `preferences.paletteCvd`,
> `preferences.paletteHint`. Nessuna dipendenza nuova; backend intatto.

> **K.5a — ⌘K Command palette (spec §8.1) — ✅ completata (questo branch)**:
> nuovo `layout/CommandPalette.svelte` montato una sola volta nell'`AppShell`
> sul tier modale (z-40, sotto i toast); lo stato `open` `$bindable` resta
> alla shell (`paletteOpen`), che lo condivide col nuovo trigger in
> `AppHeader` — pulsante di ricerca presente a ogni misura: solo icona sotto
> `lg` (affordance di ricerca sul telefono senza quinto elemento nella bottom
> nav, decisione D2) e pill etichettato con l'accordo della piattaforma
> (`⌘K`/`Ctrl K`, UA-detected) da `lg`. La chord globale ⌘K/Ctrl+K vive nel
> componente (`<svelte:window>`, quindi solo dentro il gate di auth: il Login
> non è toccato); Esc, backdrop e i cambi di rotta chiudono, e il
> `ui/focus-trap.ts` condiviso di K.1c gestisce Tab-cycling e ripristino del
> focus sul trigger (nessun focus orfano). Modello ARIA = combobox APG con
> listbox raggruppata: dialog → un solo input `role="combobox"`
> (`aria-expanded`/`aria-controls`/`aria-autocomplete="list"`/
> `aria-activedescendant`) → `role="listbox"` di `role="group"` per sezione e
> righe `role="option"` non focusabili (di proposito: l'unico tab stop è
> l'input). Sezioni, rese solo se non vuote: **Vai a** = Panoramica,
> Portafogli, Asset, Dati e sincronizzazione, Impostazioni + le quattro
> sottosezioni `settingsTabs.*` (tutte via `resolve()` + `goto`) più i
> portafogli da `portfolioApi.list()` → `/portfolios/{id}`; **Asset** =
> asset registrati da `assetApi.list()` (label nome + hint ticker) →
> `/assets/{id}`, seguiti dalla riga live «Cerca su Yahoo "…"» che invoca
> `assetApi.lookup()` solo da 2 caratteri con debounce 300 ms, counter
> anti-race e failure silenziose — la selezione naviga a `/assets` (creazione
> fuori scope, come da task); **Azioni** = Aggiungi transazione (stessa
> logica Fab K.2: portafoglio unico → dettaglione, altrimenti lista),
> Aggiorna prezzi (`pricesApi.refresh()` + gli stessi toast
> `quickActions.*`), Cambia tema (cicla chiaro → scuro → sistema sullo store
> del tema; l'hint anticipa il target), Attiva/disattiva palette CVD
> (`setCvd`, hint = variante risultante) e Mostra/nascondi barra laterale
> (solo desktop, tramite il callback `ontogglesidebar` della shell che pilota
> lo stato persistito `collapsed`). Matching senza dipendenze: matcher locale
> ~15 righe con scoring prefisso > sottostringa > sottosequenza,
> case-insensitive, su haystack `label + hint + keywords`; stato «Nessun
> risultato» dedicato. Dati pigri: fetch solo a dialogo aperto (assorbito
> dalla cache GET 60 s di `services/api.ts`, che ogni mutazione già invalida
> → self-refresh senza hook), cap 8 per sezione dinamica a query vuota,
> fallimenti silenziosi (le sezioni statiche restano utili). Tastiera: ↑/↓
> con wrap, Home/End, Invio eseguito con guardia `isComposing`, Esc chiude;
> la lista segue la riga attiva con `scrollIntoView({ block: 'nearest' })`.
> Motion: solo fade del backdrop via `backdropFade()` (0 ms con
> `prefers-reduced-motion`), nessuna animazione del pannello; layout
> centrato `max-w-lg`/`max-h-[70dvh]` su `bg-overlay/50`. Header condensante
> intatto (trigger `size="icon"` h-9 ≤ h-11). Nuove chiavi i18n EN/IT (shape
> identici): gruppo `commandPalette.*` (title, trigger, inputLabel,
> placeholder, sectionGoTo/sectionAssets/sectionActions, noResults, yahooRow
> con `{query}`, searching, toggleTheme/toggleCvd/toggleSidebar, hintNavigate/
> hintSelect/hintClose); le label di destinazioni e azioni riusano i gruppi
> esistenti. Nessuna dipendenza nuova; backend e pagine invariate.

> **K.5d — Drill-down allocazioni frontend (spec §6.5, decisione D4) — ✅
> completata (questo branch)**: i grafici di allocazione sono diventati
> punti d'ingresso: prop opzionale `onDrill` in `ClassDonut` (chiave classe
> grezza della fetta cliccata) e `ExposureBarChart` (nome grezzo della riga
> — `US`, `North America`, `Financials` — anche quando `labelFor` mappa
> l'etichetta dell'asse); il click arriva dal prop `onclick` (evento ECharts)
> del wrapper svelte-echarts, quindi il binding si rigenera con l'istanza
> alla re-init `{#key}` senza rompere il re-init, e con drill attivo
> fette/barre mostrano il cursore `pointer` (default altrimenti). Nessun
> drill per `ExposurePie` (composizione di un singolo asset, non un
> aggregato) né per le modali esposizione. Nuovo componente
> `domain/AllocationDrillPanel.svelte`: elenca gli asset contribuenti di
> un bucket renderizzando `ui/Drawer` (≥ `lg`) o `ui/Sheet` (< `lg`) via lo
> store `viewport`; controllato (`open`/`onClose` della pagina), carica alla
> apertura e a ogni cambio di `dim`/`key` con un `fetcher` iniettato per
> scope (`dashboardAllocationDrill` per il patrimonio, `allocationDrill(id, …)`
> per il portafoglio — `GET /dashboard/allocation/drill` e
> `GET /portfolios/{id}/allocation/drill`, chiave URL-encoded da `params`
> quindi a posto anche con spazi), request id monotònico contro le risposte
> stale e stati secondo le convenzioni `ui/AsyncCard` (skeleton, errore su
> una riga + Riprova, vuoto «Nessun asset in questa fetta», dati). Tabella
> `ui/Table`/`Th`/`Td` con caption sr-only e cinque colonne — Asset (ticker
> linkato a `/assets/{id}` + nome muted su seconda riga), Valore, Peso
> (dentro il bucket), Contributo e Quota della fetta (contributo ÷ totale
> del bucket, con guardia a trattino per bucket vuoti) — nell'ordine
> decrescente di contributo già servito dal backend; sotto `sm` le righe
> si impilano in chiave–valore e nessun viewport scrollabile annidato
> (scorre solo il corpo del drawer/sheet). Dashboard e tab Allocazione
> montano UN pannello ciascuno con stato condiviso (`drillOpen`/`drillDim`/
> `drillKey`/`drillTitle`): `key` è il valore grezzo del grafico, `title`
> l'etichetta che il grafico mostra (`ASSET_CLASS_LABELS` per le classi,
> `countryDisplayName` per i paesi, verbatim per regioni/settori).
> Allineato `ui/Drawer` alla controparte `ui/Sheet`: nuovo prop `closeLabel`
> per il nome accessibile della ✕ (default invariato, nessun consumatore
> rotto). Nuove chiavi i18n EN/IT (shape identici): gruppo `drill.*`
> (caption `{name}`, contributingAssets `{count}`, colAsset/
> colContribution/colShare, empty, error, retry); le intestazioni
> Valore/Peso riusano `chartView.colValue`/`chartView.colWeight` e la ✕
> `common.close`. I dati di allocazione delle pagine restano gli stessi;
> nessuna dipendenza nuova; backend intatto.

**Integrazioni pianificate**: EPIC J (J.1 prezzo manuale, J.2 metadati FI, J.3 cash/certificate,
J.7 allocazione credito) atterra nel tab **Data** e nella sezione Allocation; EPIC C (metriche di
rischio) in una card **Insights** su Overview/portafoglio; Fase 3 (sharing/ruoli) nel
ScopeSwitcher ("Shared with me") e in Settings → Members.

**Rinviato / slot riservati**: Activity consolidata cross-portafoglio (slot in "More"),
benchmark overlay (EPIC C), density toggle (fuori MVP), passkey/2FA (solo slot).

## Fase 3 — Pianificata

- Multi-tenancy familiare (portfolio_shares)
- Gestione permessi e condivisione

## Fase 4 — Futura

- Tracciamento spese e categorie
- Budget mensile
- Obiettivi di risparmio

## Comandi Utili

```bash
make up              # Avvia tutto con podman-compose (stack dev)
make down            # Ferma tutti i servizi
make reset           # Ferma i servizi e cancella i volumi dati (fresh start)
make logs            # Log in tempo reale
make migrate         # Esegui migration DB
make test            # Test Go (in container)
make test-e2e        # Test end-to-end su stack isolato
make frontend-dev    # Sviluppo frontend con hot-reload
# Test manuali API (estensione REST Client in VS Code) sullo stack test:
#   tests/api-test.http — richieste in ordine contro http://localhost:8081/api/v1
# Smoke test EPIC B sulle allocazioni (stack test, porta 8081):
#   tests/test-epic-b.sh [--step | --no-seed]
# Per ricreare container dopo modifiche (stack dev):
podman-compose -f docker-compose.dev.yml stop <service>
podman rm <container>
podman-compose -f docker-compose.dev.yml up -d --build <service>
```
