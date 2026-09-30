---
title: Concetti
order: 2
description: Portafogli, asset, transazioni e valute — le quattro idee su cui si regge Peculium.
---

Tutto in Peculium nasce da quattro concetti: registri **transazioni** su
**asset** all'interno di **portafogli**, e le **valute** rendono i numeri
confrontabili. L'[Avvio rapido](/it/manual/quickstart) li ha già messi in
pratica; questo capitolo spiega cosa sono e come si combinano.

## Portafoglio

Un portafoglio è un contenitore di posizioni con una propria valuta. Puoi
averne quanti ne vuoi — una suddivisione tipica è un portafoglio per il lungo
periodo e uno per il trading attivo — e ciascuno viene tracciato, graficato e
riepilogato a sé. Nella dashboard sono aggregati tutti.

![L'elenco dei portafogli](/screenshots/it/manual/concepts-portfolios.png)

*Portafogli — una scheda per portafoglio, ciascuna con la valuta in cui si esprime.*

## Asset

Un asset è uno strumento che puoi detenere: un'azione, un ETF, un'obbligazione.
Porta con sé ticker, nome, tipo, valuta e classe di asset. Gli asset sono un
**catalogo condiviso**, non una copia per portafoglio: uno strumento viene
registrato una sola volta e tutti i portafogli che lo detengono puntano allo
stesso record — aggiungere un ticker già esistente lo riusa, senza duplicarlo.
Exchange, ISIN e gli altri metadati restano modificabili nella pagina dell'asset.

![L'elenco degli asset](/screenshots/it/manual/concepts-assets.png)

*Asset — il catalogo condiviso; la colonna Valuta mostra in che valuta è quotato ciascun asset.*

## Transazione

Una transazione è ciò che effettivamente registri, dentro un portafoglio:
**acquisto**, **vendita**, **dividendo**, **split** o **commissione**, con
quantità, prezzo e data — più commissioni e note facoltative. Posizioni,
valori e performance che Peculium mostra sono tutti calcolati a partire dalle
tue transazioni e dallo storico prezzi; le posizioni non si modificano mai direttamente.

![La scheda Attività di un portafoglio](/screenshots/it/manual/concepts-transactions.png)

*Attività — tutte le transazioni di un portafoglio, filtrabili per tipo.*

## Valute e valuta base

Gli asset sono quotati nella loro valuta e i portafogli si esprimono nella
loro: un'istanza tipica mescola USD ed EUR (e spesso di più). Due impostazioni
governano il miscuglio:

- Il **catalogo delle valute** (Impostazioni → **Valute**) elenca le valute
  gestite dall'istanza: solo queste sono proposte nella scelta della valuta di
  un portafoglio o di un asset.
- La tua **valuta base** (Impostazioni → **Profilo**) è l'unica valuta con cui
  la dashboard consolida i valori di tutti i tuoi portafogli, convertendo le
  altre con tassi di cambio.

![L'elenco delle valute gestite](/screenshots/it/manual/concepts-currencies.png)

*Impostazioni → Valute — il catalogo delle valute gestite da questa istanza.*

![Il modulo del profilo con il selettore della valuta base](/screenshots/it/manual/concepts-base-currency.png)

*Impostazioni → Profilo — scegli la valuta base in cui vuoi vedere i totali.*

---

Questo è l'intero modello: le transazioni spostano asset dentro e fuori dai
portafogli, e la valuta base rende il risultato confrontabile. Per metterlo in
pratica, riparti dall'[Avvio rapido](/it/manual/quickstart) oppure guarda cosa
sanno fare questi numeri nella pagina [Funzionalità](/it/features).
