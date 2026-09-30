---
title: Avvio rapido
order: 1
description: Dall'istanza appena avviata alla tua prima dashboard, in pochi minuti.
---

Questo capitolo ti accompagna da zero al primo risultato in Peculium. La
configurazione completa della stack vive nella [pagina di
installazione](/install); qui diamo per assunto che Peculium sia già in esecuzione.

## Cosa ti serve

Docker e una stack Peculium avviata seguendo la [guida all'installazione](/install).
Serve poi solo un browser puntato sulla web app (in default `http://localhost:3000`).

## Primo accesso

La schermata di login è anche il punto in cui crei il tuo account: scegli
**Registrati**, inserisci email e password, poi accedi. Il primo account creato
su un'istanza nuova diventa l'**amministratore**, cioè l'utente che gestisce gli altri.

![La schermata di login di Peculium](/screenshots/it/manual/quickstart-login.png)

*Login e registrazione condividono la stessa schermata: accedi, o registra il primo account.*

## Crea il tuo primo portafoglio

Vai su **Portafogli** e apri **Crea portafoglio**. Basta un nome; la descrizione
è facoltativa e la valuta (USD per default) è quella in cui il portafoglio
esprime i suoi totali. Il portafoglio è il contenitore in cui registri le
transazioni: ad esempio uno per broker o per strategia.

![La pagina dei portafogli con il tuo nuovo portafoglio vuoto](/screenshots/it/manual/quickstart-portfolio.png)

*Un portafoglio vuoto, pronto da riempire.*

## Aggiungi un asset

Gli asset sono gli strumenti che detieni, identificati dal ticker Yahoo Finance.
Vai su **Asset**, clicca **Aggiungi**, digita un ticker come `AAPL` e selezionalo
dalla ricerca: Peculium lo registra con i suoi dettagli. Exchange, ISIN e gli
altri metadati restano modificabili in seguito nella pagina dell'asset.

## Registra la tua prima transazione

Apri il portafoglio, clicca **Aggiungi transazione** e scegli l'asset appena
registrato. Seleziona il tipo — **acquisto**, **vendita** o **dividendo** —
inserisci quantità, prezzo e data, poi salva. Commissioni e note sono facoltative.

![Il dialogo di aggiunta transazione compilato per un acquisto](/screenshots/it/manual/quickstart-add-transaction.png)

*Il primo acquisto: asset, tipo, quantità e prezzo bastano per vedere numeri reali.*

## Scopri la dashboard

Apri la **Dashboard**: il valore totale delle tue posizioni, l'utile/perdita, le
ciambelle di allocazione e il grafico delle performance sono calcolati con i
prezzi che Peculium aggiorna per te, aggregati su tutti i tuoi portafogli.

![La dashboard con le tue prime posizioni](/screenshots/it/manual/quickstart-dashboard.png)

*La dashboard è la prima schermata dopo il login: già una transazione la rende realistica.*

## E adesso?

- La pagina [Funzionalità](/features) mostra tutto il resto: allocazioni,
  esposizione ETF, viste multi-valuta.
- La [guida all'installazione](/install) copre la gestione e l'aggiornamento della stack.
- Gli altri capitoli di questo manuale compaiono nella barra laterale e sulla
  [home del manuale](/manual).
