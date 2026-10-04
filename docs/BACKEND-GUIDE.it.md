# Peculium — Il backend spiegato

> Questo documento spiega come funziona il backend di Peculium: il programma
> che gestisce i dati finanziari e li mette a disposizione del sito web.
> Non richiede conoscenze di programmazione: i concetti di server, database e
> linguaggi vengono spiegati man mano. Se non hai mai aperto un file di codice,
> leggi prima il capitolo 2 ("Concetti di base") e il capitolo 18 ("Leggere il
> codice"), poi il resto ti sarà chiaro.

---

## 1. Cosa fa Peculium

Peculium è un'applicazione per tenere traccia degli investimenti: registri cosa
compri e cosa vendi, e l'app ti mostra quanto valgono i tuoi titoli, quanto hai
guadagnato o perso, e come è andato il portafoglio nel tempo.

Per funzionare servono due "mondi" che collaborano:

- Il **frontend**: la pagina web che vedi nel browser (grafici, form, pulsanti).
- Il **backend**: il programma "dietro le quinte" che legge e scrive i dati,
  fa i calcoli finanziari e risponde alle richieste del frontend.

Questo documento parla del backend.

---

## 2. Concetti di base

Prima di entrare nei dettagli, un piccolo glossario. Se già conosci questi
concetti, puoi saltare il capitolo.

- **Server / processo**: un programma in esecuzione che resta in attesa di
  richieste. Il backend è un programma Go compilato in un file eseguibile.
- **API / endpoint**: un "numero di telefono" che il backend espone. Il
  frontend chiama un endpoint (es. `POST /auth/login`) e riceve una risposta.
  In gergo si parla di *richiesta* (request) e *risposta* (response).
- **HTTP**: il "linguaggio" con cui frontend e backend si parlano. I verbi
  principali sono `GET` (chiedere dati), `POST` (creare qualcosa),
  `PATCH` (modificare), `DELETE` (cancellare).
- **JSON**: il formato testuale in cui viaggiano i dati dentro una richiesta.
  È fatto di coppie "nome: valore" tra parentesi graffe, ad esempio
  `{"email": "mario@example.com", "nome": "Mario"}`. Si legge come una scheda
  compilata.
- **Database**: un programma che conserva i dati in modo ordinato su disco.
  Peculium usa **PostgreSQL**.
- **Tabella**: dentro al database i dati sono organizzati in tabelle (come
  fogli di calcolo) con righe e colonne. La tabella `users`, per esempio,
  contiene una riga per ogni utente.
- **SQL**: il linguaggio con cui si interroga il database
  (`SELECT ... FROM ... WHERE ...`).
- **Chiave primaria (PRIMARY KEY)**: la colonna che identifica in modo univoco
  ogni riga (come un codice fiscale).
- **Chiave esterna (FOREIGN KEY)**: una colonna che "punta" a un'altra tabella
  per collegare le righe tra loro (es. una transazione punta al portafoglio a
  cui appartiene).
- **Migrazione**: un file SQL versionato che costruisce o modifica lo schema
  del database. Le migrazioni si applicano in ordine.
- **Cache**: una copia di dati già calcolati o scaricati, tenuta da parte per
  non ripetere lo stesso lavoro.
- **Rate limit / throttle**: limitare il numero di chiamate verso un servizio
  esterno nell'unità di tempo, per non essere bloccati.
- **Container**: un ambiente isolato in cui un programma gira con tutto ciò che
  gli serve. Peculium usa Docker.
- **Redis**: un database in memoria (molto veloce) che qui funziona da
  "contatore condiviso" per tenere sotto controllo le chiamate verso Yahoo.

Un'analogia per tenere tutto insieme: il **backend è la cucina di un
ristorante**. Il frontend (il cameriere) porta l'ordine (la richiesta HTTP), la
cucina prepara il piatto (legge i dati e fa i calcoli) e lo consegna (la
risposta JSON). Il database è la dispensa con le materie prime.

---

## 3. Visione globale

```mermaid
flowchart LR
    FE["Frontend — pagina web (porta 3000)"]
    subgraph BE["Backend (Go)"]
        direction TB
        API["cmd/server/main.go — porta 8080"]
        CHAIN["Router chi → handler → service → repository"]
        WK["cmd/worker/main.go (processo separato)<br/>aggiorna i prezzi in background"]
        API --> CHAIN
    end
    PG[("PostgreSQL 16 — dati veri + serie giornaliere materializzate")]
    RD[("Redis 7 — contatore per limitare le chiamate a Yahoo")]
    YH["Yahoo Finance (HTTP) — prezzi, storico, valute (FX), split dei titoli"]

    FE -- HTTP --> API
    CHAIN --> PG
    CHAIN --> RD
    CHAIN --> YH
    WK --> YH
    WK --> PG
```

I pezzi che girano (definiti in `docker-compose.dev.yml` per lo sviluppo locale
e in `docker-compose.yml` per lo stack di release pull-only):

- **secrets-init** — genera la password del database e la chiave di firma JWT al
  primo avvio e le conserva nel volume `peculium_secrets`; poi resta attivo come
  heartbeat che fa da gate per il database e il backend.
- **postgres** — il database vero e proprio: utenti, portafogli, transazioni,
  prezzi, tassi di cambio, serie giornaliere.
- **redis** — un magazzino in memoria usato come contatore condiviso per
  non superare il numero di chiamate consentito verso Yahoo.
- **backend** — l'API REST che parla con il frontend.
- **worker** — un processo separato che aggiorna i prezzi in background.
- **python-service** — un piccolo servizio FastAPI (`python-service/`) che recupera
  i metadati ETF (paesi/regioni e settori GICS) e risolve l'ISIN dal ticker via
  JustETF. Il backend Go lo chiama tramite `PECULIUM_PYTHON_SERVICE_URL`.
- **frontend** — la pagina web.

Il backend è **un unico programma Go** che, a seconda dell'argomento passato
quando viene avviato, fa due cose diverse (`cmd/server/main.go`):

- `server migrate` — applica le migrazioni SQL (costruisce le tabelle) e poi
  termina;
- `server` — avvia l'API HTTP e resta in ascolto.

Nel container il comando è `sh -c "/server migrate && /server"`, quindi le
migrazioni girano sempre prima dell'avvio dell'API. Lo stack di release fa
partire il backend solo quando Postgres è healthy (`depends_on: condition:
service_healthy`); il compose di sviluppo antepone allo stesso comando `sleep 3`.

---

## 4. Come si avvia il backend

La sequenza di avvio del server è:

1. **Configurazione** — `config.Load()` legge le variabili d'ambiente (con un
   valore di default se mancano). Le variabili d'ambiente sono valori di
   configurazione che si passano al programma all'avvio (ad esempio l'indirizzo
   del database), senza doverle scrivere dentro al codice. I due valori
   sensibili, la password del database e il segreto di firma JWT, accettano
   anche una variante `*_FILE` (`PECULIUM_DB_PASSWORD_FILE`,
   `PECULIUM_JWT_SECRET_FILE`): il valore viene allora letto dal file indicato
   (gli spazi bianchi in coda sono ignorati). La precedenza è: valore
   d'ambiente esplicito > contenuto del file > default.
2. **Connessione al database** — `pgxpool.New(ctx, cfg.DSN())`. `pgxpool` è un
   **pool di connessioni**: tiene pronte alcune connessioni al database, le si
   prende quando serve e le si restituisce, così non si riapre una connessione
   da zero per ogni richiesta.
3. **Redis** — si prova a collegarsi a Redis per attivare il contatore globale
   delle chiamate a Yahoo. Se Redis non risponde, il server usa un contatore
   "finto" che non blocca mai e **continua comunque** a funzionare.
4. **Router HTTP** — si prepara la lista degli endpoint (il "menù" del
   ristorante) e si aggiungono dei comportamenti comuni a tutte le richieste:
   log, recupero dagli errori, un ID per tracciare ogni richiesta, un timeout
   di 30 secondi e la gestione CORS (le regole che permettono al frontend di
   chiamare l'API).
 5. **Routes** — si collegano gli endpoint alle funzioni che li gestiscono.
    Alcuni endpoint sono pubblici (`/auth/*`), gli altri richiedono di essere
    autenticati con un token valido appartenente a un account attivo
    (controlla il capitolo sull'autenticazione), e quelli sotto
    `/api/v1/admin/*` richiedono anche il ruolo admin. Fuori da
    `/api/v1`, `GET /healthz` è una sonda di vitalità pubblica: risponde
    `{"status":"ok"}` con HTTP 200 e non tocca né il database né Redis.
    Anche `GET /api/v1/version` è pubblica: risponde
    `{"version":"...","commit":"...","built_at":"..."}` con i metadati di
    build iniettati come variabili d'ambiente (`PECULIUM_VERSION`,
    `PECULIUM_COMMIT`, `PECULIUM_BUILT_AT`), senza richiesta di login.
6. **Backfill delle serie** — dopo l'avvio, una goroutine (un "filo" di
   esecuzione che lavora in parallelo al resto del programma, così l'API può
   rispondere alle richieste mentre il calcolo prosegue) ricostruisce le serie
   giornaliere di tutti i portafogli (capitolo 9).

Allo spegnimento il server aspetta il segnale `SIGINT`/`SIGTERM` e poi chiude
le connessioni in modo ordinato, senza interrompere le richieste a metà.

---

## 5. Architettura a strati

Il codice del backend è diviso in **quattro strati**, ognuno con un compito
preciso. La regola d'oro è che ogni strato può usare solo quello sotto di sé,
mai quello sopra.

```
HTTP (router)
   │  richieste e risposte JSON
   ▼
handler   →  riceve la richiesta HTTP, capisce cosa vuole il frontend
   │           e invia la risposta. Non tocca il database.
   ▼
service   →  la logica di business: login, calcolo del prezzo medio,
   │           conversione delle valute, orchestrazione dei passaggi
   ▼
repository →  parla con PostgreSQL con SQL puro e parametrizzato
```

- **handler** (`internal/handler/`) — la "porta d'ingresso". Conosce le
  richieste HTTP ma non sa nulla di SQL.
- **service** (`internal/service/service.go`) — il "cervello". Fa i conti e
  coordina. Non sa nulla di HTTP.
- **repository** (`internal/repository/`) — il "magazziniere". Scrive le query
  SQL e converte le righe del database in oggetti Go.

I tre strati vengono **collegati a mano** (dependency injection manuale): si
creano prima i repository, poi i service (che ricevono i repository), poi gli
handler (che ricevono i service). Niente magie o framework di iniezione:

```go
repos := repository.New(dbPool)                                   // strato dati
fetcher := price.NewYahooFetcher(repos, cfg.PriceFetchInterval,
    price.WithMinInterval(cfg.YahooMinInterval),                  // coda 400ms
    price.WithRateBudget(budget),                                 // contatore Redis
)                                                                 // client Yahoo
svc := service.New(repos, jwtAuth, fetcher, cfg.LookupCacheTTL)   // logica
h := handler.New(svc, jwtAuth)                                    // HTTP
```

---

## 6. Il database

Le migrazioni (`backend/migrations/`, file numerati da `000001` a `000019`)
costruiscono lo schema. Le tabelle principali:

| Tabella | Contiene | Spiegazione |
|---|---|---|
| `users` | gli utenti | email, nome, hash della password, ruolo, stato dell'account (`active`, `pending`, `disabled`), valuta base (`base_currency`, default EUR) |
| `assets` | i titoli | ticker, nome, tipo (azione, ETF, crypto...), classe di investimento, fonte prezzi, valuta, exchange, settore, industria |
| `portfolios` | i portafogli | un portafoglio appartiene a un utente e ha una valuta |
| `portfolio_shares` | la condivisione | chi altro può vedere un portafoglio (con che ruolo) |
| `transactions` | le operazioni | compra/vendita/dividendo/split/commissione, quantità, prezzo, data |
| `prices` | i prezzi giornalieri | per ogni titolo e per ogni giorno: apertura, chiusura, volume |
| `fx_rates` | i tassi di cambio | quanto vale 1 dollaro in ogni altra valuta |
| `splits` | gli split azionari | es. un titolo da 1 azione diventa 4 azioni |
| `asset_region_weights` | l'esposizione geografica | per ogni titolo, il peso di ogni macro-regione |
| `asset_country_weights` | l'esposizione per-paese | per ogni titolo, il peso di ogni paese ISO-3166 |
| `asset_sector_weights` | l'esposizione settoriale | per ogni titolo, il peso di ogni settore GICS |
| `supported_currencies` | la lista delle valute | quali valute si possono usare (capitolo 11) |
| `lookup_cache` | la cache del cerca-titolo | risultati già scaricati da Yahoo per la ricerca automatica |
| `portfolio_series` | serie per portafoglio | valore e costo del portafoglio per ogni giorno |
| `asset_series` | serie per titolo | valore e costo di ogni titolo per ogni giorno |
| `health_events` | gli eventi di health dei prezzi | registrazioni di aggiornamenti prezzi vecchi/falliti nella pagina health |
| `server_settings` | le opzioni del server | una sola riga: auto-approvazione delle registrazioni (capitolo 14) |

Due idee fondamentali del database:

1. **Le righe si collegano tra loro.** Una transazione ha una chiave esterna
   verso il portafoglio e una verso il titolo. Così, dato un portafoglio,
   si trovano tutte le sue transazioni.
2. **Le chiavi primarie sono UUID.** Un UUID è un identificatore casuale lungo
   (es. `a3f2...`). A differenza dei numeri progressivi, un UUID si può
   generare senza chiedere al database, il che rende più semplice il lavoro in
   parallelo.

---

## 7. Una richiesta tipica: la dashboard

Prendiamo `GET /api/v1/dashboard`, l'endpoint più ricco: serve alla pagina
principale per mostrare tutti i portafogli, i titoli e i guadagni.

Cosa succede, passo passo:

1. **Router**: la richiesta arriva e viene indirizzata alla funzione
   `GetDashboard`.
2. **Middleware JWT**: il token dell'utente (capitolo 14) viene controllato e
   i dati dell'utente vengono messi "nel contesto" della richiesta.
3. **Handler**: estrae i dati dell'utente, chiama il service, e invia la
   risposta JSON.
4. **Service `GetDashboard`**: orchestrazione — carica l'utente (e la sua
   **valuta base**, vedi capitolo 10), i portafogli dell'utente, le posizioni
   dettagliate (con il prezzo medio di carico, capitolo 8) e i tassi di cambio
   per le valute coinvolte. La risposta espone `base_currency` e un
   riepilogo `summary` convertito nella valuta base; `by_currency`,
   `portfolios` e `assets` restano espressi nella propria valuta. Il
   `summary` e ogni voce di `portfolios` dividono i numeri negli oggetti
   annidati `active` e `closed` (vedi capitolo 8): `active` riporta
   `invested`, `value`, `gain_loss` e `gain_loss_pct` dei lotti ancora in
   possesso più i `dividends` delle posizioni ancora aperte (anche se
   parzialmente vendute); le posizioni aperte di asset che non hanno alcun
   prezzo sono escluse da `invested` e `value` (il loro costo non ha un
   valore di mercato con cui confrontarlo, e contarlo simulerebbe una
    perdita del -100%), mentre i loro dividendi contano comunque. `closed`
    riporta `invested` (costo AVCO dei lotti venduti), `proceeds` (incasso
    netto di vendita più i dividendi
conferiti dalle posizioni completamente chiuse), `realized` (proceeds −
     invested) e `realized_pct` (realized / invested × 100). La risposta
     espone anche `invested_assets`: l'unico elenco consolidato
     degli asset attualmente investiti, aggregato per asset su tutti i
     portafogli dell'utente (lo stesso asset presente in più portafogli è una
     sola riga) e convertito nella valuta base. Ogni riga porta `asset_id`,
     `ticker`, `name`, `currency` (quella dell'asset), `invested` (costo della
     quantità aperta), `value` (valore di mercato, al costo quando l'asset non
     ha prezzo — `has_price: false` — o quando manca il suo tasso FX verso la
     valuta base, così non compare una finta perdita), `gain_loss` e
     `gain_loss_pct`; le righe sono ordinate per `value` decrescente e le
     posizioni chiuse (qty ≤ 0) sono escluse.
5. **Repository**: esegue le query SQL, per esempio la query che carica i
   portafogli con un `LEFT JOIN` sulla tabella di condivisione.

### Il grafico del time-weighted return (`GET /dashboard/performance`)

`GET /api/v1/dashboard/performance?granularity=month|year` restituisce un
unico grafico per l'intero patrimonio (tutti i portafogli aggregati, convertiti
nella valuta base dell'utente) invece di una serie per portafoglio.
`granularity` è `month` di default; qualunque altro valore viene rifiutato
con 400. La risposta è `{currency, granularity, buckets[]}`, dove ogni bucket
ha `period` (`YYYY-MM` per i mesi, `YYYY` per gli anni), `return`, `twr`,
`invested` e `value`:

- il **`return`** è il **vero time-weighted return (TWR)** percentuale del
  bucket: i rendimenti sono misurati **giornalmente** e collegati
  geometricamente, `return = (Π (1 + r(d)) − 1) × 100` sui giorni `d` del
  bucket, dove ogni giorno porta `r(d) = (V(d) − V(d−1) − flusso(d)) /
  V(d−1)` quando `V(d−1) > 0` e viene saltato (fattore 1) altrimenti — il
  primo giorno e le pause di un patrimonio completamente liquidato non misurano
  alcun rendimento, quindi liquidare e riaprire una posizione non distorce
  mai il grafico. `V(d)` è **solo il valore di mercato**: `V(d) =
  mv_priced(d) + bond_at_cost(d)`, il valore di mercato convertito in FX
  degli asset con prezzo più il costo di quelli senza prezzo (un'obbligazione
  mantenuta al costo finché è in portafoglio; già zero una volta venduta
  interamente). Il realized non entra mai in `V`: è già catturato dal flusso
  di vendita;
- il **`twr`** è il time-weighted return cumulativo composto su tutti i
  giorni fino alla fine del bucket: `(Π (1 + r) − 1) × 100`, cioè la stessa
  collegatura geometrica dei rendimenti dei bucket;
- **`invested`** è il capitale netto impiegato: la somma cumulativa dei
  *flussi di capitale* in valuta base fino alla fine del bucket — gli stessi
  flussi ma con i dividendi contati come 0 (le distribuzioni sono reddito,
  non capitale; una vendita sopra costo può renderlo temporaneamente
  negativo);
- **`value`** è il valore di mercato `V` all'ultima data del bucket (asset
  con prezzo al valore di mercato, asset senza prezzo al costo).

I flussi di cassa esterni per transazione — collocati a **fine giornata** —
convertiti nella valuta base al tasso FX della data della transazione (il
flusso viene saltato se quel tasso manca): `buy → +(qty·prezzo + fee)`
(deposito), `sell → −(qty·prezzo − fee)` (prelievo), `fee` autonoma
`→ +importoFee` dove `importoFee = qty·prezzo`, o solo il prezzo quando la
quantità non è impostata; `dividend → −importoDiv` (stessa regola
dell'importo; reddito prelevato, quindi dà un contributo positivo al
rendimento senza toccare `invested`) e `split → 0`.

I punti giornalieri arrivano dalle serie materializzate per singolo asset,
`asset_series` (via `Series.FindPortfolio`), nella valuta del portafoglio, e
vengono convertiti nella valuta base per data attraverso lo storico FX con
pivot USD: finché il tasso per un asset manca, il suo ultimo valore
convertito viene ripetuto in avanti (zero prima della prima conversione
riuscita), e ogni asset continua a portare il suo ultimo valore convertito
nelle date in cui non ha un nuovo punto. I valori di mercato vengono
aggregati per data in `V`, i flussi raggruppati per giorno e la passeggiata
scorre l'unione delle date delle serie e delle date dei flussi in ordine
crescente (la serie materializzata copre ogni giorno solare dal primo
movimento di ogni asset, quindi in pratica i giorni dei flussi hanno sempre
un'osservazione di mercato). I periodi dei bucket sono monotoni sulle date,
quindi l'ultima data di ogni mese/anno sigilla il suo bucket e i bucket
vengono emessi in ordine crescente — i bucket senza dati vengono
semplicemente omessi. `return` e `twr` si arrotondano a 4 decimali,
`invested` e `value` a 8.

### L'elenco paginato delle transazioni (`GET /portfolios/{id}/transactions`)

`GET /api/v1/portfolios/{id}/transactions?limit=&offset=` restituisce un
involucro di paginazione — `{transactions[], total, limit, offset}`. `limit`
ha default 20 e viene
limitato a un massimo di 100 (un valore più grande torna come
`limit: 100`), `offset` ha default 0; un valore non numerico per uno dei due
parametri è un 400, e lo stesso vale per un valore negativo (espresso dal
service come `ErrInvalidInput`). La pagina è ordinata per
`date DESC, created_at DESC, id DESC` — un tie-break del tutto
deterministico, quindi due transazioni con la stessa data non possono
finire a cavallo di due pagine né ripetersi. La proprietà del portafoglio
è verificata nel service prima di qualsiasi query: il portafoglio altrui è
un 403 e uno inesistente un 404. `total` è il numero completo di
transazioni del portafoglio, non la dimensione della pagina: una pagina
oltre la fine restituisce semplicemente un array `transactions` vuoto con
il `total` corretto.

L'elenco accetta anche quattro filtri opzionali e liberamente
combinabili (assenti = comportamento di cui sopra):

| Parametro | Significato | Valore non valido |
|-----------|-------------|-------------------|
| `type` | tipo esatto: `buy`, `sell`, `dividend`, `split`, `fee` | 400 `invalid type` |
| `asset_id` | UUID esatto dell'asset | 400 `invalid asset_id` |
| `from` | data iniziale inclusiva, formato rigoroso `YYYY-MM-DD` | 400 `invalid from: date must be YYYY-MM-DD` |
| `to` | data finale inclusiva, formato rigoroso `YYYY-MM-DD` | 400 `invalid to: date must be YYYY-MM-DD` |

La query viene convertita in un `model.TransactionFilter` tipizzato
(`Type string`, `AssetID *uuid.UUID`, `From/To *time.Time` a mezzanotte UTC)
da `model.ParseTransactionFilter`; il service lo propaga sia a
`FindByPortfolioPage` sia a `CountByPortfolio`, quindi **`total` riflette il
conteggio filtrato** — il frontend può mostrare "1–20 di 42" sull'insieme
filtrato. Il repository costruisce un `WHERE` dinamico e parametrizzato a
partire dal filtro (i segnaposto sono numerati da un contatore, i valori
viaggiano solo come bind parameter — mai interpolati); i confini di data
confrontano `t.date::date` con i giorni parsati, quindi il confronto è sul giorno
di calendario ed è inclusivo su entrambi i lati (la colonna è
`TIMESTAMPTZ`). Ordine e semantica della paginazione seguono le regole
sopra.
Un `filter.Type` non vuoto fuori dall'insieme consentito è rifiutato dal
service con `ErrInvalidInput` anche se la chiamata bypassa il parser HTTP.

### Il grafico TWR per singolo portafoglio (`GET /portfolios/{id}/performance/buckets`)

`GET /api/v1/portfolios/{id}/performance/buckets?granularity=month|year`
applica esattamente lo stesso modello TWR giornaliero a un **singolo
portafoglio**, espresso nella **valuta del portafoglio stesso**: stesso
default di `granularity` (`month`, qualunque altro valore → 400) e stessa
forma di risposta `{currency, granularity, buckets[]}` del grafico
a livello di patrimonio. È la versione più semplice di quella della dashboard perché non
c'è alcuna conversione in valuta base: le serie materializzate per asset sono
già denominate nella valuta del portafoglio, quindi `V(d)` non ha bisogno di
alcun passaggio FX. Solo i flussi di cassa — registrati nella valuta
dell'asset — vengono convertiti nella valuta del portafoglio al tasso FX
della data della transazione (via lo storico con pivot USD), e una
transazione senza tasso o senza valuta nota dell'asset viene saltata,
proprio come sulla dashboard. La proprietà del portafoglio è verificata prima
della consultazione della cache (403 per il portafoglio altrui, 404 se non
esiste) e il risultato è cachato sotto la chiave `pf-perf` come
`{portfolioID}:{granularity}`.

### Il riepilogo del portafoglio (`GET /portfolios/{id}/summary`)

La pagina di dettaglio del portafoglio mostra la stessa card "Investimenti"
della dashboard: accanto ai campi piatti (`total_cost`, `total_value`,
`gain_loss`, `gain_loss_pct`, `realized_gl`, `unrealized_gl`), la risposta
porta gli oggetti annidati
`active` e `closed` con esattamente le stesse regole delle voci per-portfolio
della dashboard (vedi sopra): `active` somma costo e valore di mercato dei
lotti ancora aperti (`invested`, `value`, `gain_loss`, `gain_loss_pct`) più i
`dividends` delle posizioni ancora aperte (anche parzialmente vendute); le
posizioni aperte di asset senza prezzo restano fuori da
`invested`/`value` (il loro costo non ha un valore di mercato comparabile)
ma contano comunque i dividendi; `closed` riporta `invested` (costo AVCO dei
lotti venduti), `proceeds` (ricavi netti di vendita più i dividendi delle
posizioni completamente chiuse), `realized` e `realized_pct`. Tutto
è espresso nella **valuta del portafoglio**: non si applica nessuna
conversione nella valuta base e solo il valore di mercato passa dal fattore
FX asset→portafoglio, quindi un valore senza tasso disponibile viene omesso
e segnalato dai campi di qualità dati `fx_missing_count` /
`fx_missing_value`. I numeri sono arrotondati dallo stesso helper
`finalizeBreakdowns` della dashboard; la lista `holdings` è restituita
anch'essa.

Nel codice Go il pattern tipico per leggere più righe è:

```go
for rows.Next() {
    rows.Scan(&x, &y)   // legge una riga e la mette in variabili
}
```

Il programma scorre le righe una alla volta (`for rows.Next()`) e le copia
nelle variabili (`Scan`). Go richiede di **mappare a mano** ogni colonna in una
variabile, senza scorciatoie magiche: per ogni riga, si decide esplicitamente
quale valore va in quale variabile.

---

## 8. L'AVCO — prezzo medio di carico

L'AVCO (Average Cost) è la **logica con cui si calcola quanto vale la
posizione di un titolo**, come fa il tuo broker. È il cuore finanziario
dell'applicazione, in `internal/position/position.go`.

Immagina di comprare 10 azioni a 100 € e poi altre 10 a 120 €. Quanto hai
speso in totale? 2.200 €. Il tuo **prezzo medio di carico** è
2.200 / 20 = 110 € per azione. L'AVCO tiene traccia di questo numero mentre
avvengono le operazioni.

### Lo `State`

Per ogni titolo il motore tiene uno "stato" della posizione:

```go
type State struct {
    Qty      decimal.Decimal  // quantità di azioni che possiedi
    Avg      decimal.Decimal  // prezzo medio di carico
    Cost     decimal.Decimal  // totale investito
    Realized decimal.Decimal  // plus/minusvalenza già realizzata

    ClosedCost decimal.Decimal // costo AVCO dei lotti venduti
    Proceeds   decimal.Decimal // incasso netto di vendita
    Dividends  decimal.Decimal // dividendi incassati
}
```

> Cos'è `struct`? In Go una `struct` è un contenitore che raggruppa più valori
> con un nome: è come una "scheda" con più caselle. Qui la scheda "stato della
> posizione" ha le caselle dei lotti aperti (quantità, prezzo medio, costo
> totale, guadagno/perdita realizzata) più le metriche cumulative dei lotti
> chiusi e delle distribuzioni (costo dei venduti, incasso netto, dividendi)
> che alimentano il riepilogo attivo/chiuso della dashboard. Ogni
> campo cumulativo ha
> anche un gemello `*CCY` espresso nella valuta del titolo. `decimal.Decimal`
> è il tipo dei numeri (numeri con virgola precisi, adatti al denaro, senza
> errori di arrotondamento).

L'idea è semplice: le operazioni non modificano i dati alla rinfusa, ma
aggiornano la scheda in modo ordinato, operazione dopo operazione.

### `Apply` — come cambia lo stato con ogni operazione

- **Buy (comprato)**: aggiungi al costo `quantità × prezzo + commissione`,
  poi ricalcoli il prezzo medio: `Avg = Cost / nuovaQuantità`.
- **Sell (venduto)**: il costo della parte venduta è `Avg × quantità`. Lo
  sottrai dal costo totale; la differenza tra quello e l'incasso diventa
  `Realized` (guadagno o perdita già "incassato"). Il costo dei lotti venduti
  e l'incasso netto si accumulano anche in `ClosedCost` e `Proceeds`, così
  una posizione parzialmente venduta tiene separata la parte chiusa dai lotti
  ancora aperti.
- **Split**: la quantità viene moltiplicata per il rapporto (es. 1 diventa 4),
  ma il prezzo medio viene **diviso** per lo stesso rapporto: il costo totale
  non cambia.
- **Fee (commissione)**: si somma al costo.
- **Dividend (dividendo)**: si somma a `Realized` e, separatamente, a
  `Dividends` (la dashboard poi classifica i dividendi secondo lo stato della
  posizione: quelli di una posizione ancora aperta, anche se parzialmente
  venduta, entrano nel gruppo `active`, quelli di una posizione completamente
  chiusa si sommano ai `proceeds` del gruppo `closed`).

### `Walk`

`Walk` prende tutte le transazioni, le raggruppa per portafoglio e per titolo,
le ordina per data e **inietta gli split** nella linea del tempo. Uno split è
un fatto del titolo, quindi si applica a tutti i portafogli che possiedono quel
titolo.

---

## 9. Le serie giornaliere materializzate

Per disegnare i grafici serve, per ogni portafoglio, un valore per ogni giorno
dalla prima operazione a oggi. Questa serie non viene ricalcolata a ogni
richiesta: viene **salvata (materializzata)** nel database nelle tabelle
`portfolio_series` e `asset_series` (`internal/series/series.go`).

La logica di calcolo di un singolo giorno è:

```go
for i, d := range dates {
    // applica le transazioni avvenute fino alla data d
    // applica gli split avvenuti fino alla data d
    // prende l'ultimo prezzo disponibile ≤ d
    mv := st.Qty.Mul(priceByAsset[aid][priceDates[pricePos-1]]).Mul(rawFactor).Mul(factor)
    agg[i].MarketValue = mv   // sommato al totale del portafoglio
}
```

Due dettagli nel calcolo del valore di mercato:

- `rawFactor` — i prezzi storici di Yahoo sono **aggiustati per gli split**, e
  questo fattore li riporta alla realtà attuale;
- `factor` — la conversione dalla valuta del titolo a quella del portafoglio.

Quando la serie viene ricalcolata:

- a ogni modifica di transazioni o portafoglio (`AddTransaction`,
  `UpdateTransaction`, `DeleteTransaction`, `UpdatePortfolio`, import);
- dopo ogni aggiornamento dei prezzi (sia dal worker che da una richiesta
  manuale);
- all'avvio del server (il backfill di cui al capitolo 4).

La ricostruzione è guidata da `Recompute(portfolioID)` (un portafoglio) o
`RecomputeAll` (tutti). Leggere i grafici diventa quindi una semplice lettura
dal database, senza rifare tutti i calcoli a ogni apertura di pagina.

---

## 10. Valute e tassi di cambio (FX)

I portafogli e i titoli possono essere in valute diverse. Per sommare o
confrontare i valori serve convertire.

1. Il backend scarica da Yahoo i **tassi di cambio da dollaro (USD) verso le
   altre valute** (es. `USDCHF=X` per il franco) e li salva nella tabella
   `fx_rates`.
2. Al momento del calcolo, `series.LoadRates` carica i tassi delle valute che
   compaiono nelle posizioni.
3. Per convertire da una valuta A a una valuta B, `series.FxFactor` usa il
   **doppio passaggio attraverso il dollaro**:
   `(USD→B) / (USD→A)`.

> Non serve una tabella con tutte le coppie possibili: basta una riga per
> valuta (il tasso rispetto al dollaro) e il resto si calcola.

Se un tasso manca, la conversione non è disponibile e l'applicazione lo segnala
(nel modello compare il campo `fx_missing`).

**La valuta base.** Ogni utente ha una valuta preferita salvata in
`users.base_currency` (default EUR) e modificabile via `PATCH /users/me` con
il campo `base_currency` (un valore omesso/vuoto mantiene quello salvato; un
valore non vuoto deve essere una valuta abilitata della whitelist, capitolo
11, altrimenti la richiesta è rifiutata con 400). Tutte le aggregazioni a
livello di patrimonio della dashboard sono convertite in essa: `GET /dashboard`
restituisce `base_currency`, un riepilogo `summary` nella valuta base — gli
oggetti annidati `active` (investito, valore, guadagno/perdita dei lotti
ancora detenuti, più i dividendi delle posizioni aperte; le posizioni aperte
senza prezzo restano fuori da investito/valore, vedi capitolo 7) e `closed`
(invested = costo AVCO dei lotti venduti, proceeds = incasso di vendita +
dividendi delle posizioni completamente chiuse, realized = proceeds −
invested, realized_pct) — dove gli importi senza
tasso disponibile sono esclusi dai totali e riportati da
`fx_missing_count`/`fx_missing_value` (solo gli importi nonnulli vengono
segnalati); `GET /dashboard/performance` mostra lo stesso rendimento
percentuale time-weighted (TWR) del patrimonio per bucket mensili o annuali nella
valuta base, con le serie invested/value del capitale (capitolo 7);
`GET /portfolios/{id}/performance/buckets` mostra gli stessi bucket per un
singolo portafoglio, restando nella valuta del portafoglio (capitolo 7);
anche
`GET /dashboard/allocation` è espressa nella valuta base e aggrega tutti i
portafogli nelle ripartizioni a livello di patrimonio `classes`, `regions`, `countries`
e `sectors`. Le sezioni per-valuta
(`by_currency`) e per-portafoglio (`portfolios`, `assets`) della dashboard
mantengono la propria valuta. L'elenco `invested_assets` è invece espresso
nella valuta base: le posizioni aperte dell'utente aggregate per asset su
tutti i suoi portafogli (capitolo 7).

---

## 11. La lista delle valute disponibili

Non tutte le valute del mondo sono gestibili. L'applicazione tiene una
**whitelist** (una lista consentita) nella tabella `supported_currencies`, dove
sono già presenti due valute di base: USD e EUR.

Le valute si possono aggiungere e togliere da `/settings/currencies`
(GET = elenca, POST = aggiunge, DELETE = rimuove):

- **Aggiungere** una valuta: prima si controlla che Yahoo conosca la
  conversione USD → quella valuta. Se non la conosce, la valuta non può essere
  gestita e la richiesta viene rifiutata.
- **Rimuovere** una valuta: non si può rimuovere il dollaro (è la base di
  tutto) né una valuta che è ancora usata da qualche titolo o portafoglio.

---

## 12. I prezzi e il collegamento con Yahoo

Il pacchetto `internal/price/` si occupa di parlare con **Yahoo Finance** per
recuperare prezzi, storico, tassi di cambio, split e risultati della ricerca
dei titoli.

### Il client Yahoo (`YahooFetcher`)

Ha un client HTTP con timeout di 15 secondi, due "mappe di raffreddamento" per
titolo (storico e split) e un **`sync.Mutex`** che protegge quelle mappe.

> Cos'è un `Mutex`? Se più parti del programma (le goroutine) lavorano in
> parallelo e condividono le stesse "mappe" (le liste di chi è stato già
> aggiornato), potrebbero scrivere nello stesso momento e corrompere i dati.
> Il `Mutex` (da "mutual exclusion", esclusione reciproca) fa sì che solo una
> goroutine alla volta possa accedere alle mappe: è come una porta con una sola
> chiave. Gli altri programmi la definiscono spesso "lock" (serratura).

### Refresh intelligente (`RefreshStale`)

Per non chiamare Yahoo inutilmente, prima di aggiornare un prezzo controlla
tre criteri:

1. non c'è mai stato un prezzo → scarica;
2. l'ultimo aggiornamento è molto recente → salta (throttle);
3. l'ultimo prezzo di chiusura è già quello del giorno lavorativo atteso →
   salta.

Solo gli asset con `price_source = 'yahoo'` (o vuoto, per sicurezza) vengono
inviati a Yahoo — per quotazioni, storico e split; gli asset con
`price_source = 'manual'` o `'none'` vengono saltati del tutto e non appaiono
tra gli asset obsoleti.

Quando bisogna aggiornare, le quotazioni correnti e i tassi di cambio vengono
presi **in batch** tramite l'endpoint `spark` di Yahoo: una sola chiamata per
gruppi di 50 titoli. I titoli che mancano dalla risposta vengono richiesti uno
a uno.

### Tenere sotto controllo le chiamate

Yahoo limita il numero di chiamate che accetta. Ogni chiamata a Yahoo passa
quindi attraverso due "freni":

1. una **coda FIFO** (primo arrivato, primo servito) che garantisce un
   **intervallo minimo** tra una chiamata e l'altra (default 400ms,
   configurabile con `PECULIUM_YAHOO_MIN_INTERVAL`);
2. un **contatore globale condiviso** (`RateBudget`), implementato in Redis:
   un tetto di richieste per finestra di tempo (default 8 richieste al
   secondo, configurabile con `PECULIUM_YAHOO_GLOBAL_RATE` e
   `PECULIUM_YAHOO_GLOBAL_WINDOW`). Il contatore è condiviso tra server e worker,
   così i due processi insieme non sforano il limite.

Se Redis non è raggiungibile, il contatore viene disattivato e si procede
comunque (si rischia di più, ma l'app non si blocca).

### Riportare i problemi

Quando l'utente chiede un aggiornamento prezzi (`POST /prices/refresh`), la
risposta non è solo la lista dei titoli aggiornati: è un **report** con:

- `refreshed` — i titoli aggiornati con successo;
- `issues` — i problemi, ognuno con un codice stabile:
  `rate_limited` (Yahoo ha rifiutato per troppe chiamate), `http_<status>`
  (un errore HTTP specifico) o `error`;
- `rate_limited` — un riepilogo rapido: "c'è stato un blocco da rate limit?";
- `finished_at` — quando l'aggiornamento è terminato (timestamp UTC; la
  dashboard lo mostra come "Prices updated: …").

Il frontend usa questo report per mostrare un avviso non bloccante se qualche
aggiornamento è fallito.

### Storico e split

- **Storico (`EnsureHistory`)**: riporta i prezzi dalla prima operazione a
  oggi, riprendendo da dove si era rimasti (scarica tutto se manca, altrimenti
  solo dall'ultima data disponibile). Il salvataggio è idempotente per data:
  riscrivere lo stesso giorno non crea duplicati.
- **Storico completo (`HistoryAsset.Full`)**: quando un titolo non è mai stato
  backfillato (`assets.history_backfilled = FALSE`), il primo sync scarica lo
  **storico completo** dei prezzi da Yahoo (dal 1970) in una passata, poi porta
  la flag a `TRUE`. Da lì in poi il sync è incrementale. La pagina asset
  espone questo comando come **"Backfill storico completo"**
  (`POST /assets/{id}/backfill-history`): forza il re-download completo e
  invalida i prezzi in cache.
- **Split (`EnsureSplits`)**: scarica gli eventi di split da Yahoo e li
  salva (anche qui in modo idempotente su `asset_id, date`).

### Profilo ed esposizione dell'asset

Per la pagina di dettaglio asset il backend dialoga anche con l'endpoint
`quoteSummary` di Yahoo (che richiede l'handshake con **crumb** + cookie di
sessione a vita breve, vedi `meta.go`):

- **Profilo (`FetchAssetProfile`)**: il modulo `assetProfile` fornisce il
  `sector` GICS e l'`industry` di una singola azione.
- **Esposizione settoriale (`FetchAssetExposure`)**: per un ETF, il modulo
  `topHoldings` espone `sectorWeightings` (una frazione per chiave di settore),
  che il backend converte nei nostri 11 settori GICS canonici (in percentuale).
  Il fetch è una **preview in sola lettura**: né i pesi di settore né i campi
  profilo (sector/industry/country) vengono persistiti — il salvataggio avviene
  solo con `PUT /assets/{id}/exposure`.
- **Esposizione ETF via JustETF (`FetchETFExposure`)**: il backend può
  scaricare l'esposizione **completa** paesi/regioni e settori di un ETF dal
  microservizio `python-service` (`POST /assets/{id}/fetch-etf-exposure`). Il
  servizio Python legge JustETF (tabelle complete via AJAX "Show more"), il
  package Go `geo` mappa ogni paese a una macro-regione e normalizza i settori
  al set GICS. Se l'asset non ha ISIN, questo viene **risolto automaticamente
  dal ticker** (suffisso `.MI/.DE/.L...` rimosso, risultati ordinati per
  similarità di nome con l'asset) e persistito sull'asset — la scrittura
  dell'ISIN è l'unica cosa che il fetch memorizza. Per il resto il fetch è una
  **preview in sola lettura**: i paesi raw (normalizzati a codici ISO-3166
  alpha-2), le regioni derivate e i settori GICS vengono restituiti nelle tre
  dimensioni — `countries`, `regions` e `sectors` — ma NON persistiti; finiscono
  nel database solo quando l'utente salva con `PUT /assets/{id}/exposure`.
- **Esposizione ETF via Morningstar (`FetchMorningstarExposure`)**: è
  disponibile una seconda fonte: `POST /assets/{id}/fetch-morningstar-exposure`
  (solo ETF; se manca l'ISIN viene auto-risolto via Morningstar sul mercato del
  ticker). L'endpoint del
  python-service `GET /api/v1/etf/{isin}/morningstar-exposure` usa un
  **resolver custom** (modulo `app/morningstar.py`, senza mstarpy): il dominio
  `global.morningstar.com` delle SAL API informali usate da mstarpy è bloccato
  (403), quindi il resolver esegue un **bootstrap Chromium headless** (nel
  container: `chromium`+`chromium-driver`+`xvfb` via apt) che risolve il challenge
  AWS WAF su `www.morningstar.com` e ricava il **Bearer JWT** da
  `/api/v2/stores/maas/token` (cache ~1h); le chiamate dati viaggiano poi via
  `requests` con Bearer+cookie verso `www.us-api.morningstar.com/sal/sal-service/etf/...`
  (settori `portfolio/v2/sector/{sid}/data`, paesi `portfolio/regionalSectorIncludeCountries/{sid}/data`,
  **regioni ufficiali** `portfolio/regionalSector/{sid}/data`,
  ISIN→securityId via `www.morningstar.com/api/v2/search?q={isin}`). I pesi
  paesi sono tenuti come riportati: Morningstar restituisce la lista paesi
  completa (51 voci, molte a 0; la paginazione 10×6 è solo UI client-side) con
  una quota residuale non esposta come paese, quindi la somma è ~95% (nessuna
  forzatura a 100). Le chiavi regione Morningstar (`northAmerica`,
  `unitedKingdom`, `japan`, `australasia`, ...) sono mappate 1:1 sulla
  tassonomia canonica Peculium e restituite come dimensione `regions`. Come
  gli altri fetch, l'endpoint è una **preview in sola lettura**: paesi, settori
  e regioni ufficiali (quando presenti) vengono restituiti ma NON persistiti —
  l'unica scrittura è l'ISIN auto-risolto; il salvataggio avviene con
  `PUT /assets/{id}/exposure`. Altrimenti (o per JustETF) è la preview a
  derivare le regioni dai paesi lato server come dimensione esplicita (la
  derivazione appartiene
  solo al prefill dai provider: il `PUT` manuale con `{countries}` **non**
  riscrive le regioni); il residuo (100 − somma paesi) confluisce
  nella regione `Other / Not Classified`, così le regioni in preview sommano
  sempre a 100.
  Le regioni canoniche sono **10 + `Other`**:
  North America, Latin America, United Kingdom, Europe Developed, Europe
  Emerging, Africa / Middle East, Japan, Australasia, Asia Developed, Asia
  Emerging, Other / Not Classified (UK/Japan/Australasia separate; TW/KR in
  Asia Developed).
- **Asset class (recupero info asset / `GET /assets/meta`)**: Yahoo non
  espone `assetClass` (il modulo `quote` di quoteSummary non esiste, il v7 `/quote` non lo
  restituisce). La classe viene derivata in `FetchMeta` con `geo.ClassifyAssetClass`
  (categoria fondo Morningstar `defaultKeyStatistics.category`/`fundProfile.categoryName`
  con `FetchFundCategory`, più euristica sul nome; default dal tipo).
  È accorpata al recupero info asset (usato alla creazione e da "Aggiorna da Yahoo") e
  **non** alla lettura dei settori: `fetch-exposure` non la aggiorna. L'override manuale
  nell'editor asset vince sempre (aggiornato solo se vuota o `other`).
- **Esposizione geografica**: per una singola **azione**, il paese (dal profilo
  asset) viene mappato a una macro-regione al 100% (`geo.RegionForCountry`).

Nota sull'**ISIN**: Yahoo non espone l'ISIN in nessun modulo. Per gli ETF il
valore viene risolto automaticamente dal ticker tramite il servizio
JustETF; il campo resta comunque modificabile a mano nella pagina asset come
fallback. Le risposte di exposure (`GET/PUT /assets/{id}/exposure`,
`fetch-exposure`, `fetch-etf-exposure`, `fetch-morningstar-exposure`)
includono il campo `isin` persistito (`AssetExposure.ISIN`), così il frontend
può sincronizzarlo dopo un fetch. La risposta `GET /assets/{id}/exposure`
espone la dimensione **countries** zero-filled sull'intera lista ISO canonica, e
`PUT /assets/{id}/exposure` accetta un array opzionale `countries`: tiene solo
codici ISO canonici e **rifiuta una somma paesi sopra 100** (una somma sotto
100 è legittima — la copertura dei provider è spesso ~92–95%, e non c'è un
minimo). Il salvataggio dei paesi **non** tocca la dimensione regioni: ogni
dimensione nel body della `PUT` è salvata indipendentemente, quindi un body
`{countries}` lascia le regioni memorizzate esattamente come sono. Le regioni
si ricalcolano dai paesi solo tramite l'endpoint esplicito di derivazione (o
vengono aggiornate esplicitamente via `{regions}`). Gli utenti possono
aggiungere/rimuovere paesi dalla lista canonica e modificarne i singoli pesi.
Un endpoint complementare `POST /assets/{id}/exposure/derive` calcola le regioni
da un body `{countries}` **senza persistire** (usato dal pulsante "Calcola da
paesi" nella UI).
**Validazione al salvataggio delle regioni**: l'array `regions` esplicito
accetta un totale **≤ 100** (sotto 100 è valido; sopra 100 è rifiutato). La UI
non mostra né modifica mai "Other / Not Classified", quindi quando il client
invia regioni con somma sotto 100 senza riga Other, il backend **inietta il
residuo in `Other / Not Classified` prima di persistere**, mantenendo
l'invariante salvata «le regioni sommano a 100» su cui si basa l'aggregazione
geografica del portafoglio. I settori conservano la regola esatta 100 ± 0,5.
**Provenienza dell'esposizione**: ogni dimensione salvata registra anche *da
dove arriva*. `PUT /assets/{id}/exposure` accetta le stringhe opzionali
`countries_source`, `regions_source` e `sectors_source` (es. `morningstar`,
`justetf`, `yahoo`); quando una dimensione è presente nel body ma la sua
sorgente è assente o vuota viene registrata come `manual`. Le risposte
`GET/PUT /assets/{id}/exposure` includono quindi una mappa `provenance` — solo
per le dimensioni salvate almeno una volta, omessa se vuota — con una entry
`{ "source", "updated_at" }` per dimensione, es. `"countries": { "source":
"morningstar", "updated_at": "2026-09-05T10:00:00Z" }`, così l'UI può mostrare
badge "da Morningstar (2026-09-05)" che sopravvivono a un reload (persistita in
`asset_exposure_provenance`, migrazione `000017`). Le risposte dei fetch/prefill
(`fetch-exposure`, `fetch-etf-exposure`, `fetch-morningstar-exposure`) sono
anteprime e **non** portano provenienza: finché l'utente non salva non si
scrive nulla.

### Cache dell'esposizione dei provider (`exposure:<source>:<ISIN>`)

I due fetch di prefill che interrogano il python-service
(`POST /assets/{id}/fetch-etf-exposure` e
`POST /assets/{id}/fetch-morningstar-exposure`) memorizzano nella cache Redis
delle lookup (`s.repos.Lookup`) il **payload grezzo del provider**, con chiave
`exposure:<source>:<ISIN>` (source è `justetf` o `morningstar`, l'ISIN è
maiuscolo; la chiave Redis completa è `pc:lookup:exposure:...`). La prima
richiesta su un ISIN esegue il fetch pesante (Morningstar richiede una
sessione Chromium/SAL) e lo salva; le successive rispondono dalla cache senza
richiamare il provider. Ogni fonte ha la sua voce: prefillare da JustETF non
scalda Morningstar e viceversa. Il TTL è `PECULIUM_EXPOSURE_CACHE_TTL` (default
7 giorni). I risultati senza paesi non vengono mai cachati, così un fetch
vuoto transitorio non resta in giro per una settimana. La cache accelera solo
la lettura: le voci sono payload del provider, non pesi persistiti (il
salvataggio continua ad avvenire solo con `PUT /assets/{id}/exposure`).
`?refresh=1` (accettato anche `refresh=true`) su uno dei due endpoint forza un
fetch fresco dal provider, saltando la lettura e riscrivendo la cache: è una
leva lato backend. `fetch-exposure` di Yahoo
resta deliberatamente senza cache (è economico e condivide già la cache meta
di Yahoo).

### Invalidation della cache (`bumpRev`)

Ogni lettura in cache (`cached()`, capitolo 7) è chiavata su un **numero di
revisione** globale. Dopo ogni scrittura (nuovi prezzi, un backfill, un
aggiornamento dell'esposizione, ...) il service chiama `bumpRev` così la
lettura successiva salta la cache obsoleta. È anche per questo che
`SyncAssetData` e il sync in background dopo la creazione di un asset
invalidano la cache dei prezzi: altrimenti un asset appena backfillato
continuerebbe a mostrare un grafico vecchio e parziale fino alla scadenza
della TTL di cache.

### La ricerca dei titoli (`lookup_cache`)

Quando l'utente cerca un titolo per ticker, il risultato viene salvato nella
tabella `lookup_cache` per un po' di giorni. Così la prossima ricerca uguale
non rifà la chiamata a Yahoo.

---

## 13. Il worker

Il processo worker (`cmd/worker/main.go`) è separato dal server. Ogni intervallo
(`PECULIUM_PRICE_FETCH_INTERVAL`, default 1 ora) aggiorna i prezzi:

```go
ticker := time.NewTicker(interval)
// esegue subito una volta, poi un ciclo che aspetta:
//   - il prossimo tick (aggiorna i prezzi)
//   - il segnale di chiusura (si spegne)
for { select { case <-ticker.C: FetchAll(); case <-ctx.Done(): return } }
```

Perché un processo separato e non una goroutine dentro il server? **Isolamento**:
se Yahoo va giù, si riavvia il worker senza toccare l'API. Il frontend non si
accorge di nulla.

Il worker, all'avvio, aspetta che il server abbia applicato le migrazioni
(controlla che le tabelle delle serie esistano) e solo dopo parte. A ogni giro,
dopo l'aggiornamento dei prezzi, ricalcola anche le serie giornaliere di tutti
i portafogli (capitolo 9), così i dati restano sempre allineati.

---

## 14. Autenticazione con JWT

L'autenticazione usa **JWT** (JSON Web Token): un token firmato e
"senza stato", cioè il server non deve ricordarsi chi è collegato, perché il
token stesso contiene i dati e una firma crittografica (HMAC-SHA256) che ne
garantisce l'autenticità. Il segreto della firma sta nella configurazione.

Al login vengono emessi **due token**:

- **access** (dura 15 minuti): contiene l'id utente, l'email, il ruolo e il
  tipo `token_type=access`. È il biglietto con cui il frontend chiama le API;
- **refresh** (dura 72 ore): contiene solo l'id utente e il tipo
  `token_type=refresh`. Serve a ottenere una nuova coppia di token quando
  quello di accesso scade (rotazione automatica).

Il middleware controlla ogni richiesta protetta: se il token non c'è, non è
valido, o non è del tipo `access`, la richiesta viene rifiutata.

Le password sono salvate come **hash bcrypt** (non in chiaro).

> Cos'è un *hash*? È una "impronta digitale" della password: una stringa
> calcolata dalla password con una formula a senso unico (non si può tornare
> indietro dalla stampa alla password). Al momento del login si ricalcola
> l'impronta della password inserita e la si confronta con quella salvata.
> Così, anche se qualcuno ruba il database, non può leggere le password.

### Stato dell'account e approvazione delle registrazioni

Ogni account porta con sé uno **stato** — `active`, `pending` (in attesa di
approvazione) o `disabled` — e lo stato viaggia nell'oggetto utente restituito
da register, login e `GET /users/me`.

- Il **primo registrato** su un server vuoto diventa automaticamente l'`admin`
  attivo. Tutto avviene in una sola transazione che prima blocca la riga delle
  impostazioni, così due iscrizioni simultanee non possono essere trattate
  entrambe come la prima.
- I registrati successivi ricevono il ruolo `viewer`. Con l'impostazione di
  server `auto_approve_registrations` **attiva** (il default) sono subito
  `active`; con **disattiva** sono `pending`. La registrazione riesce sempre —
  è lo stato nella risposta a indicare al frontend di mostrare "in attesa di
  approvazione".
- Il login accetta solo account `active`: uno `pending` viene rifiutato con 403
  "account pending approval", uno `disabled` con 403 "account disabled" (le
  stesse regole valgono per il refresh di una coppia di token).

Dopo il controllo JWT, un secondo middleware carica l'account dietro il token
**a ogni** richiesta protetta e lo lascia passare solo mentre lo stato
salvato è `active`: un utente sconosciuto o cancellato riceve 401, uno non
attivo 403. Approvare o disabilitare un account ha quindi effetto immediato,
anche sui token emessi prima della modifica.

### Area admin

I ruoli `owner` e `admin` sono i ruoli da amministratore (un `owner` — il
ruolo di default dello schema — conta come admin ovunque); `editor` e
`viewer` sono utenti normali. Gli endpoint admin stanno sotto
`/api/v1/admin`, dietro il middleware JWT, il middleware dello stato attivo
e un cancello admin che risponde 403 ai non-admin:

| Endpoint | Cosa fa |
|---|---|
| `GET /admin/users` | elenca tutti gli account: id, email, nome, ruolo, stato, created_at (l'hash della password non è mai esposto) |
| `PATCH /admin/users/{id}` | imposta `role` e/o `status` (validati contro gli enum); rifiuta con 409 di degradare o disabilitare **l'ultimo admin attivo** |
| `POST /admin/users/{id}/reset-password` | imposta una nuova password per l'account (minimo 8 caratteri, senza chiedere la corrente) |
| `GET /admin/settings` | legge le impostazioni di server (`auto_approve_registrations`, `updated_at`) |
| `PATCH /admin/settings` | aggiorna `auto_approve_registrations` |
| `GET /admin/db/backup` | scarica il dump completo del database come allegato |
| `POST /admin/db/restore` | sostituisce l'intero database a partire da un dump caricato (distruttivo, richiede conferma) |

### Backup e ripristino del database (`GET /admin/db/backup`, `POST /admin/db/restore`)

Questi due endpoint agiscono sull'**intero database PostgreSQL** (ogni tabella,
ogni utente) eseguendo i tool client `pg_dump` / `pg_restore`, che l'immagine
server del backend include (`postgresql16-client`, PostgreSQL 16); l'immagine
del worker non li contiene. Le coordinate di connessione vengono dalla stessa
configurazione DB dell'applicazione; la password arriva ai tool attraverso la
variabile d'ambiente `PGPASSWORD` (mai sulla riga di comando) e l'impostazione
TLS attraverso `PGSSLMODE`.

`GET /admin/db/backup` esegue `pg_dump -Fc` — un dump completo nel **custom
format** di PostgreSQL (compresso, ripristinabile in modo selettivo) e canalizza
l'output del tool direttamente nella risposta HTTP, senza mai tenere l'intero
archivio in memoria. La risposta è un download: `Content-Type:
application/octet-stream` e `Content-Disposition: attachment` con nome file
`peculium-db-<YYYY-MM-DD>.dump` (data UTC). Gli header dell'allegato vengono
inviati prima che il tool parta, quindi un fallimento di `pg_dump` si manifesta
come download troncato e il dettaglio ripulito finisce nel log del server:
l'output dei tool e le credenziali non sono mai esposti al client.

`POST /admin/db/restore` accetta l'archivio come **caricamento multipart** nel
campo file `dump`, insieme alla conferma esplicita `confirm=replace` (come
campo del form o parametro di query): il ripristino **sostituisce l'intero
database**, quindi la richiesta è rifiutata con 400 senza quell'esatta
conferma, così come una richiesta senza dump file. Il caricamento viene scritto
in un file temporaneo (rimosso dopo l'uso) e passato a
`pg_restore --clean --if-exists --no-owner --no-acl -d <db>`, che elimina e
ricrea gli oggetti presenti nell'archivio. In caso di successo la risposta è un
riepilogo JSON (`dump_bytes`); in caso di fallimento un 500 che riporta l'output
errore di `pg_restore` con la password del database mascherata, così che l'admin
capisca cosa non funziona. Poiché l'archivio contiene anche la tabella `users`,
un ripristino riscrive gli account: sessioni e token già emessi possono
riferire righe non più esistenti, e l'admin viene avvisato di **effettuare di
nuovo il login**.

---

## 15. Operazioni atomiche — `WithTx`

Quando un'operazione coinvolge più scritture sul database, si vuole che
**o riesca tutta, o non cambi nulla**. Questo si chiama transazione atomica.

```go
func (r *Repository) WithTx(ctx, fn func(*Repository) error) error {
    tx, _ := r.DB.Begin(ctx)
    defer func() { _ = tx.Rollback(ctx) }()   // se fn fallisce → annulla tutto
    rr := &Repository{ User: &userRepo{db: tx}, ... }
    if err := fn(rr); err != nil { return err }
    return tx.Commit(ctx)                      // se va tutto bene → salva tutto
}
```

Due idee Go da notare:

1. **`DBTX` interface**: sia il pool di connessioni sia una transazione
   espongono gli stessi metodi (`Exec`, `Query`, `QueryRow`). I repository
   quindi funzionano identici sia sul pool che dentro una transazione.
2. La funzione interna riceve un **Repository "clonato"** i cui componenti
   sono legati alla transazione. Il `defer Rollback` dopo un `Commit`
   riuscito è una no-op innocua: se la funzione fallisce, invece, annulla
   tutto.

Un esempio di uso: l'importazione di un portafoglio in modalità "sostituisci"
cancella e ricrea il portafoglio **atomicamente** — se un passaggio fallisce,
il vecchio portafoglio resta intatto.

### Export/import del portafoglio: versionamento e compatibilità

`GET /portfolios/{id}/export` produce un documento JSON con un campo
`version` (la versione del formato, attualmente `1`); `POST /portfolios/import`
lo consuma. Il formato è deliberatamente **additivo**: i campi `price_source`
e `asset_class` per asset che l'export scrive sono opzionali (`omitempty`),
quindi i documenti che non li contengono restano validi e recuperabili.

Quando l'importer crea un asset il cui ticker non esiste ancora, ogni
informazione mancante viene riempita con un default che soddisfa i vincoli del
database: il nome ripiega sul ticker, il tipo su `stock`, la valuta su `USD`,
`asset_class` sulla classe di default per il tipo e `price_source` su `yahoo`.
Un `price_source` sconosciuto nel documento non fa fallire l'importazione:
anche in questo caso si ripiega su `yahoo`. Un documento con `version` più
recente di quella supportata dall'importer viene rifiutato con un 400 chiaro
(`unsupported export version N`) invece di un errore generico.

### Backup e ripristino per utente (`GET /backup`, `POST /backup/restore`)

`GET /backup` restituisce tutti i dati dell'utente autenticato come un
unico bundle JSON scaricabile (`peculium-backup-<email>-<data>.json`). Come
l'export del portafoglio, il documento è **versionato** (campo `version`,
attualmente `1`) e **additivo**: i campi opzionali possono mancare all'import,
e un bundle più recente di quello che il server capisce viene rifiutato con
un 400 (`unsupported backup version N`). Il bundle contiene:

- `user` — email e nome dell'account da cui il bundle proviene
  (informativi) e la sua **valuta base**, l'unica impostazione utente che
  il ripristino riapplica;
- `currencies` — le voci di valute supportate in uso: la valuta base e
  tutte le valute di portafogli e asset;
- `portfolios` — un normale documento di export del portafoglio per
  ciascun portafoglio **posseduto** dall'utente (i portafogli condivisi in
  ingresso non vengono duplicati); ogni voce è autonoma e importabile anche
  da sola attraverso l'endpoint di import del portafoglio;
- `assets` — i dati per-asset che nessun provider può rifornire, indicizzati
  per ticker: i metadati identificativi, l'**esposizione** (righe paesi,
  regioni e settori con, nei campi `*_source`, la provenienza di ogni
  dimensione salvata) e le **righe dei prezzi manuali**.

I prezzi Yahoo, la cronologia FX e gli eventi health sono esclusi di
proposito: sono dati dei provider, che fetcher e worker rigenerano sul
server ripristinato.

`POST /backup/restore?mode=add|replace` riscrive il bundle sull'account
autenticato — lo scenario previsto è un server reinizializzato da zero, dove
l'utente ha ricreato l'account a mano. Il body è il JSON del bundle stesso.
La modalità decide cosa succede ai dati esistenti:

- `add` (default quando `mode` è omesso) è non distruttivo: ogni
  portafoglio del bundle viene importato come nuovo e i dati esistenti
  restano intatti;
- `replace` elimina prima i portafogli posseduti dall'utente — CASCADE
  porta via le loro transazioni — e poi importa il bundle. Non esiste una
  conferma lato server per `replace`: è la UI client a confermare prima
  della chiamata.

Gli asset sono globali e condivisi tra utenti, quindi il ripristino **non
ne elimina mai nessuno**: un ticker già presente viene riusato così com'è —
i suoi metadati identificativi condivisi non vengono sovrascritti — e uno
assente viene creato dal bundle con gli stessi default dell'importatore del
portafoglio. L'esposizione viene ripristinata solo per le dimensioni che
hanno effettivamente righe (una dimensione assente dal bundle lascia
invariata quella salvata), e i prezzi manuali fanno upsert su
`(asset, data)`. Le voci delle valute aggiungono alla whitelist globale i
codici mancanti e non modificano mai quelli esistenti. Tutta la scrittura
avviene in un'unica transazione e, dopo il commit, le serie di ogni
portafoglio creato vengono ricalcolate.

La risposta è il riepilogo dell'operazione: la `mode` effettivamente
applicata e i contatori `portfolios_created`, `transactions_created`,
`assets_created` e `assets_reused`.

---

## 16. Il ciclo dei dati completo

Mettere insieme tutti i pezzi:

```
L'utente crea un titolo ──► POST /assets ──► CreateAsset
                            in background: scarica split e storico da Yahoo

Il worker, ogni ora: FetchAll (quote + valute in batch) ──► RecomputeAll (serie)

L'utente modifica una transazione ──► Recompute(portafoglio) → serie aggiornate

L'amministratore aggiunge una valuta ──► POST /settings/currencies
                            → verifica conversione su Yahoo → whitelist

L'utente apre la dashboard ──► GET /dashboard:
    posizioni (AVCO) + tassi di cambio
    → summary convertito nella valuta base dell'utente → JSON al frontend

L'utente guarda il grafico dei rendimenti ──► GET /dashboard/performance?granularity=month|year:
    punti giornalieri per asset dalle serie materializzate + FX per data
    + flussi esterni giornalieri (asset con prezzo al valore di mercato,
      senza prezzo al costo)
    → vero TWR mensile/annuale (collegatura geometrica dei rendimenti
      giornalieri) + TWR cumulativo, nella valuta base, con serie
      invested/value → JSON al frontend

L'utente guarda il grafico dei rendimenti di un portafoglio ──► GET /portfolios/{id}/performance/buckets?granularity=month|year:
    stesso modello TWR giornaliero per un solo portafoglio, nella valuta del
    portafoglio (serie senza conversione; solo i flussi la richiedono)
    → JSON al frontend

L'utente apre la pagina asset ──► GET /assets/{id}/quote (+ /prices?...&full=1):
    range di quota + storico prezzi dal database → JSON al frontend

L'utente modifica l'esposizione ──► PUT /assets/{id}/exposure
    → salva asset_country_weights / asset_region_weights / asset_sector_weights
      (ogni dimensione salvata indipendentemente: i paesi non riscrivono le
       regioni; somma paesi non vincolante) → bumpRev

L'utente clicca "Calcola da paesi" ──► POST /assets/{id}/exposure/derive
    → {countries} → {regions} derivate via AggregateRegions (nessuna persistenza) → riempie la tabella regioni

L'utente clicca "Aggiorna da Yahoo" ──► POST /assets/{id}/fetch-profile
    → quoteSummary (crumb) → settore/industria (+ paese emittente) → salvati sull'asset

L'utente precompile l'esposizione ──► POST /assets/{id}/fetch-exposure
                                    | /fetch-etf-exposure | /fetch-morningstar-exposure
    → dati del provider (settori Yahoo | python-service paesi + settori
      + regioni ufficiali, GET /api/v1/etf/{isin}/[morningstar-]exposure)
      restituiti come PREVIEW in sola lettura — nulla persistuto tranne
      l'ISIN auto-risolto → bumpRev solo su quella scrittura
    → l'utente conferma coi pulsanti Salva → PUT /assets/{id}/exposure (persistenza)
```

---

## 17. Mappa dei file

```
backend/
├── cmd/
│   ├── server/main.go      # avvio API, routing, migrazioni, backfill serie
│   ├── worker/main.go      # aggiornamento prezzi in background
│   └── secrets/main.go     # secrets-init: genera i segreti al primo avvio
├── internal/
│   ├── auth/jwt.go         # JWT: generazione, validazione, middleware
│   ├── config/config.go    # variabili d'ambiente + DSN di connessione
│   ├── geo/geo.go          # macro-regioni, settori GICS, paesi ISO canonici, mappatura paese→regione
│   ├── handler/            # livello HTTP (auth.go, admin.go, admin_db.go, backup.go, portfolio.go, settings.go, ...)
│   ├── middleware/         # controlli d'account per richiesta: stato attivo + cancello admin
│   ├── model/              # strutture dati con tag JSON
│   ├── position/           # motore AVCO (State, Apply, Walk)
│   ├── price/              # client Yahoo (yahoo.go, spark.go, meta.go, throttle.go, report.go, ...) + fetcher JustETF/Morningstar
│   ├── repository/         # query SQL (repository.go = "hub" + asset.go + exposure.go + settings.go + WithTx + DBTX)
│   ├── secrets/            # materializzazione dei segreti per l'immagine secrets-init (Ensure, Health)
│   ├── series/             # serie giornaliere materializzate (Recompute, LoadRates, FxFactor)
│   └── service/            # logica di business (service.go, admin.go, backup.go, dbmaint.go)
├── migrations/             # SQL versionato (000001..000019)
└── go.mod
```

---

## 18. Leggere il codice Go

Nel documento compaiono frammenti di codice Go. Ecco le basi per leggerli
senza aver mai programmato.

- **Le istruzioni finiscono alla riga.** Ogni riga è un'istruzione: niente
  parentesi o punti e virgola da cercare. I `//` indicano un **commento**: il
  testo dopo `//` non è codice, è una nota per chi legge.
- **`func` definisce una funzione**: un blocco di codice con un nome, pronto a
  essere richiamato. Es. `func (s *Service) GetDashboard(...) { ... }`. Il nome
  prima di `(` indica *a quale oggetto appartiene* (in questo caso al service).
- **Le parentesi `{ }` delimitano il corpo** della funzione: tutto quello che
  sta tra le graffe è ciò che la funzione fa.
- **`:=` crea una variabile** e le assegna un valore in una volta sola.
  Es. `x := 5` significa "crea una casella chiamata x e mettici 5".
- **`struct` raggruppa valori**: una scheda con più caselle con nome (come
  nella "scheda dello stato" del capitolo 8).
- **`package` e `import`**: `package` dichiara il "gruppo" a cui appartiene un
  file; `import` carica codice già scritto da altri (le librerie).
- **`context.Context`**: un "portabagagli" che accompagna ogni richiesta e
  contiene informazioni di contorno (chi è l'utente, quando fermarsi). Quando
  l'utente chiude la pagina, il contesto lo comunica e i lavori in corso si
  fermano.
- **Le funzioni restituiscono più valori**, e il secondo è quasi sempre
  `err` (errore). La regola Go: **gli errori sono valori da controllare**.
  `if err != nil { return err }` si legge così: "se c'è stato un errore,
  fermati e restituisci l'errore". È il modo in cui Go segnala i problemi:
  non interrompe il programma all'improvviso, lo fa decidere a chi scrive il
  codice, riga per riga.

Con queste poche regole, i frammenti di codice nel documento si leggono come
frasi: "crea la connessione, se va male fermati e segnala, altrimenti continua".
