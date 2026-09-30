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
locale, aggiorna le quotazioni più recenti in background e ti dice quanto
sono freschi i numeri: la barra di testata porta il timbro **"Prezzi alle
{ora}"**, e i valori consolidati usano esattamente quei prezzi. Ti servono
adesso, non dopo? **Aggiorna prezzi** li recupera su richiesta.

![La testata dell'app con il timbro di freschezza](/screenshots/it/manual/data-header.png)

*La testata — «Prezzi alle …» dice quali prezzi stanno usando i tuoi numeri.*

## Qualità dei dati

Quando un recupero non riesce, sotto la testata compare una fila di chip:
ognuno nomina il problema e linka dove risolverlo — alcuni prezzi non
aggiornati per la **limitazione Yahoo**, un numero di aggiornamenti prezzi
non riusciti, oppure un importo **escluso — cambio mancante** per alcune
posizioni. Meglio escludere che inventare: una posizione senza tasso utilizzabile
viene lasciata fuori e segnalata, mai convertita male in silenzio. Un asset
senza prezzi mostra **senza prezzo**: il suo valore è **mantenuto al costo**
— quanto hai pagato — così il suo P/L risulta zero invece di tirare a indovinare.

## Storico prezzi e backfill

Ogni pagina asset grafica lo **storico prezzo** registrato. Quando lo storico
è rado o assente — un asset appena registrato, un buco nei record — lancia
**Backfill storico completo** dal menu dell'asset (o dalla scheda **Dati**):
Peculium scarica l'intero storico giornaliero da Yahoo e il grafico si
riempie. Gli stessi menu offrono **Aggiorna da Yahoo**, che però rinnova i
metadati dell'asset, non i prezzi.

![Il grafico dello storico prezzi di un asset](/screenshots/it/manual/data-price-history.png)

*Storico prezzo — il backfill scarica l'intera serie in un colpo solo.*

## Fonte prezzi per asset

La scheda **Dati** dell'asset decide da dove arrivano i suoi prezzi:

- **Yahoo Finance** — il default: quotazioni e backfill automatici. Usala per
  tutto ciò che Yahoo quota per ticker.
- **Prezzo manuale** — inserisci prezzi datati a mano, per asset senza fonte
  automatica (un fondo non quotato, un'obbligazione a tasso concordata).
- **Nessun prezzo** — escludi l'asset: resta valorizzato al costo, quanto
  investito.

## La pagina Dati e sincronizzazione (admin)

Gli amministratori hanno un referto di salute della sincronizzazione sotto
**Dati e sincronizzazione**: il **Tasso di successo** con i totali di
**successi**, **fallimenti** e **richieste limitate**, e la tabella **Eventi
recenti** — timestamp, tipo, stato, codice, messaggio, durata. Scegli
la finestra con il selettore di periodo (**Oggi / Ultime 24h / Ultimi 100**)
e premi **Aggiorna ora** per ricaricare il referto. Quando un chip di qualità
continua a tornare, questa pagina dice se è Yahoo in generale a dare problemi
o se fallisce un asset specifico.

![La pagina Dati e sincronizzazione](/screenshots/it/manual/data-sync-health.png)

*Dati e sincronizzazione — tasso di successo, totali ed eventi recenti della sincronizzazione prezzi.*

Una sincronizzazione silenziosa significa grafici onesti: guarda in
[Dashboard e analisi](/it/manual/dashboard) cosa dicono i numeri dei tuoi
portafogli.
