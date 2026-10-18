---
title: Dati e sincronizzazione
order: 5
description: Da dove arrivano i prezzi, cosa significano gli avvisi e come verificare una sincronizzazione sana.
---

Ogni grafico vale quanto i prezzi che ha dietro. Questo capitolo copre da
dove arrivano, cosa succede quando mancano e come controllare la
sincronizzazione. Il lato delle valute è in [Concetti](/it/manual/concepts);
la pagina dell'asset è già trattata nell'[Uso quotidiano](/it/manual/daily-use).

## Da dove arrivano i prezzi

I prezzi vengono da **Yahoo Finance**. Peculium conserva uno storico prezzi
locale ed esegue ogni lavoro verso l'esterno — aggiornamenti prezzi, download
dello storico, metadati, controllo split — come **job in background**: la
richiesta risponde subito e le pagine si aggiornano da sole quando i dati
arrivano, così una fonte lenta non blocca mai l'app. La barra di testata porta
il timbro **"Prezzi alle {ora}"**, e i valori consolidati usano esattamente
quei prezzi. Ti servono adesso, non dopo? **Aggiorna prezzi** accoda
quotazioni fresche su richiesta.

![La testata dell'app con il timbro di freschezza](/screenshots/it/manual/data-header.png)

*La testata — «Prezzi alle …» dice quali prezzi stanno usando i tuoi numeri.*

## Qualità dei dati

Quando un recupero non riesce, sotto la testata compare una fila di chip:
ognuno nomina il problema e linka dove risolverlo — alcuni prezzi non
aggiornati per la **limitazione Yahoo** o per un aggiornamento
**parzialmente fallito**, oppure un importo **escluso — cambio mancante** per
alcune posizioni. Meglio escludere che inventare: una posizione senza tasso
utilizzabile viene lasciata fuori e segnalata, mai convertita male in
silenzio. Un asset senza prezzi mostra **senza prezzo**: il suo valore è
**mantenuto al costo** — quanto hai pagato — così il suo P/L risulta zero
invece di tirare a indovinare.

## Storico prezzi e completezza

Ogni pagina asset grafica lo **storico prezzo** registrato e dice quanto è
completo. Mentre un download è in corso vedi una **barra di avanzamento**
invece di un grafico vuoto, e si aggiorna da sola man mano che arrivano le
barre. Quando lo storico **manca**, il grafico offre il pulsante **Backfill
storico completo**; quando è scaricato solo **in parte**, una nota lo segnala
con lo stesso pulsante per completarlo. Il backfill è disponibile in ogni
momento anche dalla scheda **Dati** dell'asset. La stessa scheda offre
**Aggiorna da Yahoo**, che però rinnova i metadati dell'asset, non i prezzi.

Borse diverse chiudono in orari diversi, quindi l'ultimo close di ogni asset
può avere una data diversa: è normale. La lista degli asset mostra l'ultimo
close di ciascuno **con la sua data**, e la pagina dell'asset etichetta
quella data come **"Dati al {date}"**.

![Il grafico dello storico prezzi di un asset](/screenshots/it/manual/data-price-history.png)

*Storico prezzo — il backfill scarica l'intera serie come job in background.*

## Fonte prezzi per asset

La scheda **Dati** dell'asset decide da dove arrivano i suoi prezzi:

- **Yahoo Finance** — il default: quotazioni e backfill automatici. Usala per
  tutto ciò che Yahoo quota per ticker.
- **Prezzo manuale** — inserisci prezzi datati a mano, per asset senza fonte
  automatica (un fondo non quotato, un'obbligazione a tasso concordata).
- **Nessun prezzo** — escludi l'asset: resta valorizzato al costo, quanto
  investito.

## La pagina Dati e sincronizzazione

**Dati e sincronizzazione** nella sidebar è un unico elenco di tutto ciò che
l'app scambia con l'esterno: ogni **job** in background (aggiornamento prezzi,
backfill storico, sincronizzazione asset, backfill metadati, recupero split)
compare come una riga con il suo avanzamento o errore, e ogni singola
**chiamata** ai provider — prezzi, cambi, split, ricerche, letture di
esposizione ETF — le compare accanto, dalla più recente. Ogni job indica la
propria destinazione: il ticker dell'asset o il nome del portafoglio.

- Il selettore della **finestra** — **24h / 7 giorni / Tutto** — limita
  insieme le caselle di riepilogo (tasso di successo, successi, fallimenti,
  richieste limitate) e l'elenco, così i numeri corrispondono sempre a quanto
  elencato.
- I filtri **Tipo** e **Stato** riducono l'elenco, per esempio ai soli
  fallimenti di un tipo di chiamata.
- **Mentre ci sono job in corso** la pagina si aggiorna da sola; in ogni caso
  **Aggiorna ora** ri-interroga subito il referto.
- Aprendo un job vedi le **sue chiamate** in un pannello laterale proprio
  accanto all'elenco — il dettaglio di cosa ha scaricato, cosa è fallito e
  quanto tempo ha preso.
- L'attività più vecchia di **90 giorni** viene rimossa automaticamente, così
  la pagina non cresce all'infinito; un amministratore può anche svuotare
  tutta l'attività con il pulsante **Cancella attività**, dopo una conferma.

Quando un chip di qualità continua a tornare, questa pagina dice se è Yahoo in
generale a dare problemi o se fallisce un asset specifico.

![L'elenco di attività di Dati e sincronizzazione](/screenshots/it/manual/data-sync-health.png)

*Dati e sincronizzazione — job e chiamate ai provider in un unico elenco, con
finestra temporale, filtri e caselle di riepilogo.*

Una sincronizzazione silenziosa significa grafici onesti: guarda in
[Dashboard e analisi](/it/manual/dashboard) cosa dicono i numeri dei tuoi
portafogli.
