# Peculium — Il frontend spiegato

> Questo documento spiega come funziona il frontend di Peculium: la pagina web
> che vedi nel browser (grafici, form, pulsanti). È il compagno della guida al
> backend (`docs/BACKEND-GUIDE.it.md`) e della guida al database
> (`docs/DATABASE-GUIDE.it.md`) e non richiede conoscenze di programmazione: i
> concetti di componente, rotta e chiamata API vengono spiegati man mano.
>
> Per i lettori di lingua inglese esiste la versione
> `docs/FRONTEND-GUIDE.en.md`.

---

## 1. Cos'è il frontend

Il frontend è l'applicazione web di Peculium: l'utente accede, crea portafogli,
registra transazioni, aggiunge titoli e guarda i grafici (performance,
allocazione, prezzi).

Alcuni fatti sullo stato attuale:

- **Tecnologia**: SvelteKit 5 (Svelte 5 "rune"), TypeScript, Tailwind CSS e
  **ECharts** per i grafici.
- **Modello di esecuzione**: una **SPA lato client** ("single page
  application"): il server invia un guscio statico e tutto il rendering avviene
  nel browser, che recupera i dati dal backend con `fetch`.
- **Dove gira**: i file compilati sono serviti da **nginx** dentro il container
  `frontend`, pubblicato da docker-compose sulla porta host **3000**
  (http://localhost:3000). nginx funziona anche da **reverse proxy**: le
  richieste del browser a `/api/...` vengono inoltrate al backend Go sulla
  porta 8080.

Il frontend è "stupido di proposito": disegna pagine e grafici, ma ogni numero
(valore del portafoglio, gain/loss, ROI, allocazioni) viene calcolato dal
**backend** (vedi la guida al backend, capitoli 8 e 9) e arriva nel browser come
JSON.

---

## 2. Concetti di base

Un piccolo glossario, nello stesso spirito della guida al backend. Se conosci
già questi termini, salta al capitolo 3.

- **SPA (single page application)**: un'applicazione fatta di una sola pagina
  HTML; quando navighi, il contenuto cambia "sul posto" senza ricaricare la
  pagina.
- **Componente**: un mattoncino dell'interfaccia (una card, una tabella, un
  grafico). In Svelte un componente è un file `.svelte` che contiene HTML, CSS
  e logica.
- **Rotta / pagina**: un URL che l'app può mostrare (`/portfolios`,
  `/assets/123`, ...). In SvelteKit ogni cartella sotto `src/routes/` con un
  file `+page.svelte` è una pagina.
- **Runa**: un simbolo speciale di Svelte 5 che rende lo stato "reattivo" (l'
  interfaccia si aggiorna da sola quando i dati cambiano). Le tre più comuni
  sono `$state` (una variabile reattiva), `$derived` (un valore calcolato da
  altri) e `$effect` (codice che viene rieseguito quando cambiano le sue
  dipendenze).
- **API / endpoint**: un "numero di telefono" del backend (guida al backend,
  capitolo 2).
- **JSON**: il formato testuale usato per scambiare i dati con il backend
  (guida al backend, capitolo 2).
- **JWT / token**: la credenziale che dimostra che hai effettuato l'accesso. Il
  backend emette due token (access e refresh); il frontend li conserva nel
  `localStorage` del browser (capitolo 9).
- **localStorage**: una piccola area di memoria del browser che sopravvive ai
  ricaricamenti della pagina. Peculium ci salva i due token.
- **Libreria grafici / ECharts**: una libreria pronta per disegnare grafici
  (a linee, a torta, ...). Peculium usa ECharts tramite il wrapper
  `svelte-echarts`.
- **Proxy / reverse proxy**: un server (qui nginx) che riceve le richieste e le
  inoltra altrove. Il browser pensa di parlare con "il suo" server, ma `/api/...`
  viene inoltrato al backend Go.
- **CORS**: una regola di sicurezza del browser sulle chiamate a un server che
  vive su un'origine *diversa* (es. localhost:3000 → localhost:8080). Nella
  configurazione standard il browser chiama solo la propria origine (il proxy
  nginx), quindi il CORS non viene coinvolto.

Un'analogia: il **frontend è la sala di un ristorante**. Le pagine sono i
tavoli, i componenti sono i piatti e il client API è il cameriere che porta gli
ordini in cucina (il backend).

---

## 3. Come gira

```mermaid
flowchart LR
    BR["Browser — http://localhost:3000"]
    subgraph FE["container frontend"]
        NGX["nginx — porta 80<br/>file statici + fallback SPA try_files"]
    end
    SPA["App SvelteKit nel browser<br/>adapter-static (ssr=false)<br/>routes + lib/services/api.ts"]
    BE["backend (Go) — porta 8080"]

    BR --> NGX
    BR --> SPA
    SPA -- "fetch('/api/v1/...') stessa origine" --> BR
    BR -- "GET /api/..." --> NGX
    NGX -- "proxy_pass → http://backend:8080" --> BE
```

I pezzi che girano (definiti in `docker-compose.dev.yml` per lo sviluppo locale
e in `docker-compose.yml` per lo stack di release pull-only):

- **frontend** — la pagina web. L'immagine è costruita da `frontend/Dockerfile`
  in due fasi:
  1. `node:22-alpine` installa le dipendenze (`npm ci`) ed esegue
     `npm run build` (in `package.json` è `vite build`);
  2. `nginx:alpine` copia il risultato (`/app/build`) in
     `/usr/share/nginx/html` insieme a `frontend/nginx.conf`, in ascolto sulla
     porta 80. docker-compose la pubblica come **3000:80**.
- **backend** — l'API Go sulla porta 8080 (guida al backend). Il container
  frontend dipende da esso.

### Come funziona il build SvelteKit

- `frontend/svelte.config.js` usa **`@sveltejs/adapter-static`** con
  `fallback: 'index.html'`: un build "statico", adatto a un server che serve
  solo file (nginx). Il `fallback` lo rende una SPA senza flash: ogni URL
  sconosciuto restituisce il guscio `index.html`, che poi carica la pagina
  giusta.
- `frontend/src/routes/+layout.ts` imposta `export const ssr = false` e
  `export const prerender = false`: niente rendering lato server, niente pagine
  precompilate. Il browser riceve `index.html` + gli asset JS/CSS e l'app
  disegna tutto lato client.
- L'output del build va in `frontend/build/` (già presente nel repo).

### nginx

`frontend/nginx.conf` fa due cose:

- `location /api/` → `proxy_pass http://backend:8080;` (più gli header `Host`
  e `X-Real-IP`). Ogni chiamata API quindi esce dal browser sulla stessa
  origine (`/api/v1/...`) e raggiunge il backend Go.
- `location /` → `try_files $uri $uri/ /index.html;` — serve i file statici e
  ripiega sul guscio SPA per le rotte client-side.

### Modalità sviluppo

`make frontend-dev` esegue `cd frontend && npm run dev`: il dev server Vite
sulla **porta 5173** con hot-reload. `vite.config.ts` definisce un proxy per
tutto ciò che sta sotto `/api` → `http://backend:8080`, così le pagine possono
chiamare il backend in esecuzione come se fosse sulla stessa origine. (Il
target del proxy è il nome del servizio docker-compose, quindi funziona solo
dove `backend` è un hostname risolvibile, es. nella rete dei container.)

### CORS

Il backend imposta gli header CORS (guida al backend, capitolo 4), ma nella
configurazione standard non vengono mai usati: il browser chiama sempre
`http://localhost:3000` e nginx inoltra al backend, quindi non c'è nessuna
richiesta cross-origin. Il CORS conta solo se l'API viene chiamata direttamente
da una pagina servita altrove.

---

## 4. Struttura delle cartelle

```
frontend/
├── svelte.config.js        # adapter-static + fallback index.html
├── vite.config.ts          # porta dev 5173 + proxy /api
├── tailwind.config.js      # contenuto Tailwind: ./src/**/*.{html,js,svelte,ts}
├── postcss.config.js       # tailwindcss + autoprefixer
├── Dockerfile              # build node → serve nginx (porta 80)
├── nginx.conf              # file statici + proxy /api/ verso backend:8080
├── static/peculium.svg     # favicon
└── src/
    ├── app.html            # HTML radice (bootstrap tema, meta theme-color, favicon, titolo)
    ├── app.css             # @tailwind + token semantici (:root / .dark) + base layer
    ├── app.d.ts            # namespace App di SvelteKit (segnaposto)
    ├── lib/                # codice condiviso (la "parte interessante")
    │   ├── components/     # primitive ui/, layout/ (AppShell + chrome adattivo), Toaster + wrapper ECharts
    │   ├── services/api.ts # l'unico client API (capitolo 5)
    │   ├── stores/         # auth.svelte.ts, toast.svelte.ts, theme.svelte.ts, viewport.svelte.ts (rune Svelte 5)
    │   ├── i18n/           # dizionari en.ts/it.ts + store del locale reattivo
    │   ├── chartPalette.ts # palette serie + risoluzione token a runtime (dark-aware)
    │   ├── chartTheme.ts   # temi ECharts registrati per light/dark
    │   └── format.ts       # formattatori + etichette classi (capitolo 6)
    └── routes/             # le pagine
        ├── +layout.ts      # ssr=false, prerender=false
        ├── +layout.svelte  # guardia auth, AppShell, Toaster
        ├── +page.svelte    # Dashboard (/)
        ├── login/          # login + registrazione (una pagina, un toggle)
        ├── assets/         # elenco titoli + creazione (autocomplete)
        ├── assets/[id]/    # shell del dettaglio asset: header sticky,
        │   │               #   tab; tutto il data loading + context
        │   ├── +page.svelte        #   tab Panoramica (indice): grafico
        │   │                        #   prezzi, "Dove è detenuto", dati principali
        │   ├── exposure/           #   tab Esposizione (paesi/regioni/settori)
        │   └── data/               #   tab Dati (form metadati, zona pericolosa)
        ├── portfolios/     # elenco portafogli + CRUD + import
        ├── portfolios/[id]/ # shell del dettaglio portafoglio: header
        │                    # sticky, strip KPI, tab; tutto il data loading
        │                    # + context condiviso
        │   ├── +page.svelte        #   tab Panoramica (indice)
        │   ├── positions/          #   tab Posizioni
        │   ├── activity/           #   tab Attività (transazioni paginate
        │   │                       #   + filtri persistiti nell'URL)
        │   ├── tx-filters.ts       #   modello filtri Attività + codec query URL
        │   └── allocation/         #   tab Allocazione
        ├── settings/       # profilo, password, preferenze, valute,
        │                   #   backup e ripristino
        └── admin/          # area "Admin": health/ (dashboard prezzi,
                            #   "Dati e sincronizzazione", aperta a tutti)
                            #   più users/ + settings/ + backup/ (solo
                            #   amministratori)
```

> **Non esiste una pagina `/register` separata**: la pagina di login contiene un
> toggle "Sign in / Register" ed entrambi i form sono gestiti lì (capitolo 10).

---

## 5. Il livello API

Tutto vive in un unico file: `frontend/src/lib/services/api.ts`. Incapsula
l'API HTTP del backend sotto `/api/v1`, aggiunge autenticazione e una piccola
cache GET, ed esporta i tipi TypeScript di tutte le risposte.

### La base e la funzione `request`

- `const BASE_URL = '/api/v1'`. `buildUrl(path, params)` costruisce
  `window.location.origin + '/api/v1' + path` aggiungendo i parametri query,
  quindi la richiesta è sempre **stessa origine** (nginx la inoltra, capitolo
  3).
- `request<T>(path, {method, body, params})` è l'unico punto d'ingresso:
  1. costruisce l'URL e legge la cache in memoria (capitolo sotto);
  2. aggiunge `Content-Type: application/json` e, se esiste un token in
     `localStorage` (`access_token`), l'header
     `Authorization: Bearer <token>`;
  3. chiama `fetch`;
  4. su **401** (e solo per i percorsi non-`/auth/`) prova a **rinnovare la
     sessione** (vedi sotto) e ritenta la richiesta una volta;
  5. su risposte non-OK legge `{error}` dal corpo e lancia
     `new Error(message)` con `status` HTTP allegato (le pagine usano `status`
     per riconoscere es. 404 o 409/422);
  6. un `204` restituisce `undefined`.

### Il flusso di refresh (gestione token)

Il backend emette token **access** di breve durata (15 min) e token **refresh**
di lunga durata (72 h) (guida al backend, capitolo 14). Il frontend li salva
entrambi in `localStorage` con le chiavi `access_token` e `refresh_token`.

Quando una richiesta torna con **401**:

1. il frontend legge `refresh_token` da `localStorage`;
2. chiama `POST /auth/refresh` con `{refresh_token}`;
3. se ha successo salva la **nuova coppia** in `localStorage`, aggiorna
   l'header `Authorization` e **ritenta la richiesta originale una volta**;
4. se fallisce (o c'è un errore di rete) cancella entrambi i token e fa un
   redirect "duro" a `/login` con `window.location.replace('/login')`
   (bypassando volutamente il router di SvelteKit).

### La cache GET

`api.ts` tiene una `Map` a livello di modulo delle risposte GET con TTL di
**60 secondi** (`CACHE_TTL_MS`). Perché: la SPA naviga senza ricaricare, quindi
rimontare la stessa pagina rifarebbe il fetch di endpoint pesanti (dashboard,
summary, storico) senza una cache. Le regole:

- **ogni richiesta non-GET svuota l'intera cache** — volutamente rozzo: quasi
  ogni mutazione può cambiare endpoint aggregati, e svuotare tutto è più
  sicuro che tenere traccia delle dipendenze;
- i dati in cache vengono **deep-copy** prima di essere restituiti
  (`structuredClone`, con ripiego sul round-trip JSON — che non deve mai
  rompere la richiesta), così i chiamanti non possono "avvelenare" la cache
  mutando una risposta;
- **gli errori (4xx/5xx) non vengono mai messi in cache**: vale il normale
  flusso throw/retry.

### I gruppi API e i loro endpoint

Il client esporta oggetti tipizzati per dominio. Ogni endpoint sotto è
verificato contro le rotte del backend (`backend/cmd/server/main.go`).

| Gruppo | Metodi | Endpoint |
|---|---|---|
| `authApi` | login, register, me, updateProfile, changePassword | `POST /auth/login`, `POST /auth/register`, `GET /users/me`, `PATCH /users/me`, `POST /users/me/password` |
| `portfolioApi` | list, create, get, update, delete | `GET /portfolios`, `POST /portfolios`, `GET/PATCH/DELETE /portfolios/{id}` |
| | summary, allocation, classAllocation, geographyAllocation, sectorAllocation, allocationDrill, performance, performanceBuckets, roi, history | `GET /portfolios/{id}/summary`, `GET /portfolios/{id}/allocation`, `GET /portfolios/{id}/allocation/class`, `GET /portfolios/{id}/allocation/geography`, `GET /portfolios/{id}/allocation/sector`, `GET /portfolios/{id}/allocation/drill?dim=class\|country\|region\|sector&key=<bucket>`, `GET /portfolios/{id}/performance`, `GET /portfolios/{id}/performance/buckets?granularity=month|year`, `GET /portfolios/{id}/roi`, `GET /portfolios/{id}/history` |
| | dashboard, dashboardAllocation, dashboardAllocationDrill, dashboardPerformance | `GET /dashboard`, `GET /dashboard/allocation`, `GET /dashboard/allocation/drill?dim=&key=`, `GET /dashboard/performance?granularity=month|year` |
| | exportDoc, importDoc | `GET /portfolios/{id}/export`, `POST /portfolios/import` |
| `assetApi` | list, search, lookup, meta | `GET /assets`, `GET /assets/search?q=`, `GET /assets/lookup?q=`, `GET /assets/meta?ticker=` |
| | get, create, update, remove | `GET /assets/{id}`, `POST /assets`, `PATCH /assets/{id}`, `DELETE /assets/{id}` |
| | quote, fetchProfile | `GET /assets/{id}/quote`, `POST /assets/{id}/fetch-profile` |
| | exposure, saveExposure, fetchExposure, fetchETFExposure, fetchMorningstarExposure | `GET /assets/{id}/exposure`, `PUT /assets/{id}/exposure`, `POST /assets/{id}/fetch-exposure`, `POST /assets/{id}/fetch-etf-exposure`, `POST /assets/{id}/fetch-morningstar-exposure` |
| | backfillHistory, sync | `POST /assets/{id}/backfill-history`, `POST /assets/sync` |
| `transactionApi` | list, create | `GET /portfolios/{id}/transactions?limit=&offset=&type=&asset_id=&from=&to=` (restituisce l'involucro `TransactionPage` — `transactions`, `total`, `limit`/`offset` applicati; limite di default 20, max 100, ordine per data decrescente; i filtri opzionali e combinabili `type` (buy/sell/dividend/split/fee), `asset_id` (uuid) e i limiti `from`/`to` `YYYY-MM-DD` inclusivi restringono le righe **e** il `total` restituito), `POST /portfolios/{id}/transactions` |
| | update, remove | `PATCH/DELETE /transactions/{id}` |
| `pricesApi` | refresh | `POST /prices/refresh` (query opzionale `portfolio_id`, restituisce un 202 `JobEnqueued`; poi `jobsApi.wait`) |
| | byAsset | `GET /prices/{assetId}?full=1` |
| `jobsApi` | get, list, wait | `GET /jobs/{id}`, `GET /jobs?limit=&offset=`, e `wait` (polling fino allo stato `done`/`failed`/`partial`; usato dal refresh prezzi e dai backfill degli asset) |
| `settingsApi` | listCurrencies, addCurrency, deleteCurrency | `GET/POST /settings/currencies`, `DELETE /settings/currencies/{code}` |
| `backupApi` | download, restore | `GET /backup` (bundle JSON servito come allegato), `POST /backup/restore?mode=add\|replace` (bundle nel body, riepilogo dei conteggi in risposta) |
| `adminApi` | listUsers, updateUser, resetPassword, getSettings, updateSettings, dbBackup, dbRestore | `GET /admin/users`, `PATCH /admin/users/{id}`, `POST /admin/users/{id}/reset-password`, `GET/PATCH /admin/settings`, `GET /admin/db/backup` (allegato `.dump` in streaming), `POST /admin/db/restore` (multipart `dump` + `confirm=replace`) (tutti riservati agli amministratori: 403 per gli altri ruoli) |
| `api` (generico) | get/post/put/patch/delete | il client grezzo, usato dalla pagina health per `GET /health/prices` |

I tipi esportati accanto (`User`, `Portfolio`, `Asset`, `Transaction`,
`TransactionPage`, `PortfolioSummary`, `AssetHolding`, `Dashboard`,
`DashboardSummary`,
`ActiveBreakdown`, `ClosedBreakdown`, `Job`, `JobEnqueued`, `AssetQuote`,
`AssetExposure`, `PortfolioHistory`, `AssetPositionSeries`,
`PortfolioExportDocument`, ...) rispecchiano i modelli del backend. I
valori monetari arrivano come **stringhe** (es. `"1234.56"`) per evitare errori
di arrotondamento in virgola mobile; le pagine li convertono con `Number()`
dove serve. `User` porta la preferenza `base_currency` (default
`"EUR"`) e `Dashboard` porta `base_currency` più l'eventuale `summary`
(`DashboardSummary`) con i totali consolidati in quella valuta. Il summary e
ogni voce di `portfolios` (`PortfolioPerformanceSummary`)
scompongono i totali nei due oggetti annidati `active` (`ActiveBreakdown`:
investito, valore, gain/loss, gain/loss % e dividendi dei soli lotti ancora
detenuti) e `closed` (`ClosedBreakdown`: investito = costo dei lotti venduti,
ricavi = ricavi netti di vendita + dividendi delle posizioni completamente
chiuse, realizzato = ricavi − investito, realizzato %); i totali vivono solo in
questi oggetti annidati. Il payload `Dashboard` non porta una serie `history`
per portafoglio: i tipi `DashboardPerformance` / `PerformanceBucket`
alimentano il grafico "Performance" e il grafico "valore vs investito"
dell'hero nella dashboard tramite `dashboardPerformance(granularity)` (`GET /dashboard/performance?granularity=month|year`,
bucket `YYYY-MM` o `YYYY` nella valuta base dell'utente). La
**stessa** forma `DashboardPerformance` alimenta anche la card "Performance"
del dettaglio portafoglio tramite `performanceBuckets(id, granularity)`
(`GET /portfolios/{id}/performance/buckets?granularity=month|year`), nella
**valuta del portafoglio** e non in quella base. Ogni bucket porta
`return` (il rendimento TWR % del periodo, barre), `twr` (il
rendimento time-weighted cumulato %, linea), `invested` (capitale netto
investito a fine bucket) e `value` (valore di mercato a fine bucket) — i due
importi nella `currency` del payload. `Dashboard` porta anche
`invested_assets` (`InvestedAsset[]`): le posizioni aperte aggregate su tutti
i portafogli nella valuta base (`ticker`, `name`, `invested`, `value`,
`gain_loss`, `gain_loss_pct`, `has_price`), ordinate per valore decrescente —
 le righe con `has_price: false` portano il valore al costo, quindi il loro
P/L è 0. `transactionApi.list(id, { limit, offset })` restituisce l'involucro
`TransactionPage`
(`transactions`, `total`, `limit`/`offset` applicati; limite di default 20,
max 100, ordine per data decrescente) invece di un semplice array, e il dettaglio
portafoglio lo pagina; accetta anche i filtri opzionali `type`, `asset_id`,
`from` e `to` (ognuno omesso se non impostato); `total` è allora il conteggio
*filtrato*.

---

## 6. Utility e risorse di formattazione

### `lib/format.ts`

Il modulo di formattazione unico, condiviso da tutte le pagine (non esiste una
cartella `utils/` o `metrics/` — vedi sotto):

| Export | Cosa fa |
|---|---|
| `currencySymbol(code)` | restituisce il simbolo di una valuta da una piccola tabella (`USD → $`, `EUR → €`, `GBP → £`, `CHF → CHF`, `JPY → ¥`, ...), ripiegando sul codice stesso per quelle sconosciute |
| `formatCurrency(amount, currency='USD')` | `simbolo + toLocaleString(...)` con esattamente 2 decimali, es. `$1.234,56`. Accetta `number` o `string` |
| `formatPercent(value)` | `toFixed(2) + '%'`, es. `12,34%`. Accetta `number` o `string` |
| `formatSignedPercent(value)` | come `formatPercent` ma aggiunge un `+` esplicito sui valori positivi, es. `+3,42%` / `-1,20%`. Usato nei tooltip del grafico Performance (sia il `return` di periodo sia il `twr` cumulato sono percentuali in cui il segno veicola il significato). Accetta `number` o `string` |
| `ASSET_TYPES` / `ASSET_CLASSES` | elenchi di valori (sola lettura) offerti dai selettori Tipo/Classe: `stock, etf, bond, mutual_fund, crypto, commodity` / `equity, bond, commodity, currency, crypto, real_estate, mixed, other` |
| `assetTypeLabel(type)` / `assetClassLabel(cls)` / `priceSourceLabel(source)` | etichette localizzate dei valori grezzi del backend, risolte con `t()` dell'i18n così seguono la lingua dell'interfaccia (tipi `asset.typeStock` … `asset.typeCash`, classi `asset.classEquity` … `asset.classOther`, fonti `asset.priceSourceYahoo` / `priceSourceManual` / `priceSourceNone`); i valori fuori tabella ripiegano sulla stringa grezza; reattive: si aggiornano al cambio di lingua |

Gli helper di etichettatura sono usati ovunque questi valori vengano mostrati:
`assetClassLabel` dentro `ClassDonut` (le ciambelle delle classi di attività
nella card "Allocazione complessiva" della dashboard e nella sezione allocazione
del dettaglio portafoglio, che etichetta da sé le chiavi del backend) e nei
titoli del drill di allocazione di entrambe le pagine, e nel dettaglio asset
(chip d'identità, dato principale, selettore "Classe"); `assetTypeLabel` e
`priceSourceLabel` etichettano allo stesso modo le superfici Tipo e Fonte
prezzo del dettaglio asset e della modale di creazione.

### Calcoli di valore e metriche

Non esiste un modulo dedicato alle metriche: ogni pagina calcola i propri
valori derivati inline con le rune **`$derived`** di Svelte 5. I principali:

- **Dashboard** (`routes/+page.svelte`): lo stato della card Performance +
  del grafico dell'hero è un `$state` `granularity`
  (default 'month') più un singolo `$state` `perf` (una sola chiamata
  `dashboardPerformance(granularity)` alimenta **entrambi** i grafici)
  ricaricato da un `$effect` a ogni cambio del selettore (un id di
  richiesta monotònico scarta le risposte obsolete); il grafico dell'hero è
  inoltre finesttrato lato client da un `$state` `heroPeriod` bucket-driven
  (1Y = ultimi 12 / 3Y = ultimi 36 / TUTTO sui bucket mensili,
  persistito in `localStorage['peculium-hero-period']`);
  `hasMultipleCurrencies` pilota il donut "Allocation by portfolio" (valori grezzi nascosti e nota
  "valute miste" quando i portafogli usano valute diverse);
  `glClass` sceglie la classe testo verde/rosso per un guadagno o una perdita.
- **Dettaglio portafoglio** (`routes/portfolios/[id]/+page.svelte`): lo stato
  della card "Performance" replica quello della dashboard: un
  `$state` `granularity` (default 'month') più un `$state` `perf` ricaricato da
  un `$effect` a ogni cambio del selettore (un id di richiesta monotònico
  scarta le risposte obsolete), alimentato da
  `performanceBuckets(id, granularity)`;
  `regionBarRows` / `sectorBarRows` / `countryBarRows` mappano i carichi di
  allocazione sulla forma generica `ExposureBarRow` consumata da
  `ExposureBarChart`; `geoUniverseNote` /
  `sectorUniverseNote` costruiscono le didascalie di copertura "Universo
  azionario" dai `covered_value`/`excluded_value` di ciascun payload.
- **Dettaglio asset**: `chartSeries` nel tab Panoramica (`routes/assets/[id]/
  +page.svelte`) ordina le righe prezzo e `zoomStart` mappa l'intervallo
  selezionato (`RANGES`: `1M` 30 giorni, `3M` 90 giorni, `1Y` 365 giorni,
  `YTD`, `MAX` illimitato) su uno zoom in-place; i campi quote
  `change_1d/1w/1m/1y/ytd` sono renderizzati come chip delta nell'header
  della shell (etichette `1G/1S/1M/1Y/YTD` ↔ `1D/1W/1M/1Y/YTD` via
  `t()`); `sumRegions` / `sumSectors` / `sumCountries` e i rispettivi guard
  di validità (`regionsValid` / `sectorsValid` / `countriesValid`) vivono
  nella shell dell'asset (`+layout.svelte`) accanto ai dati che validano.

---

## 7. I componenti ECharts

Tutti i grafici vivono in `frontend/src/lib/components/` e usano **ECharts 5**
tramite il wrapper `svelte-echarts` (dipendenze `echarts` e `svelte-echarts`
in `package.json`).

### Il pattern di import (tree-shaking)

Ogni componente grafico segue lo stesso schema:

```svelte
<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use } from 'echarts/core'
  import { LineChart } from 'echarts/charts'          // solo ciò che serve
  import { GridComponent, TooltipComponent, DataZoomComponent } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'

  use([LineChart, GridComponent, TooltipComponent, DataZoomComponent, CanvasRenderer])

  let { series = [], currency = 'USD' } = $props()

  const options = $derived.by((): EChartsOption => ({ ... }))
</script>

<div class="h-[340px] w-full">
  {#key resolved()}
    <Chart {init} {options} theme={PECULIUM_CHART_THEMES[resolved()]} />
  {/key}
</div>
```

La chiamata `use(...)` registra solo i moduli di cui il grafico ha bisogno
(bundle più piccolo); `init` (da `echarts/core`) viene passato al wrapper
`<Chart>`, che inizializza l'istanza al mount. Le opzioni sono dichiarate come
`EChartsOption` e ricalcolate con `$derived.by`, così il grafico reagisce ai
cambiamenti di `$props`.

### Grafici dark-aware

Anche i grafici seguono il tema (capitolo 8). L'helper `resolved()` dello store
del tema dice se l'app sta disegnando in chiaro o scuro; il blocco
`{#key resolved()}` forza il wrapper a **reinizializzare** il grafico quando il
tema cambia, perché `svelte-echarts` legge la prop `theme` solo una volta al
mount. Due helper lo rendono possibile:

- `lib/chartTheme.ts` registra due temi ECharts (`peculium-light`,
  `peculium-dark`) costruiti dagli stessi token del CSS (colori di assi,
  legenda, tooltip) e li espone come `PECULIUM_CHART_THEMES`;
- `lib/chartPalette.ts` espone `resolvePalette()` (i colori serie
  `--chart-1..12`, letti dal DOM e messi in cache per tema) e
  `chartSemanticColors()` per le linee speciali (cost basis, realizzato,
  marcatori split, fetta "Other").

Le label delle torte richiedono un colore esplicito: a differenza di assi e
legenda, le label delle pie ECharts **non** ereditano il `textStyle` del tema,
quindi ogni donut imposta `label.color` dal foreground del tema e disattiva il
bordo bianco di default (altrimenti, in scuro, le label apparirebbero come testo
scuro contornato di bianco).

### I wrapper dei grafici

| Componente | Grafico | Usato per |
|---|---|---|
| `PriceChart.svelte` | **line** singola (prezzi di chiusura), asse x temporale, dataZoom `inside` + `slider` | la pagina **dettaglio asset**: storico prezzi con selettore 1M/3M/1Y/YTD/MAX. Carica sempre tutto lo storico: i selettori fanno uno **zoom in-place** (coppia `start`/`end` percentuali, `end`=100) senza ricaricare dati; uno zoom/spostamento manuale **deseleziona** il pulsante attivo e preserva la vista. Gli **split** sono disegnati come `markLine` tratteggiata viola etichettata con il rapporto (`Split 4:1`), come in `PositionChart`. Stato vuoto → "Nessun dato prezzi disponibile" |
| `PositionChart.svelte` | **tre linee**: cost basis (grigia, a scalini), market value (verde, liscia), realized (ambra) + marcatori viola per gli split | la card "Performance history" del **dettaglio portafoglio**, come **vista secondaria** sotto la card percentuale "Performance": un menu a tendina passa dal portafoglio al singolo asset. Gli split sono disegnati come `markLine` tratteggiata verticale sulla linea del valore di mercato, etichettata con il rapporto (`7:1`, `4:1`) |
| `PerformanceChart.svelte` (`lib/components/domain/`) | **combo barre + linea in percentuale** su asse x a **categorie**: una barra `return` per bucket (rendimento TWR % del periodo) colorata verde/rosso secondo il segno (semantici `positive`/`negative`, `itemStyle` per barra), linea `twr` (rendimento time-weighted cumulato %) nell'ambra semantica, asse y formattato in % e tooltip `+3,42%` (`formatSignedPercent`, niente valuta), etichette dei periodi formate per granularità (`giu 2025` / `2025`), dataZoom `inside` + `slider`, legenda `Gain/Loss` / `Cumulative`, stato vuoto "No data" | la card "Performance" della **dashboard**, alimentata da `dashboardPerformance(granularity)` (`GET /dashboard/performance`, selettore mensile/annuale con `SegmentedControl` nell'intestazione della card), e la card "Performance" del **dettaglio portafoglio**, alimentata da `performanceBuckets(id, granularity)` (`GET /portfolios/{id}/performance/buckets`, proprio selettore mensile/annuale, nella valuta del portafoglio). Le barre mostrano il rendimento generato dentro ogni mese/anno, la linea il TWR cumulato — sono percentuali pure, quindi il componente non riceve la prop `currency` |
| `CapitalChart.svelte` (`lib/components/domain/`) | **due linee** sugli **stessi** bucket a categorie: `invested` (capitale netto investito, linea a scalini `end` nel grigio semantico `costBasis`) e `value` (valore di mercato, linea liscia nel verde semantico `marketValue`), tooltip in valuta con `formatCurrency(value, currency)`, dataZoom `inside` + `slider` (o solo `inside` e canvas da 240px con la prop `compact`), legenda `Invested` / `Value`, re-init theme-aware (`{#key}`), stato vuoto "No data" | il grafico valore-vs-investito dell'**hero** della dashboard, alimentato dalla **stessa** chiamata `dashboardPerformance(granularity)` e dagli stessi bucket di `PerformanceChart` (importi nella **valuta base** dell'utente, `currency` del payload) e con lo stesso selettore mensile/annuale; l'hero passa `compact` e finestra i bucket lato client con i chip periodo bucket-driven |
| `Sparkline.svelte` (`lib/components/domain/`) | minuscola **linea senza assi**: niente legenda/tooltip/zoom, griglia ai bordi zero; accetta numeri semplici (asse indice nascosto) o punti `{date, value}` (`SparklinePoint`, asse **temporale** nascosto così i buchi di calendario restano veritieri — non mescolare le due forme), il verde semantico `marketValue` di default con override `color` opzionale, riempimento d'area discreto al 10% (`area`), `smooth` + `sampling: 'lttb'`, nessun hover (`silent`), strip d'altezza fissa via `heightClass` (default `h-10`); sotto i 2 punti **non renderizza nulla**; wrapper `role="img"` con `aria-label` (del chiamante, altrimenti `sparkline.trend`), re-init theme-aware (`{#key}`) | il fondo delle **card portafoglio** della **dashboard**: strip con lo storico del valore di mercato del portafoglio, alimentato da `portfolioApi.history(id)` (le stringhe `market_value` della serie mappate in punti `{date, value}`) caricato in background dopo il payload principale della dashboard; se la chiamata fallisce la card resta senza sparkline, in silenzio |
| `ExposurePie.svelte` | **ciambella** (raggio 45%–70%), palette a 12 colori, legenda mostrata solo con ≤ 6 righe, righe a peso zero filtrate; `complete={false}` la rende **aperta** quando le righe sommano < 100 (una fetta residua trasparente tiene veritieri gli angoli — niente fetta grigia "Other") | pagina dettaglio asset (donut regioni con `complete={false}` e donut settori) e le due modali esposizione (in modalità `mute`: regioni in `ExposureGeoModal`, settori in `ExposureSectorModal`). I paesi (pagina e modale geografica) sono liste a barre, mai una pie. Accetta `ExposureRow[]` (`{name, weight}`). |
| `ClassDonut.svelte` (`lib/components/domain/`) | **ciambella** delle classi di asset (stesso stile radius/palette/etichette di `ExposurePie`): righe `AssetClassSlice[]` (`{class, value, weight}`) mappate con `assetClassLabel` per i nomi localizzati, tooltip con importo (`formatCurrency`) e peso (`formatPercent`), fetta `other` in grigio spento, righe a peso zero scartate, stato vuoto "Nessuna allocazione per classi"; prop `label` opzionale per l'intestazione sopra il grafico; l'opzionale `onDrill?: (classKey) => void` rende le fette cliccabili — al click la invoca con la chiave classe **grezza** (il click arriva dal prop `onclick` del wrapper svelte-echarts, che ri-registra l'handler a ogni re-init `{#key}`) e le fette mostrano il cursore pointer | il pannello classi della card "Allocazione complessiva" della **dashboard**, alimentato da `dashboardAllocation().classes` (patrimonio intero, valuta base), e il pannello classi della sezione "Allocazione" del **dettaglio portafoglio**, alimentato da `classAllocation(id).classes` (valuta del portafoglio); entrambi i call site passano `onDrill` e aprono il `AllocationDrillPanel` condiviso di pagina |
| `ExposureBarChart.svelte` (`lib/components/domain/`) | **barre orizzontali** riutilizzabili su righe generiche `{name, value, weight}[]` (`ExposureBarRow`): barre ordinate **per valore decrescente** (risortese in modo difensivo nel componente, righe non positive scartate; asse categorie `inverse`, quindi la barra più grande sta in alto), peso % stampato a fine barra, tooltip con importo (`formatCurrency(value, currency)`) e peso (`formatPercent`), asse dei valori nascosto (le barre servono solo a confrontarsi tra loro), altezza del canvas proporzionale al numero di righe, `colorFor?: (name) => string` per colore per-riga (altrimenti palette `resolvePalette` per indice), `labelFor?: (name) => string` per mappare le etichette dell'asse (l'asse mostra il nome leggibile — es. codice ISO → nome completo del paese via `countryDisplayName` — e il tooltip aggiunge il nome grezzo tra parentesi quando differisce, "United States (US)"; la colonna delle etichette si allarga a 140px quando `labelFor` è attivo), `maxVisibleRows?: number` collassa il grafico a quel numero di barre con un pulsante "Mostra tutti" che espande in place (nessun viewport con scorrimento interno — la pagina è l'unico contenitore scrollabile), `label` e `note` (didascalia muted) opzionali, stato vuoto "No data", re-init theme-aware (`{#key}`); l'opzionale `onDrill?: (rawName) => void` rende cliccabili le barre — al click la invoca con il nome **grezzo** della riga (la chiave del bucket, es. `US` / `North America` / `Financials`, anche quando `labelFor` mappa l'etichetta dell'asse) tramite il prop `onclick` del wrapper svelte-echarts, e le barre drillabili mostrano il cursore pointer | i pannelli regioni, settori e paesi della card "Allocazione complessiva" della **dashboard**, alimentati da `dashboardAllocation().regions` / `.sectors` / `.countries`, e gli stessi tre pannelli della sezione "Allocazione" del **dettaglio portafoglio**, alimentati da `geographyAllocation(id).regions` / `sectorAllocation(id).sectors` / `geographyAllocation(id).countries` (in valuta del portafoglio); i chiamanti mappano `RegionAllocation`/`SectorAllocation`/`CountryAllocation` su `ExposureBarRow`; i paesi portano codici ISO alpha-2 renderizzati con `labelFor={countryDisplayName}` e `maxVisibleRows={10}` su entrambe le pagine — si vedono le ~10 barre maggiori e un pulsante "Mostra tutti" rivela le altre; i pannelli regioni e settori non passano nulla: etichette invariate e tutte le righe visibili — le ~10 macro-regioni non hanno mai bisogno del cap; entrambi i call site passano `onDrill` e aprono il `AllocationDrillPanel` condiviso di pagina |
| `AllocationDrillPanel.svelte` (`lib/components/domain/`) | **pannello drill-down di allocazione** di sola lettura: gli asset contribuenti di un bucket di allocazione, renderizzato come `ui/Drawer` a ≥ `lg` e `ui/Sheet` sotto (via lo store `viewport` — la stessa coppia drawer/sheet del form transazione). Controllato come le due primitive (`open`/`onClose` di props, lo stato `title`/`dim`/`key` resta del chiamante); `fetcher: (dim, key) => Promise<AllocationDrill>` è iniettato per scope (dashboard: `dashboardAllocationDrill`, tab portafoglio: `allocationDrill(id, …)`) e chiamato all'apertura e a ogni cambio di `dim`/`key` a pannello aperto, con request id monotònico che scarta le risposte stale (i quattro stati seguono le convenzioni `ui/AsyncCard`: skeleton / errore su una riga + Retry / vuoto "Nessun asset in questa fetta" / dati). La tabella riusa `ui/Table`/`Th`/`Td` con caption sr-only e sotto `sm` collassa in righe key–value impilate — Asset (ticker che linka a `/assets/{id}` + nome muted su seconda riga), Valore (`formatCurrency` nella valuta del payload), Peso (`formatPercent`, quota dell'asset nel bucket), Contributo (valore × peso/100 nel bucket) e Quota della fetta (contributo ÷ `total`, con guardia a trattino per i bucket vuoti); le righe mantengono l'ordine decrescente di contributo del backend e il pannello non aggiunge nessuna area di scroll annidata (scorre il corpo del drawer/sheet) | montato **una volta per pagina** dalla dashboard (card "Allocazione complessiva") e dal tab Allocazione del portafoglio: ogni `ClassDonut`/`ExposureBarChart` passa lì un `onDrill` che apre il pannello con il bucket cliccato; il `title` del drill è l'etichetta che il grafico mostra (nome classe via `assetClassLabel`, paese esteso via `countryDisplayName`, nome verbatim per regioni/settori) |
| `InvestmentsTable.svelte` (`lib/components/domain/`) | tabella **active/closed** condivisa (`active: ActiveBreakdown`, `closed: ClosedBreakdown`, `currency`, `title` opzionale): colonne Investito / Valore-Ricavi / Gain-Loss / % / Dividendi, righe Active e Closed, P/L firmato colorato con `pnlColorClass`, importi via `formatCurrency` | il disclosure "Dettaglio" dell'hero nella **dashboard** (valuta base, dentro un `<details>` sotto il numero hero) e la card KPI del **dettaglio portafoglio** (valuta portafoglio) |
| `PositionTable.svelte` (`lib/components/domain/`) | tabella posizioni generica sul tipo `PositionRow` (`{assetId?, ticker, name?, qty?, cost?, value?, realized?, unrealized?, roi?, closed?, price?, priceCurrency?}`); `showCost`/`showRealized`/`showUnrealized` mostrano le colonne opzionali, `showPrice` aggiunge la colonna Price (prima di Qty, formattata con `priceCurrency`, visibile anche sulle righe chiuse), `linkAssets` collega il ticker alla pagina asset; le righe chiuse mostrano `-` su tutte le celle tranne il realizzato | la tabella Positions del **dettaglio portafoglio** |
| `AllocationDonut.svelte` (`lib/components/domain/`) | ciambella theme-aware di quote `{name, value}[]` (pesi ricalcolati sul totale positivo); `showValue={false}` nasconde il valore nel tooltip (donut multi-valuta) | la card "Allocation by portfolio" della **dashboard** |
| `AssetCombobox.svelte` (`lib/components/domain/`) | combobox filtrabile sugli asset già registrati (ticker/nome, max 8 righe); emette l'id dell'asset selezionato | la modale transazione. La ricerca ticker Yahoo per creare asset vive in `AssetSearchAutocomplete` |
| `TransactionTable.svelte` (`lib/components/domain/`) | tabella transazioni (Data/Asset/Type badge/Qty/Price/Total/Azioni) con azione di modifica allineata a destra | la card Transactions del **dettaglio portafoglio**; la pagina le passa una pagina da 20 righe alla volta e mostra i pulsanti Previous/Next con l'intervallo sotto di essa |
| `AddTransactionModal.svelte` (`lib/components/domain/`) | form di aggiunta/modifica/eliminazione transazione: combobox asset, tipo (buy/sell/dividend), quantità/prezzo o importo, data, commissioni, note; validazione inline e totale live; gestisce chiamate API e toast. Si presenta come `ui/Modal` da `sm` in su e come `ui/Sheet` (bottom sheet) sui telefoni (store `viewport`), condividendo un'unica coppia di snippet form/piè; Elimina rimuove la riga subito e mostra un toast **undo** da 5 s invece del `ConfirmDialog` — l'undo re-INVIA il payload catturato, con nuovo id | la pagina **dettaglio portafoglio**, aperta da "Add Transaction" e dall'azione di modifica della tabella |
| `SettingsTabs.svelte` (`lib/components/domain/`) | barra di tab basata su link per le subroute delle Impostazioni (Profilo / Password / Preferenze / Valute / Backup e ripristino), tab attivo marcato con `aria-current="page"`; il contenitore delle pill `max-w-full flex-wrap` mantiene tutte e cinque le tab raggiungibili a larghezza telefono | tutte e cinque le pagine **Settings** |
| `ChartTableToggle.svelte` (`lib/components/ui/`) | disclosure segmentata **Grafico ⇄ Tabella** condivisa: wrapper sottile di `SegmentedControl` legato allo stato `view` interno (`'chart' \| 'table'`, `$bindable`) del grafico che lo ospita, con etichette `Chart`/`Table` da `chartView.*`; il nome accessibile della tablist interpola il titolo proprio del grafico, se ce l'ha (`chartView.ariaNamed`, altrimenti il generico `chartView.aria`) | incorporato da `ExposureBarChart`, `ClassDonut`, `ExposurePie`, `PerformanceChart`, `CapitalChart` e `AllocationDonut` (vedi la nota "Vedi come tabella" qui sotto); i chiamanti lo nascondono con `showTableToggle={false}` dove sotto al grafico è già presente un elenco delle stesse righe |

I tooltip formattano i valori monetari con `formatCurrency` (capitolo 6), le
date con `new Date(...).toLocaleDateString()`.

#### Il toggle "Vedi come tabella"

Ogni wrapper di grafico con dati incorpora il `ui/ChartTableToggle`
condiviso: la card può passare a una **`<table>` accessibile delle stesse
righe già disegnate sul canvas** — il requisito WCAG "ogni grafico offre
un equivalente tabellare", che vale anche come esperienza
dati su telefono e come percorso per screen reader. Il pattern è uniforme
nei sei wrapper:

- il toggle vive **dentro il componente**, quindi ogni call site lo riceve
  gratis; `showTableToggle={false}` lo disattiva (usato nel tab Esposizione
  del dettaglio asset, dove le ciambelle elencano già ogni riga nella
  legenda sotto al grafico, e nelle due anteprime `mute` dentro le modali
  esposizione, la cui griglia di pesi editabile *è* quella tabella);
- **prima il grafico**: il rendering di default non cambia; gli stati
  vuoti hanno la precedenza sul ramo tabellare, quindi non viene mai
  resa una tabella vuota e il toggle stesso non appare senza dati;
- in vista tabella il canvas viene **smontato** (esce dall'albero di
  accessibilità e si ridisegna da capo al ritorno);
- le tabelle riusano le primitive `ui/Table`/`THead`/`TBody`/`Tr`/`Th`/`Td`
  con un `<caption>` `sr-only` (`chartView.caption`), intestazioni con
  `scope="col"`, celle numeriche allineate a destra in `tabular-nums` e
  gli stessi formattatori dei tooltip (`formatCurrency`, `formatPercent`,
  `formatSignedPercent`; la colonna del rendimento mantiene
  `pnlColorClass` come le barre verdi/rosse). Le etichette di riga
  seguono il trattamento del grafico: `ExposureBarChart` mostra il nome
  leggibile di `labelFor` con il codice grezzo tra parentesi
  ("United States (US)"), `ClassDonut` il nome restituito da `assetClassLabel`, e
  `AllocationDonut` omette la colonna dell'importo con `showValue={false}`
  (donut multi-valuta), rispecchiando il suo tooltip. Colonne per
  grafico: nome/valore/peso (barre, donut classi), nome/peso
  (`ExposurePie`), periodo/rendimento/TWR cumulativo (`PerformanceChart`),
  periodo/investito/valore (`CapitalChart`), nome/valore/peso
  (`AllocationDonut`).
- in `ExposureBarChart` il cap `maxVisibleRows` dei paesi è un controllo di
  **collasso**, non un viewport con scorrimento: il grafico
  mostra le prime `maxVisibleRows` barre più un pulsante "Mostra tutti" che
  espande in place, così la pagina resta l'unico contenitore scrollabile; la
  modalità tabella elenca sempre tutte le righe. Sotto `sm` tutte e sei le
  tabelle collassano ogni riga in una griglia chiave–valore impilata (le parti
  della tabella vengono blockificate, il nome occupa tutta la larghezza, le
  celle numeriche condividono la seconda riga) così non introducono mai una
  scrollbar orizzontale a 393px; da `sm` in su la tabella classica è
  `table-fixed w-full` (il contenuto va a capo nelle celle) quindi non può mai
  superare la card su nessun browser.

`PriceChart`, `PositionChart` e le `Sparkline` delle card non hanno il toggle
tabellare: i primi due sono lo strumento dello storico prezzi (con
proprio selettore 1M–MAX) e la vista secondaria di supporto
sotto la card percentuale; la sparkline è per definizione un elemento di
sola forma i cui numeri compaiono già nella card come testo.

### Dove vengono usati

- **Dettaglio asset** — `PriceChart` nel tab
  Panoramica per lo storico prezzi (con
  zoom in-place e marcatori {@code split}); `ExposurePie` nel tab
  Esposizione per la distribuzione geo/settoriale. La **modifica**
  dell'esposizione avviene in **due modali** (`ExposureGeoModal` per
  paesi + regioni, `ExposureSectorModal` per i settori), montate una sola
  volta nella shell: sulla tab
  restano solo i grafici; il pulsante "Modifica" di ciascuna card (icona
  matita) apre la propria modale con le griglie dei pesi, la
  validazione somma=100 (regioni/settori) e i salvataggi indipendenti.
- **Dettaglio portafoglio "Allocazione"** — una griglia `lg:grid-cols-2` di pannelli che
  replica la card della dashboard, in **valuta del portafoglio**: `ClassDonut`
  sugli `AssetClassSlice[]` restituiti da `portfolioApi.classAllocation`
  (chiavi mappate con `assetClassLabel` dal componente stesso) più i
  pannelli `ExposureBarChart` di regioni, settori e paesi alimentati da
  `geographyAllocation(id)` (`regions` + `countries`) e
  `sectorAllocation(id)` (`sectors`), con le stesse didascalie
  equity-universe e lo stesso `maxVisibleRows={10}` per i paesi della
  dashboard.
- **Dashboard "Allocazione complessiva"** —
  alimentata da `GET /dashboard/allocation`, che espone quattro dimensioni:
  `classes` (`ClassDonut`), `regions`, `sectors` e `countries` (tutti e tre
  `ExposureBarChart`), disposte in una
  griglia `lg:grid-cols-2` dentro la card. Le classi coprono tutto il patrimonio;
  regioni, settori e paesi sono calcolati sull'universo **equity-only**
  (azioni sempre; ETF/fondi solo quando `asset_class` è `equity` o
  `real_estate`); bond, crypto, commodity e fondi non classificati sono
  esclusi e riportati come `covered_value` / `excluded_value`, che le barre
  della dashboard trasformano in una didascalia muted
  "Universo azionario: X% del portafoglio" (mostrata solo quando qualcosa è
  stato escluso).
- **Dashboard** — `PerformanceChart` (rendimento in %: barre + linea TWR
  cumulata) sulla card di zona B e `CapitalChart` (investito vs valore,
  variante compatta dell'hero) nell'hero, entrambi alimentati da
  una **singola** chiamata `dashboardPerformance(granularity)`
  (selettore mensile/annuale; l'hero inoltre finestra i bucket lato
  client), più i widget della card "Allocazione
  complessiva" (donut classi, barre regioni/settori/paesi) e una
  `Sparkline` sul fondo di ogni card portafoglio, alimentata da una
  chiamata `portfolioApi.history(id)` di background per portafoglio emessa
  dopo l'arrivo del payload della dashboard.
- **Dettaglio portafoglio** — il medesimo `PerformanceChart` condiviso nella
  card "Performance", alimentato da
  `performanceBuckets(id, granularity)` con un proprio selettore
  mensile/annuale (bucket nella valuta del portafoglio), renderizzato sopra la
  vista secondaria "Performance history" con `PositionChart`.

---

## 8. Stile, design system e dark mode

La UI si basa su un piccolo design system interno.

### Token semantici

I colori non sono scritti direttamente nelle pagine. `tailwind.config.js`
definisce un insieme di token di colore **semantici** — `background`,
`foreground`, la **scala di elevazione** `surface-0..3` (+ gli alias legacy
`surface` = `surface-1` e `surface-raised` = `surface-2`), `muted`,
`muted-foreground`, `border`, `input`, `ring`, `accent` (+
`accent-hover`/`accent-foreground`/`accent-text`), `positive`, `negative`,
`warning`, `info` (+ `info-foreground`, per gli stati informativi come
l'aggiornamento dei prezzi), `overlay`, `chart-1..12`, `chart-muted`, `chart-grid` —
mappati su custom property CSS definite in `app.css` (`:root` e `.dark`).
Poiché i valori sono terne HSL composte con
`hsl(var(--token) / <alpha-value>)`, i modificatori di opacità funzionano
(`bg-accent/10`).

- **Scala di elevazione**: quattro superfici semantiche —
  `surface-0` (sfondo app), `surface-1` (card), `surface-2` (raised/drawer),
  `surface-3` (popover/tooltip). Il tema scuro le separa con una scala di
  luminosità di ~6–8 punti per gradino; il tema chiaro mantiene bianchi i gradini 1–3 e
  affida la separazione alla rampa di ombre.
- Raggi: `rounded-card` / `rounded-control`; elevazione: `shadow-card` /
  `shadow-raised` / `shadow-popover`; focus coerente: la classe `.focus-ring`.
- **Token di motion**: durate `duration-fast` (120 ms),
  `duration-base` (200 ms), `duration-slow` (320 ms) e un'unica curva ease-out
  `ease-standard`; `app.css` disattiva transizioni e animazioni sotto
  `prefers-reduced-motion: reduce`.
- **Scala tipografica**: default di Tailwind più i gradini nominati
  `text-hero` (40 px semibold, per i KPI della overview) e `text-micro`
  (etichette da 11 px).
- **Font**: **Inter** (UI) e **JetBrains Mono** (ticker, ISIN,
  codici) sono **self-hosted** via `@fontsource/inter` (400/500/600/700) e
  `@fontsource/jetbrains-mono` (400/500), importati in
  `routes/+layout.svelte` — nessun CDN, `font-display: swap`.
  `fontFamily.sans` inizia con Inter e `fontFamily.mono` con JetBrains Mono
  (fallback di sistema mantenuti), quindi `font-mono` applica lo stack mono
  ovunque sia usato.
- Tailwind è caricato tramite `app.css` (le tre direttive `@tailwind`) e
  PostCSS (`postcss.config.js`: `tailwindcss` + `autoprefixer`).
- `lib/chartTheme.ts` replica i token per la canvas (ECharts non risolve le
  variabili CSS), incluso `--chart-grid`: le split line degli assi sono
  dipinte con quel colore a ~8% di opacità, così i dati restano l'elemento
  più luminoso.
- `lib/ui-colors.ts` centralizza i colori testo di P&L (`pnlColorClass`,
  `totalColorClass`). Poiché ogni
  superficie di P&L consuma i token `positive`/`negative`, la palette CVD
  opzionale qui sotto ri-veste testi e grafici senza toccare un singolo
  componente.
- **Palette CVD**: swap opt-in della coppia
  utile/perdita dal verde/rosso a una blu/arancione per persone con
  deficit di visione dei colori (derivata Okabe–Ito; chiara
  `#0072b2`/`#c2410c`, scura `#56b4e9`/`#fb923c`, tutte con contrasto testo
  ≥ 4.5:1 nel proprio tema). `app.css` aggiunge gli override
  `html.cvd` / `html.cvd.dark` di `--positive`/`--negative` (specificità
  scelta per battere sia `:root` sia `.dark`); la classe viene dipinta prima
  del primo paint dallo script di `app.html` e mantenuta in sync da
  `lib/stores/palette.svelte.ts` (stato `palette` con `cvd`, `setCvd()`,
  storage key `peculium-cvd`, listener cross-tab — stesso pattern dello
  store del tema). I grafici la ricevono via `lib/chartPalette.ts`
  (`CHART_SEMANTIC_COLORS_CVD` + `chartSemanticColors()` reattiva), e
  l'unico componente che dipinge barre positive/negative —
  `PerformanceChart` — estende il suo `{#key}` con la palette, così uno
   switch la re-inizializza. Segni e glifi ▲▼ restano in ogni caso (sono la
   garanzia di indipendenza dal colore).

### Dark mode

- **Il default è Sistema (segue l'OS)**; chiaro e
  scuro sono temi first-class, progettati allo stesso modo. L'utente può
  sovrascrivere con **Chiaro**, **Scuro** o **Sistema** dal selettore del
  tema nell'header.
- La scelta è salvata in `localStorage` (`peculium-theme`) ed è gestita da
  `lib/stores/theme.svelte.ts` (`theme`, `resolved()`, `setThemeMode()`,
  `DEFAULT_MODE = 'system'`); è sincronizzata tra le schede e segue i cambi
  dell'OS in modalità `system`.
- Uno script inline in `app.html` imposta la classe `.dark` **prima del primo
  paint**, risolvendo il `prefers-color-scheme` dell'OS quando non c'è nulla
  di valido salvato, così un reload non mostra mai il tema sbagliato (niente
  FOUC). `darkMode: 'class'` nella config di Tailwind fa sì che una sola
  classe cambi tutti i token. Un secondo script inline applica allo stesso
  modo la classe opzionale `cvd` da `localStorage['peculium-cvd']`, così
  nemmeno la palette CVD (sopra) mostra mai il verde/rosso; `html.cvd`
  deliberatamente non dipende dal tema dipinto — la coppia ha varianti
  chiara e scura.

### Primitive UI

I componenti riusabili vivono in `src/lib/components/ui/`: `Button` (varianti
primary/secondary/outline/ghost/danger/link, dimensioni, loading), `Input`,
`Textarea`, `Select`, `Field`, `Card` (+ `CardHeader`/`CardContent`), `Badge`,
`Modal`, `ConfirmDialog`, `Spinner`, `Skeleton`, `EmptyState`, le primitive
`Table` (`Table`/`THead`/`TBody`/`Tr`/`Th`/`Td`), `SegmentedControl`,
`Switch` (un toggle `role="switch"`: il booleano resta del chiamante via la
prop `checked` e `onchange` riceve il nuovo valore, così una pagina di
impostazioni può ritrarre una scrittura ottimistica in caso di errore) e
`StatCard`. Le pagine e la shell le riusano invece di duplicare markup.
`SegmentedControl` e sicuro contro l'overflow per costruzione:
la riga delle pill `max-w-full flex-wrap` con segmenti `flex-auto` basati sul
contenuto, quindi le etichette lunghe vanno a capo dentro il contenitore a
larghezza telefono invece di generare scroll orizzontale; da `sm` in su la riga
inline-flex si restringe comunque alle etichette su una sola riga (aspetto
desktop invariato). Le
azioni distruttive irriducibili usano `ConfirmDialog` al posto del `confirm()`
nativo del browser (le eliminazioni di transazioni sono
esclusi: le azioni reversibili passano prima dal toast undo —
vedi capitolo 10).

Sei primitive costruite sugli
stessi token:
`PnlValue` (il renderer canonico di guadagno/perdita — segno esplicito +
glifo ▲▼ + colore semantico, zero neutro, "positive/negative" solo per
screen reader), `AsyncCard` (stati loading/errore/vuoto/dati per
singola card, con skeleton di forma compatibile ed errore isolato su una riga
+ Retry), `PeriodChips` (selettore di periodo compatto con semantica
radiogroup e navigazione con i tasti freccia, da posizionare sul grafico),
`Drawer` (drawer di ispezione laterale destro, focus-trap + Esc/backdrop +
ripristino; usato a ≥ `lg`), `Sheet` (bottom sheet con handle di trascinamento,
stessa API; usato a < `lg`) e `Tabs` (tablist ARIA di `<a>` reali con focus
roving, per le sottopagine-entità). Il focus-trap condiviso delle
overlay e le transizioni sui token di motion sono estratti in
`ui/focus-trap.ts` e `ui/transitions.ts` (`Modal`/`MobileDrawer`
mantengono le loro ricette inline).

### La shell dell'app

`src/lib/components/layout/` contiene la **shell adattiva**. Le tre classi di
dispositivi sono guidate da `lib/stores/viewport.svelte.ts`, un piccolo store
reattivo su `matchMedia` che espone `isPhone` (< 640), `isTablet` (640–1023) e
`isDesktop` (≥ 1024); le classi CSS della shell usano i confini `sm`/`lg` di
Tailwind (gli stessi 640/1024px), quindi stato JS e CSS non divergono mai.

- **Desktop (≥ `lg`)** — `AppShell` (radice, `h-dvh` + skip-link)
  rende la `Sidebar` espandibile (240px ⇄ rail di icone da 64px, stato
  persistito in `localStorage['peculium-sidebar']`), l'`AppHeader` sticky con il
  toggle di collassamento e il controllo di aggiornamento dei prezzi, e lo
  `UserMenu` più il `VersionChip` di build nel footer della sidebar.
- **Tablet (`sm`–`lg`)** — la stessa sidebar forzata a **rail di icone da
  64px** (`AppShell` passa `collapsed={true}`; la preferenza di espansione
  persistita vale solo da `lg` in su). Nessun hamburger e nessuna barra
  inferiore: la navigazione (main, Dati e sincronizzazione, Impostazioni) e il
  menu utente + chip versione nel footer del rail restano raggiungibili
  attraverso il rail;
  l'header conserva il controllo di aggiornamento dei prezzi, il trigger del
  pannello comandi e il selettore tema.
- **Telefono (< `sm`)** — nessuna sidebar: una `BottomNav` fissa (Panoramica ·
  Portafogli · Asset · Altro) più un `Fab` ancorato sopra di essa
  che apre la `QuickActionSheet` (Aggiungi transazione → al portafoglio unico
  quando è univoco, altrimenti a `/portfolios`; Aggiungi asset → `/assets`;
  Aggiorna prezzi → `POST /prices/refresh` con toast di esito; *Inserisci
  prezzo* è una voce disabilitata con "In arrivo"). La
  voce "Altro" apre il `MobileDrawer` (focus trap + Esc/backdrop +
  chiusura alla navigazione), che rende la navigazione
  `Sidebar`; il controllo di aggiornamento dei prezzi, tema e account restano
  nell'header. A `<main>` viene riservato uno
  spazio extra in basso e la barra rispetta `env(safe-area-inset-bottom)`;
- La **versione di build** è mostrata da `layout/VersionChip.svelte` in due
  pesi, entrambi alimentati dallo store a livello di modulo `appVersion` (una
  sola `GET /api/v1/version` pubblica, fatta una volta dallo shell): un chip
  mono statico e non focalizzabile (`versione · ora di build locale`) nel footer
  della sidebar — il rail da 64px mostra la sola versione — stampato anche nella
  pagina di login, e una card di dettaglio "Informazioni sull'app" in fondo a
  Impostazioni → Preferenze con il commit (link a GitHub), l'ora di build UTC e
  un pulsante per copiare le informazioni. Quando il backend non espone i
  metadati il chip semplicemente non viene renderizzato.
  ogni target di tocco è ≥ 44px.
- **Header condensante** (tutte le misure): la shell misura lo scroll del
  contenitore scrollabile principale e commuta `condensed` oltre una soglia di
  16px; pubblica quindi l'altezza live della barra come proprietà custom
  `--app-header-h` sulla colonna scrollabile (espansa `3.5rem` = 56px,
  condensata `2.75rem` = 44px). `AppHeader` si dimensiona con
  `h-[var(--app-header-h)]` (transizione CSS sull'altezza, neutralizzata dalla
  regola globale `prefers-reduced-motion` in `app.css`) e gli header sticky
  delle shell entità (dettaglio portafoglio/asset) si impilano a
  `top-[var(--app-header-h)]` (con una `transition-[top]` analoga) così restano
  adiacenti alla barra mentre si condensa. Il default vive nel `:root` di `app.css`.
- La sezione **Admin** della sidebar renderizza da un unico punto di
  configurazione `adminItems` dentro `SidebarNav`: **"Dati e
  sincronizzazione"** (`nav.dataSync`, route `/admin/health`) è aperta a
  tutti, mentre le voci di gestione **Utenti** (`/admin/users`, `nav.users`),
  **Impostazioni server** (`/admin/settings`, `nav.serverSettings`) e **Backup
  del server** (`/admin/backup`, `nav.serverBackup`) compaiono
  solo per gli account amministratore — `isAdmin()` (da
  `lib/stores/auth.svelte.ts`) tratta il `role` di sessione `owner`/`admin`
  come equivalente ad admin. Ogni
  pagina `/admin/*` è avvolta da `AdminGate`.
- **Pannello comandi** — `layout/CommandPalette.svelte`
  è montato una sola volta nell'`AppShell`, sul tier dei modali (z-40, sotto i
  toast a z-50). L'accordo ⌘K/Ctrl+K è un handler `<svelte:window>` dentro il
  componente (registrato solo dentro il gate di autenticazione, quindi il
  Login non ne è toccato); l'`AppHeader` ospita il trigger a ogni misura —
  solo icona sotto `lg` (l'accesso alla ricerca sui telefoni: nessun quinto
  elemento nella bottom nav) e pill etichettato con l'accordo della piattaforma
  (`⌘K`/`Ctrl K`) da `lg` in su. Lo stato aperto `$bindable` è di proprietà
  della shell; Esc, il click sul backdrop e i cambi di rotta chiudono il
  pannello, e il `ui/focus-trap.ts` condiviso riporta sempre il focus al
  trigger. Dialog → un solo input `role="combobox"`
  (`aria-expanded`/`aria-controls`/`aria-activedescendant`, l'input è
  l'unico tab stop; le opzioni sono righe `role="option"` deliberatamente non
  focusabili, pattern APG) su una `role="listbox"` raggruppata con tre sezioni
  `role="group"` rese solo se non vuote: **Vai a** (Panoramica, Portafogli,
  Asset, Dati e sincronizzazione, Impostazioni + le cinque sottosezioni, più
  Utenti, Impostazioni server e Backup del server per gli account
  amministratore, poi
  ogni portafoglio da `portfolioApi.list()`), **Asset** (gli asset registrati
  da `assetApi.list()` — nome più hint col ticker — seguiti da una riga live
  "Cerca su Yahoo …" alimentata da un `assetApi.lookup()` con debounce di
  300 ms da 2 caratteri in su; la selezione naviga semplicemente a `/assets`,
  la creazione resta fuori scope) e **Azioni** (Aggiungi transazione — la
  stessa scorciatoia a portafoglio unico del `Fab` —, Aggiorna prezzi —
  `refreshPrices()` condiviso dello store + i toast `quickActions.*` —, Cambia tema — cicla
  chiaro → scuro → sistema sullo store del tema —, Attiva/disattiva palette
  CVD — `setCvd` — e Mostra/nascondi barra laterale, solo su desktop, che
  pilota lo stato `collapsed` della shell). La corrispondenza è un matcher
  locale senza dipendenze (prefisso > sottostringa > sottosequenza su un
  haystack minuscolo `label + hint + keywords`); una riga "Nessun risultato"
  copre lo stato vuoto. Le liste si caricano in modo pigro solo all'apertura
  (cap di 8 per sezione dinamica a query vuota) — la cache GET di 60 s in
  `services/api.ts` assorbe le riaperture ravvicinate e ogni mutazione la
  invalida, quindi il pannello si autoaggiorna senza hook dedicati; gli error
  restano silenziosi e le sezioni statiche rimangono utili. Tastiera: ↑/↓ con
  wrap, Home/End, Invio (con guardia IME) esegue la riga attiva, Esc chiude;
  la riga attiva segue lo scroll con `scrollIntoView({ block: 'nearest' })`.
  Il motion è un solo fade del backdrop (`backdropFade()`, 0 ms con
  `prefers-reduced-motion`). Tutta la copy passa dal gruppo dizionario
  `commandPalette.*`; destinazioni e azioni riusano le chiavi `nav.*`,
  `settingsTabs.*`, `quickActions.*`, `theme.*` e `preferences.palette*`
  esistenti, e le sezioni dinamiche anticipano negli hint lo stato target dei
  toggle (tema successivo, variante di palette).

`ScopeSwitcher` è un componente
`domain/` (consuma il payload della dashboard, non lo stato della shell).
Il controllo di aggiornamento dei prezzi e la `DataQualityStrip`
vivono nello shell/header (`PriceRefreshButton`, `DataQualityStrip`) così da
essere visibili su ogni pagina.

### Icone, toast e lingua

- **Icone**: `lucide-svelte`. Esempi: `LayoutDashboard`, `Briefcase`,
  `Banknote`, `Settings`, `LogOut`, `PanelLeft` (shell app); `Plus`, `Trash2`,
  `Pencil`, `Download`, `Upload`, `Search`, `Loader2`, `EllipsisVertical`,
  `X`, `ExternalLink`, `Activity` (pagine); `CheckCircle2`, `XCircle`,
  `AlertTriangle` (toast).
- **Toast**: un piccolo store a rune in `lib/stores/toast.svelte.ts`
  (`toast.success/error/warning`, più `toast.dismiss`) aggiunge elementi che si
  chiudono da soli dopo 3,5 s (4,5 s per i warning); `lib/components/Toaster.svelte`
  mostra la pila fissa in alto a destra con card **theme-aware**
  (`surface-raised`), icone semantiche, pulsante di chiusura e `aria-live`
  (`role="alert"` per gli errori). Ogni chiamata accetta anche
  un oggetto opzioni `{ duration?, action? }`; una `action`
  (`{ label, onclick }`) mostra un pulsante inline raggiungibile da tastiera
  nella card che esegue il handler e chiude il toast — il meccanismo dietro
  l'**undo** dell'eliminazione transazione. `<Toaster />` è
  montato una volta in `routes/+layout.svelte`, quindi ogni pagina può fare
  toast.
- **Responsive / mobile-first**: flex e grid si adattano per breakpoint
  (`flex flex-col gap-4 md:flex-row`, `grid grid-cols-2 md:grid-cols-4`,
  `sm:grid-cols-2 lg:grid-cols-3`, `md:grid-cols-3 lg:grid-cols-6`), le tabelle
  lunghe sono avvolte in `overflow-x-auto`, e la shell è adattiva: rail di
  icone sui tablet, bottom nav + Fab per le azioni rapide sui telefoni (vedi
  "La shell dell'app" sopra).
- **App-wide**: `app.html` parte con `lang="it"` (il default dell'i18n —
  vedi la nota lingua sotto — e aggiornato a runtime dal locale
  persistito), la favicon `/peculium.svg`, i meta `theme-color` chiaro/scuro
   e il bootstrap del tema pre-paint; background/foreground del body
   arrivano dai token via `app.css`.
- **Nota sulla lingua (i18n)**: le traduzioni
  girano su un layer leggerissimo senza dipendenze esterne in
  `src/lib/i18n/`. Lo store a rune `index.svelte.ts` esporta
  `SUPPORTED_LOCALES` (`['it', 'en']`), `DEFAULT_LOCALE = 'it'`, il
  `locale` reattivo (`locale.current`), `setLocale()` (valida, persiste in
  `localStorage['peculium-locale']`, sincronizza `<html lang>`, listener
  cross-tab — rispecchia lo store del tema) e `t(key, params)` che legge il
  locale reattivo così i componenti si ri-renderizzano al cambio; i
  segnaposto `{name}` sono interpolati da `params`. I dizionari sono
  oggetti annidati a due livelli — `en.ts` è la forma canonica
  (`Dictionary`), `it.ts` è verificato con `satisfies Dictionary` quindi
  una chiave mancante/in più fa fallire la build; le chiavi vengono
  appiattite in lookup dot-joined (`nav.dashboard`) e tipizzate come
  unione `MessageKey`, così anche i siti di chiamata `t()` sono controllati
  alla compile-time. Ordine di ricerca: locale attivo → inglese
  (fallback) → la chiave stessa, con warning su console solo in dev (mai
   un'eccezione). **La copy dell'interfaccia è su `t()` in tutta l'app**: la
   chrome della shell (`SidebarNav`, etichette di
   `AppHeader`/`UserMenu`/`ThemeToggle`, `SettingsTabs`, `MobileDrawer`,
   skip link), la dashboard, il dettaglio portafoglio
   (Panoramica/Posizioni/Attività e le relative tabelle), le liste portafogli e
   asset, i tab del dettaglio asset, tutte le modali (crea portafoglio/asset,
   importa portafoglio, aggiungi transazione, modifica esposizione), le pagine
   Impostazioni, la pagina Stato sincronizzazione prezzi, le pagine admin
   Utenti, Impostazioni server e Backup del server e le superfici
   allocazione/esposizione (`allocation.*`, `exposure.*`, il drill-down
   `drill.*`, le serie e gli stati vuoti dei grafici `chartView.*`, il
   `ProvenanceBadge` `provenance.*`) — più le etichette di tipo/classe/fonte
   prezzo dell'asset, che `lib/format.ts` espone come funzioni localizzate
   (`assetTypeLabel`/`assetClassLabel`/`priceSourceLabel`) così che chip
   identità, quick facts, select e ciambella delle classi seguano la lingua.
   Ogni chiave è strutturalmente identica in `en.ts`/`it.ts`. Restano nella
   lingua originale: i **valori dei settori/regioni/paesi** (es. "Financials",
   "North America", "United States") e i termini finanziari standard (`Ticker`,
   `ISIN`, `ETF`, `ROI`, `P/L`, `TWR`, `Yahoo Finance`).

---

## 9. Autenticazione e sessione

### `lib/stores/auth.svelte.ts`

Uno store a rune che contiene `auth.user` e `auth.isLoading`:

- `initAuth()` — legge `access_token` da `localStorage` e, se presente, lo
  valida con `GET /users/me` (riempiendo `auth.user`). Al fallimento cancella
  entrambi i token. Chiamato una volta dal layout radice al mount.
- `login(email, password)` — `POST /auth/login`, salva la **coppia** in
  `localStorage`, imposta `auth.user`, poi
  `window.location.replace('/')` (un reload completo, voluto).
- `register(...)` — `POST /auth/register`, restituisce l'utente creato.
  **Non** effettua il login: la pagina di login mostra "Registered! You can
  now log in." (o, se l'account nasce `pending`, la variante con approvazione
  necessaria) e torna al form di accesso.
- `auth.user` trasporta il `role` di sessione (`owner`/`admin`/`editor`/
  `viewer`) e lo `status` (`active`/`pending`/`disabled`); `isAdmin()` è
  l'unico controllo client-side lato admin (`owner`/`admin` sono
  equivalenti). La chrome della shell nasconde le voci admin quando è false,
  le pagine `/admin/*` lo riverificano via `AdminGate` e il backend applica la
  stessa regola di nuovo su ogni richiesta `/admin/*` (403).
- `logout()` — cancella entrambi i token, `auth.user = null` e
  `window.location.replace('/login')`.
- `updateProfile(name, email, baseCurrency?)` — `PATCH /users/me` e aggiorna
   `auth.user`. `base_currency` finisce nel body solo se passato;
   omesso significa "mantieni il valore salvato".

### Il layout radice (`routes/+layout.svelte`)

- finché `auth.isLoading` mostra uno spinner;
- quando lo stato è pronto: gli utenti non autenticati su qualsiasi pagina
  tranne `/login` vengono reindirizzati con `goto('/login', { replaceState:
  true })`; gli utenti autenticati su `/login` vengono mandati a `/`;
- per gli utenti autenticati disegna la shell responsive
  (`lib/components/layout/AppShell.svelte`, capitolo 8) attorno al contenuto
  della pagina;
- monta `<Toaster />`;
- una volta per caricamento di pagina (flag `synced`) chiama
  `assetApi.sync()` (`POST /assets/sync`) — la task in background del backend
  che fa il backfill di storico e split per gli asset che ne sono privi. Gli
  errori vengono ignorati: le singole pagine fanno il backfill di ciò che
  serve.

### Archiviazione dei token e refresh

- Chiavi: `localStorage.access_token` e `localStorage.refresh_token`.
- L'header Bearer e la logica 401 → refresh → retry vivono in `api.ts`
  (capitolo 5).
- Al fallimento del refresh l'app non resta mai in uno stato "a metà login":
  cancella i token e reindirizza "duro" a `/login`.

### Il refresh prezzi di sessione

Lo **shell** (`AppShell.svelte`) possiede il refresh prezzi una-volta-per-sessione:
al mount — su qualunque pagina d'ingresso, deep-link inclusi — chiama
`refreshPrices()` dallo store condiviso `$lib/stores/priceRefresh.svelte`. Quello
store è il **percorso di refresh unico** dell'app (il trigger automatico dello
shell, il controllo nell'header, il Fab e la palette comandi passano tutti di
qui), quindi trigger concorrenti si de-duplicano in un solo accodamento e un
contatore `revision` condiviso permette alle pagine che mostrano prezzi di rifare
il fetch al termine.

Il refresh è asincrono: `refreshPrices()` fa POST a `/prices/refresh`, riceve un
202 `{job_id, status}` e poi `jobsApi.wait(job_id)` interroga `GET /jobs/{id}`
finché il job non raggiunge uno stato terminale. L'esito del job guida lo store
e i toast di avviso:

- `done` → successo (il toast semplice scatta solo sui trigger manuali);
- `partial` → "Yahoo Finance ha limitato le richieste: alcuni prezzi non
  aggiornati" (alcune quotazioni sono fallite a monte);
- `failed` (o polling in timeout) → esito fallito che tinge il controllo
  nell'header e compare nella strip qualità invece di un toast sul run
  automatico.

L'esito è reattivo e a livello di modulo, quindi sopravvive alla navigazione
SPA: il `finished_at` del job guida il **`PriceRefreshButton`** sempre visibile
nell'header ("Prezzi alle HH:MM", cliccabile per aggiornare le quotazioni su
richiesta), mentre l'esito parziale / fallito e i contatori `fx_missing_*`
della dashboard (rispecchiati nello store `dashboardStatus`, seminato dallo shell
così da essere disponibili anche fuori dalla dashboard) alimentano la
**`DataQualityStrip`**, resa globalmente come sottile banda sticky sotto
l'header e mostrata solo quando c'è qualcosa da segnalare.

Lo stesso schema accoda-poi-interroga sostiene il **backfill dello storico
completo** dell'asset (`backfillHistory`): quando il job è `done`/`partial` la
pagina rifà il fetch dei prezzi, e un fallimento nel recupero degli split
dentro un backfill altrimenti riuscito emerge come avviso `partial`.

---

## 10. Le pagine, una per una

### `/` — Dashboard (`routes/+page.svelte`)

Endpoint chiamati: `portfolioApi.dashboard()` e
`portfolioApi.dashboardPerformance(granularity)`. Il refresh prezzi di sessione
vive nello shell; questa pagina osserva solo il contatore `revision` dello store
e, al termine di un refresh, rifà il fetch del payload della dashboard e dei
bucket di performance (una sola chiamata che alimenta sia il grafico "valore vs
investito" dell'hero sia la card Performance). Le uniche chiamate aggiuntive
sono gli storici delle sparkline: quando il payload della dashboard
arriva, partono in parallelo e in background uno `portfolioApi.history(id)` GET
per portafoglio (non bloccanti, silenziose in caso di errore — vedi le card
portafogli qui sotto). Lo scope switcher e la checklist di primo avvio usano
dati già presenti nel payload di `dashboard()`.

La pagina si apre con la **hero** card. Le zone restanti (card
portafogli, allocazione complessiva, asset investiti) mantengono struttura e
dati, e ogni card portafoglio porta una strip sparkline dello storico del
valore.

- **Header**: titolo "Dashboard" e **`ScopeSwitcher`**
  (`domain/ScopeSwitcher.svelte`): un `<select>` nativo costruito sulla
  ricetta `ui/Select` che elenca "Tutti i portafogli (Patrimonio)" (valore vuoto,
  la pagina corrente) e una opzione per portafoglio prese dal payload.
  Selezionare un portafoglio **naviga** — `goto()` verso `/portfolios/{id}`,
  la stessa analisi a scope portafoglio — è navigazione di secondo livello,
  non un filtro sui dati di questa pagina. Renderizzato solo quando il patrimonio
  ha portafogli. (La **`DataQualityStrip`** e il controllo di aggiornamento
  dei prezzi non sono locali alla pagina: vivono nello shell/header e sono
  visibili su ogni pagina — vedi "Il refresh prezzi di sessione" sopra.)
- **Zona A — card hero**: quando la risposta porta `summary`, la pagina si apre con **un solo
  numero hero** — `summary.active.value` (valore di mercato netto delle
  posizioni aperte) via `formatCurrency` nella **valuta base** dell'utente,
  renderizzato con il token tipografico `text-hero` e
  `tabular-nums` — sotto un'unica riga P/L firmata costruita con due
  `PnlValue` (`summary.active.gain_loss` come valuta +
  `gain_loss_pct` come percentuale; segno + ▲▼ + colore) con didascalia
  "complessivo". Seguono i chip secondari muted: **Realizzato**
  (`summary.closed.realized`, via `PnlValue`), **Dividendi**
  (`summary.active.dividends`) e **Investito** (`summary.active.invested`).
  Infine la condivisa **`InvestmentsTable`** (
  righe Active/Closed in valuta base) è collocata in un **`<details>`
  "Dettaglio"** a divulgazione progressiva sotto i chip, così il riepilogo
  resta raggiungibile senza dominare la pagina.
- **Grafico hero** (colonna di destra su desktop, impilato sotto il numero
  sui telefoni): le serie **valore vs investito** — `CapitalChart` alimentato
  dagli **stessi** bucket `dashboardPerformance` della card Performance, nella
  variante opt-in `compact` (canvas da 240px, zoom rotella/trascina
  senza slider; la geometria delle card autonome resta invariata senza
  `compact`). Una riga di `PeriodChips` **finestra i bucket lato
  client**: con bucket mensili offre `1Y`
  (ultimi 12 bucket) / `3Y` (ultimi 36) / `TUTTO`; con bucket annuali vale
  solo `TUTTO` e la riga di chip si nasconde da sola. Le opzioni derivano
  dalla `granularity` effettiva del payload (non dallo stato del selettore) e
  il periodo usato resta in `localStorage['peculium-hero-period']`.
- **Zona B — card Performance**: riga di intestazione
  con titolo e `SegmentedControl` ("Monthly" / "Annual") legato allo stato
  `granularity`, e sotto `PerformanceChart` alimentato da
  `dashboardPerformance(granularity)`: una barra `return` verde/rosso per
  bucket (rendimento TWR % del periodo) più la linea `twr` cumulativa,
  entrambe formattate in percentuale (`+3,42%`). La chiamata è isolata (se
  fallisce resta lo stato vuoto "No data"), viene ricaricata a ogni cambio di
  selettore e **alimenta anche il grafico hero** — un'unica richiesta
  `dashboardPerformance` guida entrambi i grafici. In caricamento mostra uno
   `Spinner`. La card condivide la griglia a 2 colonne `lg` con il donut
   "Allocation by portfolio".
- Card **Allocazione complessiva**: una griglia
   `lg:grid-cols-2` di quattro pannelli alimentata da `dashboardAllocation()`
   (`GET /dashboard/allocation`, aggregato nella valuta base dell'utente su
   tutti i portafogli): **Classi di attività**
  (`ClassDonut` su `classes`, tutto il patrimonio) e le barre orizzontali
  **Regioni**, **Settori** e **Paesi**, solo equity (`ExposureBarChart`
   su `regions` / `sectors` / `countries`; le righe delle regioni riportano il nome della macro-regione
   invariato; le righe dei paesi portano codici ISO alpha-2
  ma sono etichettate con il **nome completo del paese** tramite
  `labelFor={countryDisplayName}` — i codici sconosciuti ripiegano sul codice
  grezzo — e il tooltip aggiunge il codice tra parentesi, es. "United States
  (US)"; il pannello paesi passa anche `maxVisibleRows={10}`, quindi si vedono
  le ~10 barre maggiori e un pulsante "Mostra tutti" rivela le altre, mentre i pannelli
  regioni e settori restano senza limite con nomi invariati — le ~10
  macro-regioni e i settori GICS entrano sempre). I pannelli a
  barre passano un `colorFor` che mette in grigio la fetta aggregata `Other`,
  come nelle ciambelle. Tutti e tre i pannelli equity a barre ricevono i
  metadati di copertura
  `covered_value`/`excluded_value` tramite la didascalia muted "Universo azionario: X% del portafoglio"
  (prop `note`, solo quando ci sono holding non azionarie escluse); se
  l'endpoint fallisce la card mostra "Allocazione non
   disponibile" (la chiamata è isolata, non blocca la pagina). Ogni pannello
   è anche un punto d'ingresso del **drill-down**: la `ClassDonut` passa `onDrill` con la chiave classe grezza
  e le tre `ExposureBarChart` lo passano con i nomi grezzi di regione,
  settore e paese; ogni click apre l'UNICO `AllocationDrillPanel` di pagina
  (drawer laterale a ≥ `lg`, bottom sheet sotto) — intitolato con
  l'etichetta che il grafico mostra (`assetClassLabel` per le classi,
  `countryDisplayName` per i paesi) e caricato al volo da
  `portfolioApi.dashboardAllocationDrill(dim, key)`, con gli asset
  contribuenti (ticker che linka alla pagina asset, valore, peso nel bucket,
  contributo e quota della fetta) nella valuta base.
- Donut **Allocation by portfolio** (`AllocationDonut`), etichettata nella
   valuta base quando disponibile; condivide la griglia a 2 colonne di
   zona B con la card Performance.
- Card **Portfolios** (nome, valuta, valore attivo + gain/loss colorato con
  `pnlColorClass`, numero di asset) da `portfolios[].active`; una
  seconda riga compatta mostra il breakdown closed del singolo portafoglio
  ("Closed: investito · ricavi · realizzato", con i ricavi che includono già
  i dividendi delle posizioni completamente chiuse), renderizzata solo se il
  portafoglio ha effettivamente venduto lotti (`hasClosedActivity`, cioè
  `closed.invested ≠ 0`), in tono muted e con il realizzato colorato via
   `pnlColorClass`. Ogni card si chiude con una **`Sparkline`**
   in fondo con lo storico del valore di mercato del
  portafoglio: quando il payload della dashboard arriva, la pagina emette in
  **parallelo e in background** uno `portfolioApi.history(id)` per ogni
  portafoglio — le card si renderizzano subito e le sparkline arrivano quando
  le risposte landano (le stringhe `market_value` della serie diventano punti
  `{date, value}` su un asse temporale nascosto). Un contatore di round
  monotònico (last-write-wins) impedisce a una risposta obsoleta di arrivare
  dopo un round più recente, e la store chiave-per-id fa sì che una risposta
  non possa mai agganciarsi alla card sbagliata; una chiamata fallita lascia
  in silenzio la card senza strip (dato decorativo — niente toast, nessuno
   spazio riservato). A scala familiare N GET parallele sono accettabili (la
   cache GET da 60s deduplica anche il round post-refresh).
- Card **Invested assets**: un'unica `Card` con
  una tabella su `dashboard().invested_assets`, una riga per ogni asset
  **aperto** aggregato su tutti i portafogli nella **valuta base**
  dell'utente. Colonne: **Asset** (ticker collegato a `/assets/{id}` con nome
  su una seconda riga muted), **Investito**, **Valore**, **Gain/Loss**,
  **P/L %**; colonne numeriche allineate a destra in `tabular-nums`
  (`Th`/`Td align="right"`, stesse primitive e stile della tabella Investments),
  celle P/L firmate colorate con `pnlColorClass` e righe nell'ordine del
  backend (valore decrescente, nessun riordino lato client). Gli asset senza
  prezzo (`has_price: false`) portano il valore al costo — mostrano un piccolo
  badge muted **no price** accanto al ticker il cui tooltip spiega che il P/L
  è 0 perché non c'è prezzo disponibile. Con payload vuoto la card renderizza
  un `EmptyState` tratteggiato ("No invested assets yet").
- **Patrimonio vuoto**: un patrimonio vuoto mostra la **checklist guidata di primo avvio**
   (`domain/FirstRunChecklist.svelte`): un'unica `Card` la cui **lista
  ordinata** accessibile accompagna ① crea un portafoglio → ② aggiungi un
  asset → ③ registra una transazione, ogni passo collegato alla pagina dove
  avviene l'azione (`/portfolios`, `/assets`, `/portfolios` — le
  transazioni vivono dentro un portafoglio; la sheet FAB copre il percorso
  ≤2 tocchi su mobile). I passi mostrano lo stato done/current/pending
  (badge accento, ✓, testo sr-only), derivato **solo** dal payload della
  dashboard (portafogli / asset investiti / importi investiti per
  portafoglio — nessuna chiamata extra). La card scompare da sola appena il
  patrimonio ha portafogli, perché il branch del dashboard normale li richiede.

### `/login` — Sign in / Register (`routes/login/+page.svelte`)

Una sola card centrata con un toggle tra **Sign in** e **Register**
(`isRegister`). La registrazione chiede nome + email + password e, al
successo, mostra un toast e torna al Sign in; il login chiama `store.login()`
che fa un redirect "duro" a `/`. Quando le impostazioni server tengono
l'approvazione automatica spenta, il toast della registrazione avvisa che un
amministratore deve approvare l'account; l'accesso su un account non attivo
viene rifiutato dal backend (403) e il form traduce i due casi (in approvazione
/ disattivato) in messaggi localizzati chiari invece dell'errore grezzo.

### `/portfolios` — Portafogli (`routes/portfolios/+page.svelte`)

Endpoint chiamati: `portfolioApi.list()`, `settingsApi.listCurrencies()`.

- Griglia di card dei portafogli (nome, valuta, descrizione, elimina).
- Form **Create Portfolio** (nome, descrizione, valuta scelta dalla whitelist).
- **Import**: input file nascosto → analizza un documento di export JSON
  (richiede `version === 1`), mostra un'anteprima (nome/valuta/numero di
  transazioni/range date) e importa in modalità **"new"** (con un nome scelto)
  o **"overwrite"** (su un portafoglio esistente); dopo un import riuscito
  chiama `assetApi.sync()` così gli asset importati ricevono il backfill dello
  storico.
- L'**Export** vive nella pagina di dettaglio (nel menu `⋯` dell'header, sotto).

### `/portfolios/[id]` — Dettaglio portafoglio (sotto-route a tab)

Struttura: la pagina dettaglio è una shell condivisa
`routes/portfolios/[id]/+layout.svelte` +
quattro tab profondamente linkabili, ognuna una propria route —
**Panoramica** `+page.svelte` (indice), **Posizioni**
`positions/+page.svelte`, **Attività** `activity/+page.svelte` e
**Allocazione** `allocation/+page.svelte`. Le tab sono URL reali, non stato
locale: condivisibili e salvabili, il pulsante indietro funziona e aprire
direttamente `/portfolios/7/activity` carica i dati della shell e
mostra la tab Attività nello stato attivo. Le azioni sul portafoglio
(export, import, eliminazione) vivono nel menu `⋯` dell'header, non in una
quinta tab.

Condivisione dei dati: **il layout possiede ogni fetch** ed espone lo stato
reattivo + le azioni (paginazione, aggiungi/modifica transazione) alle tab
tramite un **context** tipizzato di Svelte 5 (`context.ts`: `createContext` +
`PortfolioPageContext`). I membri di stato sono getter che fanno da proxy al
`$state` del layout, quindi le tab li osservano come fossero propri; le tab
non rifetchano mai, derivano soltanto i dati di vista (righe posizioni, barre
di esposizione, note di copertura) dai payload condivisi. Il layout resta
montato tra una tab e l'altra: header, dati e finestra di transazioni
sopravvivono alla navigazione.

Endpoint chiamati: `portfolioApi.get`, `.summary`,
`.performanceBuckets`, `.history`, `.classAllocation`, `.geographyAllocation`,
`.sectorAllocation`, `transactionApi.list(id, { limit, offset, type?, asset_id?,
from?, to? })`, `transactionApi.create` (undo), `assetApi.list`; il refresh
prezzi di sessione dello shell è osservato via `priceRefresh.revision` e al
termine questa pagina rifà il fetch di summary e bucket di performance; l'header
aggiunge `portfolioApi.exportDoc`,
`.delete` (menu ⋯) e riusa `ImportPortfolioModal` (import → refill completo
della shell, transazioni riportate alla prima pagina).

L'header sticky della shell (impilato a `top-[var(--app-header-h)]`, con margini/padding
negativi che rispecchiano il `px-4 lg:px-6` / `pt-4 lg:pt-6` responsivo di `<main>`):

- Riga identità: link indietro a `/portfolios`, nome del portafoglio +
  valuta (e descrizione se presente), l'azione primaria `[+ Transazione]`
  (apre il modal, disponibile su ogni tab) e il menu `⋯`: **Esporta**
  (`portfolioApi.exportDoc(id)` → download del file JSON
   `peculium-<nome>.json`), **Importa** ed **Elimina** (dialogo
   di conferma → API → toast → ritorno alla lista). All'import modal viene
   dato *questo* portafoglio come unico target di sovrascrittura (il picker
   completo resta nella pagina lista) e l'opzione "crea come nuovo" è disponibile.
- Strip KPI ("valore + P/L sempre visibili"): numero principale
  `summary.active.value` in valuta portafoglio, P/L firmato con due
  `PnlValue` e chip muted investito / realizzato / dividendi — la
  stessa composizione dell'hero del patrimonio, in versione
  portafoglio.
- Barra `ui/Tabs`: Panoramica / Posizioni / Attività / Allocazione,
  stato attivo derivato dalla route, scorribile orizzontalmente sui
  telefoni; etichette e copia dell'header/menu passano da `t()`
  (`portfolio.*`).

TAB **Panoramica**:

- Card KPI: la condivisa `InvestmentsTable` (roll-up Active/Closed da
  `summary.active` / `summary.closed`, in valuta portafoglio) più una riga
  attenuata con il numero di asset.
- Card **Performance**: la performance percentuale del
  portafoglio, che riusa il `PerformanceChart` condiviso (barre `return`
  verdi/rosse + linea `twr` cumulata, entrambe percentuali). Una riga di
  intestazione contiene il titolo e un `SegmentedControl` ("Monthly" /
  "Annual") la cui coppia getter/setter pilota lo `$state` `granularity` di
  proprietà del layout; i bucket arrivano da
  `performanceBuckets(id, granularity)`
  (`GET /portfolios/{id}/performance/buckets`, nella **valuta del
  portafoglio**). Replica il ciclo di vita della card della dashboard: fetch
  al mount (default `month`), refill a ogni cambio del selettore con id di
  richiesta monotònico (le risposte obsolete vengono scartate), `Spinner` in
  caricamento e stato vuoto "No data" del grafico in caso di errore. Il
   fetch vive nel layout perché il refresh prezzi di sessione dello shell
   (osservato via `priceRefresh.revision`) e ogni mutazione di transazioni
   rifetchano i bucket da qualunque tab sia aperta.
- **Performance history** (vista secondaria, sotto la card percentuale): `PositionChart` con un menu a tendina per passare dal
   portafoglio al singolo asset (gli split sono disegnati sul grafico); la
   selezione è stato locale della tab.
- **Digest allocazione**: la `ClassDonut` sul payload
  condiviso delle classi (con il fallback "non disponibile" quando quello
  endpoint fallisce) più un link all'Allocazione completa.

TAB **Posizioni**: la tabella completa delle holding (`summary.holdings`,
ticker che linka a `/assets/{id}`, posizioni chiuse con badge "Closed" e
`-`) con le sue colonne/prop e la riga "No positions"
quando è vuota.

TAB **Attività**: tabella paginata (data, asset, badge del tipo, quantità,
prezzo, totale), 20 righe per pagina (`txPage`/`txLimit`/`txOffset`/`txTotal`,
di proprietà del layout, così la finestra corrente
sopravvive ai cambi di tab): la finestra viene caricata con
`transactionApi.list(id, { limit, offset, ...filtri })` e il piè di pagina sotto la
tabella — la stessa disposizione Previous/Next + intervallo "1–20 of 137"
della pagina health admin — ricarica solo le transazioni, mai l'intera
pagina. **Filtri**: la riga di filtri sopra la
tabella — `ui/PeriodChips` per il tipo (Tutte / Acquisto / Vendita /
Dividendo / Split / Commissione), un `ui/Select` sugli asset registrati nel
portafoglio (da `summary.holdings`, incluse le posizioni chiuse) e input
nativi `Dal`/`Al` per l'intervallo di date — è **stato dell'URL**: la query
è l'unica fonte di verità, il layout la interpreta (`tx-filters.ts`) e ogni
fetch delle transazioni la rispetta, quindi `total` ed etichetta intervallo
sono il conteggio *filtrato*. Cambiare filtro (`ctx.setTxFilters` →
`goto(..., { replaceState, keepFocus, noScroll })`, stessa convenzione della
navigazione da tastiera di `ui/Tabs`) riporta la finestra alla prima pagina
filtrata e ricarica solo quella; la vista è condivisibile, sopravvive a
reload e deep link (`/portfolios/7/activity?type=sell&asset=<id>&from=YYYY-MM-DD&to=YYYY-MM-DD`)
e a indietro/avanti, e lasciare la tab (href semplici, senza query) la
azzera. Un pulsante ghost **Cancella filtri** compare quando un filtro è
attivo, e un risultato filtrato vuoto confermato dal server sostituisce
tabella e piè di pagina con un `EmptyState` con la stessa azione. Etichette
dei filtri e copy clear/vuoto passano da `t()` (`activity.*`). Form di
aggiunta/modifica per **buy / sell / dividend** (il
dividendo chiede l'importo totale invece di quantità × prezzo; la quantità
viene inviata come `1`) — il form si apre come il classico
`ui/Modal` da `sm` in su e come `ui/Sheet` (bottom sheet) sui telefoni
(store `viewport`; stessi campi/validazione/totale live) e
l'eliminazione è **basata su undo**: Elimina rimuove subito
la riga e il toast di successo porta un'azione "Annulla" da 5 s che
ricrea la transazione via `transactionApi.create` col payload catturato
(nuovo id — accettabile su scala familiare);
`ConfirmDialog` è usato per eliminazione di portafoglio/asset. Il modal è montato una
volta nel layout. Dopo ogni mutazione (undo incluso) vengono rifetchati la pagina CORRENTE
delle transazioni **con i filtri attivi** (col totale; se cancellando l'ultima riga dell'ultima
pagina la finestra resta vuota, si retrocede di una pagina con l'offset
clampato al totale appena ricevuto), il summary, lo storico, i bucket della
card Performance e le allocazioni (il percorso condiviso `reloadAfterMutation`
del layout).

TAB **Allocazione** (replica la card "Allocazione complessiva"
della dashboard): una griglia `lg:grid-cols-2` di pannelli nella **valuta del
portafoglio** — **Classi di attività** (`ClassDonut` su `classAllocation()`,
chiavi delle classi mappate con `assetClassLabel` dal componente) e le
barre orizzontali **Settori**, **Regioni** e **Paesi**, solo equity
(`ExposureBarChart` sui `sectors` di `sectorAllocation()` e sui `regions` +
`countries` di `geographyAllocation()` — il pannello paesi usa
`labelFor={countryDisplayName}` e `maxVisibleRows={10}`, esattamente come la
dashboard). I pannelli equity ricevono i metadati di copertura
`covered_value`/`excluded_value` e mostrano la didascalia "Universo
azionario: X% del portafoglio" quando ci sono holding non azionarie escluse.
Ogni endpoint è isolato nel suo try/catch: in caso di errore i suoi pannelli
mostrano "non disponibile" senza bloccare la sezione né il resto della
pagina. Anche ogni pannello fa da drill: la ciambella delle
classi e i tre grafici a barre passano `onDrill` (chiave classe / regione /
settore / codice ISO, tutti grezzi) a un UNICO `AllocationDrillPanel` di
pagina (drawer ≥ `lg`, sheet sotto) il cui fetcher chiama
`portfolioApi.allocationDrill(id, dim, key)`; il pannello elenca gli asset
contribuenti nella valuta del portafoglio — ticker (che linka alla pagina
asset) + nome, valore, peso nel bucket, contributo e quota della fetta —
ordinati per contributo, esattamente come il drill della card dashboard.

### `/assets` — Asset (`routes/assets/+page.svelte`)

Endpoint chiamati: `assetApi.list()`, `settingsApi.listCurrencies()`.

- Tabella dei titoli (ticker → link al dettaglio, nome, tipo, valuta, paese,
  elimina).
- **Add Asset**: campo ticker con **autocomplete** — mentre digiti (da 2
  caratteri, debounce 350 ms) chiama `assetApi.lookup(q)`
  (`GET /assets/lookup?q=`) e mostra un menu di suggerimenti; selezionandone
  uno, il form viene arricchito con `assetApi.meta(ticker)`
  (`GET /assets/meta?ticker=`). Creazione → `assetApi.create()`.

### `/assets/[id]` — Dettaglio asset (sotto-route a tab)

Struttura: la pagina dettaglio è una shell condivisa
`routes/assets/[id]/+layout.svelte` + tre tab
deep-linkable, ciascuna una rotta propria — **Panoramica** `+page.svelte`
(indice), **Esposizione** `exposure/+page.svelte` e **Dati**
`data/+page.svelte`. Le tab sono URL reali, non stato locale: condivisibili
e salvabili, il pulsante indietro funziona, e aprire direttamente
`/assets/7/exposure` carica i dati della shell e rende attiva la
tab Esposizione.

Condivisione dei dati: il **layout possiede ogni fetch e mutazione** ed
espone lo stato reattivo + le azioni (PATCH metadati, refresh Yahoo,
backfill, eliminazione, apertura delle modali di edit) alle pagine tab
tramite un **context** tipizzato Svelte 5 (`context.ts`: `createContext` +
`AssetPageContext`). I membri di stato sono getter che fanno proxy al
`$state` del layout, quindi le tab li tracciano come propri; le tab non
rifetchano mai nulla e ri-derivano solo i dati di vista (serie del grafico,
liste di esposizione da display, tabella "Dove è detenuto"). Le modali di
edit `ExposureGeoModal`/`ExposureSectorModal` e il `ConfirmDialog` di
eliminazione sono montate **una sola volta nella shell** (stesso contratto
del modal transazioni), così le liste di edit `$bindable` in working
copy restano `$state` nativi del layout; la tab Esposizione le apre tramite
`openGeoModal`/`openSectorModal`, che ripristinano liste e badge di
provenienza dall'`exposure` salvata prima di ogni apertura. Gli helper puri
di normalizzazione delle liste (`positiveCountries`/`withoutOther`/
`sectorsList`/`capAtHundred`/`roundWeight`) vivono
in `exposure-utils.ts` accanto alle rotte, condivisi da shell e tab.

Endpoint chiamati: `assetApi.get`, `.quote`, `pricesApi.byAsset(id)`,
`assetApi.exposure(id)`, `assetApi.splits(id)`; il refresh prezzi di sessione
dello shell è osservato via `priceRefresh.revision` e rifà il fetch di
quote/prezzi.
La chiamata isolata, non bloccante e silenziosa in caso di errore
`portfolioApi.dashboard()` dietro "Dove è detenuto" (sotto) è separata. Le tab
non introducono ulteriori endpoint:
`assetApi.update`/`.meta`/`.backfillHistory`/`.remove` (usati anche per
l'eliminazione) e il set PUT/prefill/derive dell'esposizione vivono nella
shell.

L'header sticky della shell (impilato a `top-[var(--app-header-h)]`, con margini/padding
negativi che rispecchiano il `px-4 lg:px-6` / `pt-4 lg:pt-6` responsivo di `<main>`):

- Riga identità: link indietro a `/assets`, ticker (font mono) + nome,
   i chip di identità **tipo · classe · valuta · exchange**
   (`assetTypeLabel` e `assetClassLabel` da `lib/format.ts`) e il chip
   "nessun sync automatico" per fonti prezzo non-Yahoo. La riga non
   porta alcun menu azioni: le tre azioni della shell — **Aggiorna
   da Yahoo** (`assetApi.meta(ticker)` aggiorna nome/tipo/valuta/exchange;
   l'override manuale di `asset_class` vince sempre — il refresh non
   sovrascrive mai una classe diversa da `other`/vuota), **Backfill storico
   completo** (`assetApi.backfillHistory(id)` poi un `pricesApi.byAsset(id)`
   fresco — la cache GET del client è già stata svuotata dalla POST) ed
   **Elimina asset** (dialog di conferma → API → toast → ritorno a
   `/assets`) — vivono solo nella zona pericolosa del tab Dati (eseguite
   da questa layout tramite il context, spinner di busy inclusi).
 - Strip quote: il blocco "Metriche quote", mostrato nell'header sempre
   visibile — ultima chiusura in primo piano nella valuta **dell'asset**, le
   variazioni 1G/1S/1M/1Y/YTD come chip compatti firmati (`PnlValue`) e
   la data dell'ultimo prezzo ("Aggiornato il {date}"); "Nessun dato prezzo"
   quando la quote è vuota. Un 404 in caricamento reindirizza a
   `/assets`.
 - Barra `ui/Tabs`: Panoramica / Esposizione / Dati, stato attivo
   derivato dalla rotta, scroll orizzontale sul telefono; etichette delle
   tab, link indietro e i testi dei blocchi passano da `t()`
   (`asset.*`). La copy delle card non passata da `t()` resta
   invariata.

TAB **Panoramica**:

- **Storico prezzo**: `PriceChart` con il selettore 1M/3M/1Y/YTD/MAX (zoom
  in-place + marcatori di split); la selezione
  dell'intervallo è stato locale di questa card (prezzi e splits arrivano
  dal context, quindi il refresh di sessione e ogni backfill aggiornano il
  grafico in place).
- **Dove è detenuto**: una riga per ogni
  portafoglio che attualmente detiene questo asset — nome che linka a
  `/portfolios/{id}`, quantità, costo, valore e il guadagno/perdita firmato
  + ROI nella valuta dell'asset — derivato lato client dagli `assets` per
  portafoglio di `GET /dashboard` (`portfolioApi.dashboard()`,
  `PortfolioAssets[] → AssetPerformance[]` filtrati a questo id, posizioni
  chiuse escluse), senza endpoint aggiuntivo. Il fetch è isolato e non
  blocca mai la pagina: in attesa il blocco non renderizza nulla; un errore
  lo degrada a una nota muted "non disponibile"; un risultato vuoto mostra
  la riga "Non è detenuto in nessun portafoglio".
- **Dati principali**: griglia di identità sola lettura (ISIN in mono, tipo,
  classe, valuta, exchange, fonte prezzo); la modifica vive nel tab Dati.

TAB **Dati**: il form dei metadati —
stessi campi (Ticker, ISIN, Nome, Tipo, Valuta, Exchange, Classe, selettore
**Fonte prezzo** `price_source`), stesso dirty-save (`hasChanges` abilita
"Salva modifiche"; il PATCH, il `form` `$state` condiviso e la sincronzza
`form.isin` dei prefills vivono nel layout, quindi le modifiche non salvate
sopravvivono ai cambi di tab); la **zona pericolosa** con le tre azioni
della shell (aggiorna da Yahoo / backfill / elimina); e
due segnaposto muti e disabilitati — inserimento prezzo manuale e
attributi obbligazionari — mostrati come "In arrivo"
(`quickActions.comingSoon`).

TAB **Esposizione** — i widget della distribuzione geo/settoriale:

- **Distribuzione geografica** e **Distribuzione settoriale** sono **due card
  separate**. L'editing avviene
  **solo nelle modali**; la tab mantiene la presentazione. Le card renderizzano
  sempre l'**esposizione salvata** (`displayCountries` / `displayRegions` /
  `displaySectors`, derivati dallo stato `exposure` della shell caricato/salvato via API) —
  modifiche non salvate e anteprime di prefill non compaiono mai sulle card e non
  sopravvivono nemmeno alla chiusura della modale: ogni pulsante **Modifica**
  ripristina le liste di edit e i badge di provenienza della propria modale dai
  dati salvati in `exposure` prima di aprirla (`openGeoModal` /
  `openSectorModal`), quindi riaprendo si vedono sempre i dati persistiti e le
  modifiche non salvate della sessione precedente vengono scartate:
  - La **card geografica** raggruppa due box affiancati: **Paesi** — una
    **lista a barre orizzontali dei primi 15 paesi** (peso > 0, ordinati desc,
    barra scalata sul peso maggiore, nomi amichevoli da `lib/countryNames.ts`)
    — e **Regioni** — donut `ExposurePie` **aperto** (`complete={false}`: con
    totale < 100 resta un vero varco, la fetta grigia "Other / Not Classified"
    è filtrata via) con la sua legenda sotto. Il suo
    "Modifica" apre **`ExposureGeoModal`**.
  - La **card settoriale** mostra il donut `ExposurePie` dei settori con la
    legenda sotto; il suo "Modifica" apre **`ExposureSectorModal`**.
  - **`ExposureGeoModal`** (paesi-first) ha **due colonne**
    (`lg:grid-cols-2`): **Paesi a sinistra**, **Regioni a destra** (impilati
    paesi-primo su mobile).
    - **Box Paesi**: parte come **lista vuota** (non la tabella zero-filled da
      ~89 righe). Ogni riga è `codice ISO · nome · barra orizzontale · input
      peso · elimina`, ordinata per peso **desc** (riordinata su add/remove/blur,
      mai mentre si digita — la barra si anima live così le righe non saltano).
      Il colore della barra segue la palette della card per rango. Un select
      nativo + "Aggiungi" inserisce un codice canonico non ancora presente (il
      focus passa al suo input peso). La lista è **flessibile**
      (`min-h-0 flex-1 overflow-y-auto`): cresce fino a riempire il box così il
      footer "Totale" e il pulsante Salva restano in fondo, allineati al box
      regioni, e con molti paesi scorre dentro la sua area invece di allargare
      la modale. Il footer mostra "Totale X%" + barra di
      progresso; **il salvataggio è disabilitato se la somma supera 100** (sotto
      100 è ammesso). Una nota segnala il residuo non attribuito e ricorda che
      le regioni **non** vengono ricalcolate al salvataggio: si aggiornano solo
      con "Calcola da paesi" nel box regioni. **Nessun donut** in questo box.
    - **Box Regioni**: **tabella fissa delle 10 regioni canoniche** (niente
      add/remove, riga "Other / Not Classified" esclusa — filtrata alla pagina,
      non entra mai in `regionsEdit`), ogni riga con quadratino colore, nome e
      input peso, accanto a un **donut muto APERTO** (`mute complete={false}`:
      il totale < 100 lascia un varco reale invece della fetta grigia Other).
       Footer come i paesi; **salvataggio disabilitato se la somma supera 100**
       (sotto 100 è valido).
    - **Badge di provenienza**: l'header di ogni box mostra una pillola
      `ProvenanceBadge` (puntino colorato + etichetta, con la data
      dell'ultimo aggiornamento quando la dimensione è persistita — es.
      "da Morningstar (2026-09-05)") con la fonte dei dati — `manuale`,
      `da JustETF`, `da Morningstar` / `da Morningstar (regioni ufficiali)`,
      `calcolato dai paesi`, `da JustETF via paesi`. Prefill/derive impostano
       il badge; **ogni modifica manuale lo riporta a "manuale"**. La
       provenienza è **persistita per dimensione** dal backend
      (`GET/PUT /assets/{id}/exposure` rispondono con `provenance.{countries,
      regions, sectors}` = `{source, updated_at}` solo per le dimensioni
      persistite), quindi i badge — con la loro data — sopravvivono al reload.
      Le risposte di fetch/prefill non includono provenienza (le anteprime
      non sono persistite): subito dopo un prefill o una modifica manuale il
      badge mostra **solo l'etichetta**, e la data compare quando la
      dimensione viene di nuovo salvata.
  - **`ExposureSectorModal`** ha la tabella dei settori, validata a 100 ± 0,5
    (i settori richiedono il totale esatto). Nell'header
    mostra la stessa pillola `ProvenanceBadge` dei box geografici, guidata da
    `sectorsSource` + `sectorsUpdatedAt` della shell (`da JustETF`, `da Yahoo`,
    `da Morningstar`, `manuale`): ogni prefill dei settori imposta il badge
    (solo etichetta, niente data — l'anteprima non è persistita), la prima
    modifica manuale dei pesi lo riporta a "manuale" (via `onSectorsDirty`,
    azzerando anche la data) e il salvataggio persiste la fonte con una
    `updated_at` fresca che il badge mostra a ogni reload. I
    pesi di prefill/caricamento vengono arrotondati a 2 decimali e i totali
    leggermente sopra 100 corretti all'import tramite `sectorsList` (che
    avvolge `roundWeight` + `capAtHundred`, vedi la nota sulla normalizzazione
    sotto), così il rumore di virgola dei provider (es. `21,26815…` di Yahoo)
    non inonda né la tabella né i totali.
  - I pulsanti di **prefill vivono solo nelle modali**, accanto al titolo di
    ciascuna parte (icone-favicon boxate con tooltip), posizionati dove nascono
    i dati. Sono **anteprime non persistite**: ognuno scrive solo nelle liste di
    edit della modale (`countriesEdit` / `regionsEdit` / `sectorsEdit`) —
    `exposure` (e quindi le card) continua a mostrare i dati salvati finché non
    premi il pulsante **Salva** corrispondente:
    - **paesi** (`ExposureGeoModal`): **"Prefill JustETF"** (`fetchETFExposure`,
      applica solo `countries` — JustETF fornisce la lista paesi) e
      **"Prefill Morningstar"** (`fetchMorningstarExposure`, popola `countries`
      e, nell'implementazione attuale, aggiorna anche `sectors`);
    - **regioni** (`ExposureGeoModal`): **"Calcola da paesi"**
      (`assetApi.deriveRegions` → `POST /assets/{id}/exposure/derive`, calcola
      le regioni dai paesi correnti senza salvare) e **"Prefill Morningstar"**
       (`fetchMorningstarExposure`, applica solo le **regioni ufficiali
       Morningstar** — l'insieme ufficiale, non derivate);
    - settori (`ExposureSectorModal`): **"Prefill JustETF"**
      (`fetchETFExposure`, applica solo `sectors`), **"Prefill Yahoo"**
      (`fetchExposure`, i `topHoldings` Yahoo, applica solo `sectors`) e
      **"Prefill Morningstar"** (`fetchMorningstarExposure`, applica solo
      `sectors`, solo ETF come JustETF; l'endpoint è cachato per ISIN, quindi
      se paesi/regioni sono già stati letti la chiamata è immediata).
  La **palette dei colori è condivisa**
  (`$lib/chartPalette.ts`): i quadratini colorati prima di ogni nome usano
  `colorForRow`, che restituisce esattamente il colore della fetta nel chart,
  quindi quadratino e grafico combaciano sempre. I grafici nelle modali sono
  **muti** (`mute` su `ExposurePie`: nessuna etichetta di valore né tooltip
  sulle fette).
  Il salvataggio invia **solo la dimensione modificata con la sua fonte di
  provenienza** (`PUT /assets/{id}/exposure` con `{countries,
  countries_source}` o `{regions, regions_source}` — omettere una chiave
  lascia l'altra intatta; una dimensione inviata senza fonte diventa `manual`
  lato backend), poi ricarica la risposta canonica, che rinfresca `exposure`
  (le card), risincronizza le liste di edit della modale e aggiorna il badge
  di provenienza della dimensione salvata con l'`updated_at` persistita:
   dopo un salvataggio card e modale tornano coerenti. Salvare i
   paesi **non ricalcola le regioni lato server**: le regioni memorizzate
   restano invariate e il badge di provenienza delle regioni non viene
   toccato — le regioni si ricalcolano solo cliccando **"Calcola da paesi"**
  nel box regioni. L'helper condiviso
  `withoutOther` (`exposure-utils.ts`)
  scarta la riga "Other / Not Classified" dalla risposta regioni prima di
  alimentarla alla UI; il server la riaggiunge internamente così le regioni
  persistite sommano ancora a 100 per l'aggregazione del portafoglio. La
  tab renderizza le card esposizione solo quando l'asset è azionabile per l'universo
  equity (`exposureApplicable`: stock, oppure etf/mutual_fund con
  `asset_class` `equity`/`real_estate`); altrimenti mostra il banner che
  spiega che la distribuzione vale solo per gli asset azionari.
- **Prefill da Yahoo** — `assetApi.fetchExposure(id)`
  (`POST /assets/{id}/fetch-exposure`, i pesi settoriali `topHoldings` di
  Yahoo) precompila la tabella dei settori **dentro la modale**: è un'anteprima
  non persistita, la card dei settori mostra i dati salvati finché non premi
  Salva.
- **Prefill da Morningstar** — `assetApi.fetchMorningstarExposure(id)`
  (`POST /assets/{id}/fetch-morningstar-exposure`): recupera l'esposizione
  paesi e settori da Morningstar (tramite il python-service, resolver custom con
  bootstrap Chromium headless) e la **mostra in anteprima** nella lista paesi
  della modale geografica e nella lista settori della modale settoriale;
  **nulla viene persistito** — ogni dimensione viene salvata solo premendo il
  proprio pulsante Salva. Visibile solo per asset di tipo ETF (stessa regola di
  "Carica da JustETF").
- **Carica da JustETF** — `assetApi.fetchETFExposure(id)`
  (`POST /assets/{id}/fetch-etf-exposure`): recupera dal microservizio
   JustETF la distribuzione geografica (paesi → macro-regioni canoniche, e
   i paesi raw) e i settori GICS, **precompilando in anteprima non
   persistita le liste di edit della modale** (il salvataggio avviene solo con i
   pulsanti Salva); visibile solo per asset di tipo ETF
   (`asset.type !== 'etf'` ⇒ pulsante disabilitato). Sincronizza inoltre l'ISIN
   risolto dal backend nel campo ISIN del form (l'ISIN è persistito
   lato server dal fetch).
- **Normalizzazione all'import (totali leggermente sopra 100)** — alcuni
  provider (es. JustETF su LYSX.DE) pubblicano pesi già arrotondati a 2
  decimali la cui somma è 100,01: il backend accetta fino a **100,5**
  (`weightSumMax100`), ma il guard UI blocca qualsiasi valore sopra 100 e
  l'import non sarebbe salvabile. Invece di alzare la soglia, la shell
  **normalizza all'import**: `capAtHundred` (applicato in fondo a
  `positiveCountries` / `withoutOther`, quindi su ogni prefill, derivazione di
  regioni e reload canonico) prende i totali in **(100, 100,5]** e sottrae
  l'eccesso dalla **voce a peso maggiore**, così la lista somma esattamente
  100 (a pari merito vince la prima; i pesi restano stringhe a 2 decimali).
  Totali ≤ 100 sono un no-op (load/save/display invariati); totali > 100,5
  sono un'anomalia provider reale e restano invariati, così il guard continua
   a segnalarli. **Le modifiche manuali che sforano 100 non passano da questi
   helper e restano bloccate** dal guard. Anche i settori ricevono la stessa
   correzione all'import: ogni assegnazione di settori (caricamento pagina,
   prefill del provider e reload canonico dopo il salvataggio) passa da
   `sectorsList`, che prima arrotonda ogni peso a 2 decimali e poi applica
   `capAtHundred`; le modifiche manuali sui settori bypassano l'helper e
   restano governate dal guard settori (100 ± 0,5).

### `/settings` — Impostazioni (`routes/settings/+page.svelte`)

Endpoint chiamati: `settingsApi.listCurrencies()`, `updateProfile()`,
`authApi.changePassword()`, `backupApi.download()`, `backupApi.restore()`.
La barra `SettingsTabs` naviga le cinque
sezioni (Profilo · Password · **Preferenze** · Valute · Backup e ripristino) e le sue etichette
sono tradotte tramite il layer i18n (capitolo 8).

- **Profile** (nome/email/**valuta base**) e **Change password**
  (`POST /users/me/password` con `current_password` + `new_password`,
   verifica lato frontend che le due nuove coincidano). Il selettore di valuta
   base è un dropdown `CurrencySelect` alimentato da `settingsApi
   .listCurrencies()` (la whitelist abilitata), inizializzato da
   `auth.user.base_currency` (fallback `EUR`) e salvato con
   `updateProfile(name, email, baseCurrency)`; pilota il riepilogo e la
   conversione dello storico della dashboard (capitolo 10).
- **Preferenze** (`routes/settings/preferences/+page.svelte`):
   una pagina completamente tradotta. Tema **Chiaro/Scuro/Sistema** con
   una `SegmentedControl` collegata allo store del tema (default Sistema),
   **palette utile/perdita Verde/Rosso / Blu/Arancione** —
   etichette `preferences.palette*` corte così il controllo non può uscire dalla
   card a larghezza telefono; il pannello comandi le riusa come
   hint dello stato di destinazione e `preferences.paletteHint` porta la
   spiegazione estesa — con una seconda
   `SegmentedControl` collegata allo store della palette (`setCvd` /
   `palette.cvd` — si applica subito, persiste in
   `localStorage['peculium-cvd']`, i grafici si re-inizializzano allo
   switch) e **lingua** dell'interfaccia (Italiano/English, default
   italiano) con una `Select` collegata a `setLocale` in
   `lib/i18n/`. Tutte si applicano subito e persistono in `localStorage`
   (niente pulsante di salvataggio); cambiando lingua la navigation della
   shell si ri-renderizza sul posto. Il tab si trova tra Password (=
   Sicurezza) e Valute.
- **Valute gestite**: il CRUD della whitelist valute — aggiungi un codice di 3
  lettere (un 422 dal backend significa che Yahoo non ha la conversione
  USD→codice e il frontend mostra un messaggio dedicato; 409 = già presente),
  elimina con conferma (409 = in uso o protetta). I simboli sono renderizzati
  con `currencySymbol()`.
- **Backup e ripristino** (`routes/settings/backup/+page.svelte`): export/import
  dei dati del singolo utente come un unico bundle JSON versionato. Il
  **Download** chiama `backupApi.download()` — una GET grezza attraverso il
  client condiviso `request` (l'allegato deve arrivare come `Blob`, quindi a
  `request` viene chiesta la `Response` grezza) — e lo salva con l'helper
  condiviso `saveBlob`, che prende il nome suggerito dall'`Content-Disposition`
  della risposta. Il **Ripristino** prima legge il file scelto lato client (JSON
  non parsabile → messaggio file non valido, senza toccare l'API), poi sceglie
  la strategia: **add** ("Aggiungi ai dati attuali", default non distruttivo,
  selezionato per ogni nuovo file scelto) o **replace** ("Sostituisci i dati
  attuali" — distruttivo: cancella i portafogli dell'account prima di importare;
  con questa opzione attiva il pulsante Ripristina prende la variante `danger` e
  la richiesta parte solo dopo un `ConfirmDialog` esplicito). `POST
  /backup/restore?mode=…` risponde il riepilogo dei conteggi
  (`portfolios_created`, `transactions_created`, `assets_created`,
  `assets_reused`): toast di successo e conteggi lasciati visibili in una
  griglia `<dl>` inline (2 colonne sul telefono, 4 da `sm`); la validazione di
  formato e versione è compito del server, quindi un 400 (formato/versione non
  supportati) mappa sul messaggio localizzato bundle non valido, tutto il resto
  sul fallback di ripristino fallito. Gli asset sono globali e nessuna modalità
  li elimina: i ticker esistenti vengono riusati, i mancanti ricreati.

### `/admin/health` — Price Sync Health (`routes/admin/health/+page.svelte`)

L'unica pagina che usa il **client generico**: `api.get('/health/prices?period=today|24h|100')`
(stessa origine `/api/v1/health/prices`). Un selettore di periodo (Today / Last
24h / Last 100) limita il riepilogo, che il backend calcola dalla tabella
`health_events` sulla finestra scelta (non si azzera al riavvio). Mostra 4
card di riepilogo (Success Rate, Total Successes, Total Failures, Rate Limited)
e una tabella paginata degli eventi recenti (timestamp, tipo, badge dello stato,
codice, messaggio, durata; 50 per pagina con Previous/Next e indicazione
dell'intervallo), con un pulsante "Refresh Now".

### Area Admin (`routes/admin/…`)

La home di gestione per gli account amministratore, raggiungibile dalla
sezione Admin della sidebar, dal foglio "Altro" del telefono e dal pannello
comandi — che renderizzano le sue voci (`Utenti`, `Impostazioni server`,
`Backup del server`) solo
quando `isAdmin()`. Il
contenuto delle pagine è avvolto da `AdminGate`
(`lib/components/domain/AdminGate.svelte`): chi ha un ruolo di sessione non
equivalente ad admin e arriva alla route (es. digitando l'URL) trova uno stato
vuoto "Solo amministratori" al posto della pagina; il backend risponde 403 su
ogni richiesta `/admin/*` indipendentemente, quindi questo è solo gating
presentazionale.

### `/admin/users` — Gestione utenti (`routes/admin/users/+page.svelte`)

Chiama `adminApi.listUsers()`, `adminApi.updateUser()`,
`adminApi.resetPassword()`. Una tabella di tutti gli account (Email, Nome,
Ruolo, Stato, Creato, Azioni) con un `Badge` per il ruolo (`owner`/`admin`
accent, gli altri neutral) e uno per lo stato (active positivo, pending
warning, disabled negativo; le etichette degli enum sono localizzate da
`admin.*`, i valori inattesi mostrano la stringa grezza). Le azioni di riga
sono pulsanti icona con l'email dell'obiettivo nel nome accessibile, e i
controlli della riga si bloccano mentre una richiesta per quella riga è in
corso:

- **Approva** (`UserCheck`, registrazioni pending) e **Riattiva** (`Check`,
  account disabled) fanno PATCH `status: active` diretto; **Disattiva**
  (`Ban`) fa PATCH `status: disabled` dietro `ConfirmDialog` danger;
- un `Select` di ruolo per riga — i valori assegnabili sono
  `admin`/`editor`/`viewer`; il legacy `owner` compare solo sulla riga che
  ancora lo porta (così un cambio è possibile senza rinominarlo in silenzio)
  e una nota sotto la tabella spiega che non è assegnabile a nuovi account;
  la selezione è ottimistica e viene ritratta in caso di errore;
- **Reimposta password** (`KeyRound`) apre una `Modal` che chiede una nuova
  password (minimo 8 caratteri, validazione inline con le chiavi `password.*`)
  e POSTa `/admin/users/{id}/reset-password`.

Ogni PATCH riuscito sostituisce la riga con il record restituito (senza
rifetch) e, quando la riga modificata è l'account corrente, sincronizza anche
`auth.user` — un auto-declassamento rivela subito lo stato "Solo
amministratori" di `AdminGate`. Gli errori diventano toast localizzati: 409
(cambiamento che lascerebbe il server senza un amministratore attivo) e 404
(utente sconosciuto) hanno messaggi dedicati; tutto il resto mostra il
fallback generico di aggiornamento/reset. Le stringhe grezze del backend non
arrivano mai alla UI.

### `/admin/settings` — Impostazioni server (`routes/admin/settings/+page.svelte`)

Chiama `adminApi.getSettings()` / `adminApi.updateSettings()`. Una sola card
con un `ui/Switch` per l'unico impostazione dell'intero server, **Approvazione
automatica delle nuove registrazioni** (`auto_approve_registrations`): se
attiva i nuovi account accedono subito, se disattiva le registrazioni nascono
`pending` e un amministratore le approva dalla pagina Utenti. Il toggle si
applica subito (PATCH ottimistica al cambio, ritratta con toast d'errore se
fallisce) e un timbro "Ultimo aggiornamento" sta sotto l'etichetta.

### `/admin/backup` — Backup del server (`routes/admin/backup/+page.svelte`)

Dump e ripristino dell'intero database di server (`adminApi.dbBackup` /
`adminApi.dbRestore`, endpoint riservati agli amministratori e pagina avvolta
da `AdminGate`). La card **Scarica** trasmette un dump completo `pg_dump`
(formato custom) via `GET /admin/db/backup` (`application/octet-stream`,
allegato `peculium-db-<date>.dump`) con lo stesso percorso blob autenticato
grezzo del backup per-utente: è il `Content-Disposition` del server a dare il
nome del file salvato. La card **Ripristina** sceglie un file `.dump` (il
client non ispeziona mai il binario; valida il server) e blocca la richiesta
dietro una conferma scritta: un `Modal` danger (`admin.dbConfirmWarning`
porta l'avviso completo — ogni tabella riscritta, utenti inclusi, sessione
possibilmente invalidata) il cui pulsante di conferma si abilita solo quando
l'amministratore digita esattamente la frase di conferma
(`admin.dbConfirmPhrase`, `REPLACE`/`SOSTITUISCI` per lingua). Il caricamento
è `POST /admin/db/restore` in `multipart/form-data` con
l'archivio nel campo file `dump` e il letterale `confirm=replace` come campo
del form (`request` invia il `FormData` così com'è, lasciando al browser la
derivazione del `Content-Type` con il boundary). Il successo risponde
`{dump_bytes}`: toast più pannello inline con la dimensione dell'archivio
ripristinato e un avviso persistente di nuovo login, poiché la tabella utenti fa
parte dell'archivio e la sessione corrente potrebbe non corrispondere più ai
dati ripristinati. Gli errori riportano deliberatamente il messaggio del
backend (400 per conferma/file mancanti, 500 con lo stderr sanificato di
`pg_restore` — un ripristino parzialmente applicato deve essere leggibile
nella UI); solo gli errori di rete senza status ripiegano sul generico
localizzato. Entrambe le operazioni hanno 30 minuti di margine lato server: i
pulsanti mostrano lo spinner e la pagina riporta un avviso "tieni aperta
questa pagina".