# Peculium — Fase 1: Specifica Dettagliata

## Panoramica

Prima release funzionante: autenticazione multi-utente, gestione portafogli e asset, dashboard con statistiche base.

---

## Moduli

### 1. Autenticazione & Utenti

- [ ] Registrazione con email + password
- [ ] Login con JWT (access + refresh token)
- [ ] Profilo utente (nome, avatar, preferenze)
- [ ] Ruoli: `owner` (chi crea), `admin`, `editor`, `viewer`
- [ ] Preliminari per multi-tenancy familiare (invito via email)

### 2. Asset & Categorie

- [ ] Asset finanziari: azioni, ETF, obbligazioni, fondi, crypto, materie prime
- [ ] Dati per asset: ticker, ISIN, nome, valuta, paese, settore GICS
- [ ] Classificazione gerarchica: Categoria → Settore → Industria
- [ ] Ricerca asset con autocomplete (da DB locale + arricchimento Yahoo Finance)
- [ ] Upload manuale o da file CSV (per batch import)

### 3. Portafogli

- [ ] CRUD portafogli
- [ ] Ogni portafoglio ha: nome, descrizione, valuta base, note
- [ ] Un utente può avere N portafogli (es. "Pension fund", "Trading", "Kids education")
- [ ] Condivisione: un portafoglio può essere visibile ad altri utenti con permessi granulari

### 4. Transazioni

- [ ] Tipi: Buy, Sell, Dividend, Split, Fee
- [ ] Dati transazione: asset, quantità, prezzo, valuta, data, commissioni, note
- [ ] Supporto valute multiple (con tasso di cambio al momento della transazione)
- [ ] Storico completo transazioni con filtri e ricerca
- [ ] Lotto minimo (es. frazioni di azioni)

### 5. Prezzi di Mercato

- [ ] Fetch automatico prezzi via Yahoo Finance (gratuito)
- [ ] Cache prezzi su DB (tabella prices) con timestamp
- [ ] Fallback: se fetch fallisce, usa ultimo prezzo disponibile
- [ ] Refresh periodico (configurabile: ogni ora/giorno)
- [ ] Possibilità di inserimento manuale prezzo
- [ ] Storico prezzi per calcolo ROI e grafici

### 6. Dashboard & Statistiche (Base)

Visualizzazioni minime della prima release:

- [ ] **Valore totale portafoglio** (somma posizioni a prezzo corrente)
- [ ] **Gain/Loss totale** e percentuale
- [ ] **Gain/Loss realizzato** (da vendite) vs **non realizzato** (da posizioni aperte)
- [ ] **ROI per asset** (ritorno sull'investimento)
- [ ] **Asset allocation** (grafico a torta per tipo/asset)
- [ ] **Andamento storico valore portafoglio** (line chart over time)
- [ ] **Top performer / worst performer**

### 7. API (Endpoint principali)

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh

GET    /api/v1/users/me
PATCH  /api/v1/users/me

GET    /api/v1/assets
GET    /api/v1/assets/:id
POST   /api/v1/assets
GET    /api/v1/assets/search?q=   ← autocomplete

GET    /api/v1/portfolios
POST   /api/v1/portfolios
GET    /api/v1/portfolios/:id
PATCH  /api/v1/portfolios/:id
DELETE /api/v1/portfolios/:id

GET    /api/v1/portfolios/:id/transactions
POST   /api/v1/portfolios/:id/transactions

GET    /api/v1/portfolios/:id/performance
GET    /api/v1/portfolios/:id/allocation
GET    /api/v1/portfolios/:id/roi

GET    /api/v1/prices/:asset_id
POST   /api/v1/prices/refresh
```

---

## Cosa NON include la Fase 1

- Backtest e Monte Carlo → **Fase 2**
- Gestione spese e budget → **Fase 4**
- Obiettivi di risparmio → **Fase 4**
- Mobile app nativa → futuro

---

## Criteri di completamento Fase 1

1. Un utente può registrarsi, creare portafogli, inserire transazioni
2. I prezzi vengono aggiornati automaticamente (o manualmente)
3. La dashboard mostra valore, gain/loss e allocazione
4. Un secondo utente può essere invitato e vedere i portafogli condivisi
5. `docker compose up` fa partire tutto
