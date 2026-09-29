# Peculium — Piano di Sviluppo

## 1. Vision & Architettura

**Peculium** è una webapp self-hosted pensata per homelab, multi-utente, per il tracciamento di investimenti e finanze personali.

### Stack tecnologico

| Livello | Tecnologia | Motivazione |
|---------|-----------|-------------|
| **Backend** | Go 1.23 (con Chi) | Performance, binary singolo, container minimale, ideale per homelab |
| **Frontend** | SvelteKit 2 + Svelte 5 + TypeScript + Vite | SPA moderna, runes API, routing basato su file |
| **Database** | PostgreSQL 16 | Dati finanziari relazionali, CTE per statistiche |
| **Cache/Jobs** | Redis 7 | Rate-limiting Yahoo, caching prezzi |
| **Container** | Docker/Podman + Compose | Homelab standard |
| **Auth** | JWT + refresh token | Self-hosted, no dipendenze esterne |
| **Grafici** | ECharts 5 | ROI, trend, distribuzione |
| **API Design** | RESTful | |

### Perché Go?
- Binary unico, basso consumo di RAM/CPU (ideale per homelab)
- Build veloci, deploy semplice
- Ottimo supporto concorrenza per fetch prezzi multi-fonte

---

## 2. Roadmap (fasi)

### FASE 0 — Setup progetto
- [x] Struttura repository (monorepo con backend Go + frontend SvelteKit)
- [ ] Docker + docker-compose con Postgres + Redis
- [x] CI/CD base (GitHub Actions: build + vet + test Go, check + lint frontend) — **EPIC H.1 (#32)**
- [x] Release automation: publish delle immagini multi-arch su GHCR + GitHub Release sui tag `v*` — **#131**
- [ ] Task runner / Makefile per comandi comuni

### FASE 1 — Core: Auth & Gestione Investimenti
- [ ] Modello dati: User, Portfolio, Asset, Transaction
- [ ] Registrazione/Login multi-utente (JWT)
- [ ] CRUD portafogli e transazioni (acquisto/vendita)
- [ ] Integrazione prezzi via API esterne (Yahoo Finance, Alpha Vantage, ecc.)
- [ ] Dashboard di base: valore portafoglio, gain/loss

### FASE 2 — Statistiche & Visualizzazioni
- [ ] ROI per asset e per portafoglio
- [ ] Distribuzione territoriale (per sede legale dell'asset)
- [ ] Distribuzione per categoria industriale (GICS)
- [ ] Grafici: andamento storico, composizione portafoglio
- [ ] Report periodici (mensile/trimestrale)
- [x] Pagina dettaglio asset (metadati, storico prezzi, distribuzioni geo/settoriali) — **EPIC B.10 (#45)**
- [x] Microservizio Python per metadata ETF (JustETF scraping) — **EPIC B.5 (#11)**
- [x] Endpoint allocazione geografica (weighted sum by region) — **EPIC B.6 (#12)**
- [x] Endpoint allocazione settore (weighted sum by GICS) — **EPIC B.7 (#13)**
- [x] Chart dashboard/portafoglio geo & settore (GeographyChart + SectorChart, universo equity-only + coverage) — **EPIC B.8 (#14)**
- [x] Storico tassi di cambio (FX history, per-date nei series) — **EPIC B.9 (#44)**
- [x] Asset con ticker non-Yahoo: price_source (yahoo/manual/none) — **EPIC G.7 (#53)**
- [x] Chart storico asset: zoom in-place + selettore YTD — **EPIC F.9 (#52)**
- [x] Pagina asset: solo pie chart + modale di modifica esposizione — **EPIC F.10 (#64)**
- [x] Split come marcatori sul chart storico asset (`GET /assets/{id}/splits` + markLine)
- [x] Per-country exposure: tabella `asset_country_weights` + 3 dimensioni (countries/regions/sectors) — **EPIC B.13 (#58)**
- [x] Morningstar exposure source: resolver custom (bootstrap Chromium headless per WAF+JWT, poi SAL service via requests), rotta backend `POST /assets/{id}/fetch-morningstar-exposure`, prefill frontend — **EPIC B.14 (#59)**
- [x] Follow-up B.13/B.14: fetch provider come anteprima non persistente, cache Redis (TTL + `?refresh=1`), provenienza persistita (sorgente + data), prefill settori da Morningstar, redesign modali geo/settore (paesi-first, badge sorgente) — **PR #67**
- [x] Design system & dark mode: token semantici + tema a 3 modalità (default dark), sweep colori, primitive `ui/`, AppShell responsive con sidebar collassabile e ThemeToggle — **EPIC D (#37)**
- [ ] Nuove asset class: bond, certificati, fondi pensione, conti deposito — **EPIC J (#113)**: prezzo manuale (J.1 #105), metadati fixed income (J.2 #106), tipi `cash`/`certificate` (J.3 #107), esposizione fixed income (J.4 #108), maturazione interessi (J.5 #109), metriche bond (J.6 #110), allocazione credito (J.7 #111), piani pensionistici (J.8 #112)
- [ ] Redesign UX/UI completo (navigazione, layout, design system) per PC/tablet/mobile — **EPIC K**, branch isolato `feat/K-ux-redesign`: fondazioni (K.1), shell adattiva (K.2), Overview (K.3), pagine entità a tab (K.4), power layer (K.5). Spec in `docs/UX-REDESIGN.en.md` / `.it.md`

### FASE 3 — Multi-tenancy & Family Sharing
- [x] Ruoli utente e area admin: primo utente = `admin`, approvazione registrazioni, gestione utenti (approva/disabilita, cambio ruolo, reset password) — **#57 (fase A)**. I permessi per-utente sugli asset sono tracciati in #141.
- [ ] Condivisione portafogli tra familiari
- [ ] Viste aggregate famiglia

### FASE 4 — Finanza Personale (estensioni future)
- [ ] Tracciamento spese / categorie
- [ ] Budget mensile
- [ ] Risparmi e obiettivi
- [ ] Reportistica finanziaria unificata

### FASE 5 — Produzione Homelab
- [ ] reverse proxy (Traefik / Caddy) con SSL
- [ ] Backup automatico DB
- [ ] Healthcheck e monitoring
- [ ] Documentazione deploy

---

## 3. Modello Dati (bozza)

### Core
```
User         → id, email, name, password_hash, role, created_at
Portfolio    → id, user_id, name, description, currency, created_at
Asset        → id, isin, ticker, name, type, asset_class, price_source, country, exchange, currency, sector, industry
               + maturity_date, issuer, issuer_country, attributes JSONB (fixed income, from EPIC J / J.2)
Transaction  → id, portfolio_id, asset_id, type (buy/sell), quantity, price, date, fees, notes
Price        → id, asset_id, date, open, high, low, close, volume, source
FxHistory    → base_currency, quote_currency, date, rate, source
AssetRegion  → asset_id, region, weight
AssetSector  → asset_id, sector, weight
AssetCountry → asset_id, country, weight (ISO-3166 alpha-2, from B.13)
AssetExposureProvenance → asset_id, dimension, source, updated_at (where each dimension came from + last update, from B.14)
AssetCredit  → asset_id, rating, weight (credit exposure, post-MVP EPIC J / J.7)
```

### Finanza (Fase 4)
```
Expense      → id, user_id, category_id, amount, date, description, recurring
Budget       → id, user_id, category_id, amount, period (monthly/yearly)
Goal         → id, user_id, name, target_amount, current_amount, deadline
```

---

## 4. Principi di design

1. **Privacy-first**: tutto rimane in homelab, niente dato esce
2. **API-first**: ogni funzionalità backend è accessibile via API
3. **Tutto containerizzato**: `docker compose up` per far partire tutto
4. **Minimal dependencies**: poche librerie esterne, facile da mantenere
5. **Offline-resilient**: gestione gracevole quando le fonti prezzi non rispondono
6. **Mobile-friendly**: interfaccia responsive (PWA opzionale)

---

## 5. Struttura directory

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
│   │   ├── model/       # Struct/entity
│   │   ├── repository/  # DB queries
│   │   ├── service/     # Business logic
│   │   ├── price/       # Price fetcher (Yahoo, etc.)
│   │   ├── geo/         # Macro-regioni, settori GICS, mappature paese/regione
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
│   │       └── settings/
│   ├── svelte.config.js
│   ├── vite.config.ts
│   ├── Dockerfile
│   └── package.json
├── python-service/            # FastAPI: metadata ETF da JustETF (B.5)
│   ├── app/                   # main.py, scraper.py, schemas.py
│   ├── tests/                 # pytest
│   ├── Dockerfile
│   └── requirements.txt
├── tests/                     # test e2e su stack isolato
│   ├── api-test.http          # collection REST Client (VS Code)
│   └── test-epic-a.sh
└── docs/
    ├── BACKEND-GUIDE.en.md
    ├── BACKEND-GUIDE.it.md
    ├── DATABASE-GUIDE.en.md
    ├── DATABASE-GUIDE.it.md
    ├── FRONTEND-GUIDE.en.md
    ├── FRONTEND-GUIDE.it.md
    ├── UX-REDESIGN.en.md
    └── UX-REDESIGN.it.md
```

---

## Stato attuale (23 Set 2026)

**Release v0.6.1** pubblicata su `main` (bug release: stato aggiornamento prezzi globale
nell'header — stampa cliccabile e fascia qualità dati su ogni pagina —, traduzione completa IT/EN
delle schermate rimanenti, fix UX mobile di menu entità, drill-down allocazioni, tooltip grafici e
command palette).
Precedenti release: **v0.1.0** (25 Ago 2026), **v0.2.0** (30 Ago 2026, EPIC A + EPIC B),
**v0.3.0** (11 Set 2026, asset editing overhaul), **v0.4.0** (13 Set 2026, design system & dark
mode, EPIC D/E), **v0.5.0** (17 Set 2026, EPIC I — dashboard & portfolio v2) e **v0.6.0**
(21 Set 2026, EPIC K — redesign UX/UI).

Fase 0 e Fase 1 completate (incluso EPIC A — data correctness & security). Lo sviluppo attivo
procede su `develop`. Realizzate in EPIC B: la **pagina dettaglio asset** (#45, B.10),
il **backfill country/ISO** (B.3), il **microservizio Python JustETF** per l'esposizione ETF e
l'auto-resolve ISIN (B.5), le asset class con allocazione per classi (B.11/B.12), gli **endpoint di
allocazione geo/settore a livello portafoglio** (B.6/B.7), le **chart dashboard/portafoglio** con
universo equity-only e metadati di copertura (B.8, #14), lo **storico FX per-data** (B.9, #44),
il **per-country exposure storage** (B.13, #58: tabella `asset_country_weights`, 3 dimensioni
countries/regions/sectors) e **Morningstar come fonte esposizione** (B.14, #59: resolver custom con
bootstrap Chromium headless per WAF+JWT, rotta backend
`POST /assets/{id}/fetch-morningstar-exposure`, prefill frontend).
Poi EPIC D (design system e dark mode, completata: token, tema a 3 modalità con default dark,
primitive `ui/` e nuovo AppShell responsive), EPIC C (metric di rischio) ed EPIC E (pagine e
componenti di dominio, completata in v0.4.0: dashboard, dettaglio portafoglio, assets/portafogli
con modali, login e impostazioni a tab).
Prossimi: **EPIC I — Dashboard & portfolio v2** (vista aggregata in valuta base, chart e tabelle)
ed **EPIC C — metric di rischio**. **EPIC I completato** (I.1–I.9, PR #98, branch
`feat/I.1-base-currency`): valuta base, dashboard attivo/chiuso, grafico performance TWR +
capitale, allocazioni per classe/paese, tabella asset investiti consolidata, KPI e allocazioni
del dettaglio portafoglio allineati alla dashboard, performance a barre, transazioni paginate.
Nuovo epico pianificato: **EPIC J — nuove asset class (#113)** — obbligazioni, certificati,
fondi pensione e conti deposito, con prezzo manuale (J.1 #105), metadati fixed income (J.2 #106),
tipi `cash`/`certificate` (J.3 #107) come prima PR consigliata, poi esposizione fixed income
(J.4 #108), maturazione interessi (J.5 #109), metriche bond (J.6 #110), allocazione credito
(J.7 #111) e piani pensionistici (J.8 #112). Altro candidato: **EPIC C — metric di rischio**.
La gestione del capitale disponibile (versamenti/prelievi, conto titoli) è tracciata a parte
nell'issue #101. Vedi STATUS.md per lo stato dettagliato.

**Redesign UX/UI — EPIC K (rilasciata in v0.6.0)**: analisi UX/UI completa basata solo sulle
funzionalità attuali e proposta di un'interfaccia moderna per PC/tablet/mobile
(`docs/UX-REDESIGN.en.md` / `.it.md`); implementazione K.1–K.5 completata (fondazioni, shell
adattiva, Overview, pagine portafoglio/asset a tab, command palette, vista tabella, palette CVD,
drill-down delle allocazioni) e mergiata su `develop` (PR #115).

