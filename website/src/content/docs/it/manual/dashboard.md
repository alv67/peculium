---
title: Dashboard e analisi
order: 4
description: Come leggere ciò che Peculium calcola per te — valore netto, performance e allocazioni.
---

Tutto ciò che registri nell'[Uso quotidiano](/it/manual/daily-use) viene
calcolato automaticamente; questo capitolo spiega come leggere il risultato.
Il modello dietro i numeri è in [Concetti](/it/manual/concepts).

## La dashboard

Appena entrato, la dashboard mostra il quadro completo: **Valore netto**,
utile/perdita complessivo e **Dividendi**, con il **Dettaglio** degli
**Investimenti** — investito, valore/ricavi, utile/perdita e realizzato,
su righe **Attive** e **Chiuse**. Accanto ai numeri principali:

- il grafico **Valore vs investito**, con i chip di periodo **1Y / 3Y / TUTTO**;
- la card **Performance**: barre **Mensili** o **Annuali** con la linea
  **TWR cumulativo**;
- la ciambella **Allocazione per portafoglio**, che mostra come il patrimonio
  è distribuito tra i tuoi portafogli.

Usa il selettore **Ambito** per concentrare tutta la pagina su un singolo
portafoglio invece della vista consolidata.

![La dashboard](/screenshots/it/manual/analytics-dashboard.png)

*La dashboard — valori di testa, grafico valore-vs-investito, performance e allocazione in un colpo d'occhio.*

## Il dettaglio del portafoglio

Aprire un portafoglio mostra la stessa immagine in formato ridotto: la scheda
**Panoramica** porta la striscia di KPI — valore netto, investito,
utile/perdita, dividendi, attive contro chiuse — e la vista performance del
singolo portafoglio. Le altre schede sono l'elenco [Attività](/it/manual/daily-use)
e la sua allocazione.

![La scheda Panoramica di un portafoglio](/screenshots/it/manual/analytics-portfolio.png)

*Panoramica del portafoglio — gli stessi KPI calcolati su un solo portafoglio.*

## Allocazioni

La scheda **Allocazione** scompone il portafoglio per **classe di attività**,
per geografia — **Paesi** e **Regioni** — e per **settore**. Geografia e
settori coprono solo l'universo azionario: le attività non azionarie sono
escluse e la copertura azionaria rimanente è dichiarata, così i grafici non
possono sovrastimare la precisione in silenzio. La dashboard mostra le stesse
viste consolidate (**Allocazione complessiva**). Due abitudini di interazione
valgono la pena:

- **Clicca una fetta o una barra** per aprire il dettaglio: un pannello elenca
  gli asset contribuenti di quel gruppo con **Valore**, **Contributo** e
  **Quota della fetta**.
- **Passa un grafico alla vista tabella** (Grafico / Tabella) quando vuoi
  numeri esatti e non forme.

![La scheda Allocazione del portafoglio](/screenshots/it/manual/analytics-allocations.png)

*Scheda Allocazione — ciambelle per classe, geografia e settore; clicca una fetta per vedere gli asset dietro.*

## Leggere i numeri

- **Posizioni attive contro chiuse**: una posizione è attiva finché detieni
  l'asset; una volta venduta completamente, si chiude.
- **Non realizzato contro realizzato**: le posizioni aperte mostrano
  l'utile/perdita calcolato sull'ultimo prezzo — non l'hai ancora incassato.
  Le posizioni chiuse mostrano il realizzato, definito dal ricavato della
  vendita contro il costo.
- I **dividendi** sono incasso in contanti, contato a parte rispetto alle
  variazioni di prezzo.
- Tutto è consolidato nella tua **valuta base** (vedi
  [Concetti](/it/manual/concepts)): i tassi di cambio vengono applicati e,
  quando un tasso manca, l'importo viene segnalato ed escluso invece di
  essere convertito male in silenzio. I valori usano i prezzi più aggiornati
  recuperati per te — l'indicazione «Prezzi alle …» vicino ai numeri di testa
  mostra a quando risalgono.

Metti alla prova i numeri: l'[Avvio rapido](/it/manual/quickstart) porta da
un'istanza vuota a grafici reali in cinque minuti.
