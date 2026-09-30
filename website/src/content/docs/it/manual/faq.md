---
title: Domande frequenti
order: 9
description: Risposte brevi alle domande che ricorrono più spesso, ognuna con il capitolo che approfondisce.
---

Prima la risposta breve, poi il link per la storia completa. Se la tua
domanda non è qui, la barra laterale elenca tutto il resto del manuale.

## Non riesco ad accedere — il mio account è "in approvazione"

La registrazione può richiedere un'approvazione: se il server ha
l'approvazione automatica disattiva, dopo la registrazione compare "Un
amministratore deve approvare il tuo account" e l'accesso resta negato finché
non succede. Un amministratore ti approva in **Admin → Utenti** (la riga
mostra il badge In approvazione) — vedi
[Amministrazione](/it/manual/administration). Un account disattivato ha
bisogno della stessa pagina: un amministratore lo **riattiva**.

## I prezzi non si aggiornano — vedo un avviso di limitazione

Yahoo Finance limita il traffico di richieste intenso, e Peculium lo mostra
con onestà nella striscia sotto la testata ("Alcuni prezzi non aggiornati
(limitazione Yahoo)"). Non c'è nulla da riparare lato tuo: il limite passa, e
puoi recuperare quotazioni fresche su richiesta con **Aggiorna prezzi**. Il
capitolo [Dati e sincronizzazione](/it/manual/data-sync) — e la pagina di
salute **Dati e sincronizzazione** per gli admin — mostrano cosa è successo e
quando.

## Un asset mostra "senza prezzo" e il suo valore è mantenuto al costo

La fonte prezzi di quell'asset è **Nessun prezzo** (oppure **Prezzo manuale**
senza ancora una voce). Mantenuto al costo significa che la posizione è
valorizzata a quanto hai pagato, così il suo P/L risulta zero invece di
tirare a indovinare. Imposta la fonte su **Yahoo Finance** per tutto ciò che
Yahoo quota, oppure registra prezzi datati a mano in Prezzo manuale: entrambe
le scelte sono nella scheda **Dati** dell'asset; vedi [Dati e
sincronizzazione](/it/manual/data-sync) e il modello in
[Concetti](/it/manual/concepts).

## Una posizione è esclusa con "cambio mancante"

La dashboard consolidata ha bisogno di un tasso di cambio dalla valuta della
posizione alla tua **valuta base**; quando nessun tasso è disponibile,
l'importo viene segnalato ed escluso invece di essere convertito male in
silenzio. Si risolve gestendo quella valuta (Impostazioni → catalogo
**Valute**) — la meccanica è in
[Valute e valuta base](/it/manual/concepts#valute-e-valuta-base).

## Ho dimenticato la password

Le istanze self-hosted non inviano link di reset via email. Se sei fuori, un
amministratore può **reimpostare la password** da **Admin → Utenti**;
dovresti cambiarla subito dopo l'accesso. Finché riesci a entrare, impostane
una nuova tu in Impostazioni → **Password** ([Impostazioni](/it/manual/settings),
[Amministrazione](/it/manual/administration)).

## Aggiungi o sostituisci nel ripristino di un backup?

**Aggiungi ai dati attuali** è non distruttiva: i portafogli del backup
vengono importati come nuovi e nulla di ciò che hai viene toccato.
**Sostituisci i dati attuali** prima elimina i tuoi portafogli e le transazioni
esistenti (gli asset non sono mai eliminati: il catalogo è condiviso). Leggi
[Backup e ripristino](/it/manual/backup) prima di usare "sostituisci".

## Dove sono salvati i miei dati — e come cambio server?

Sulla tua macchina, nel database dentro la stack di Peculium: non esce mai
dalla tua rete, a parte i dati di mercato scaricati da Yahoo. Per passare a
un nuovo server, avvia la stack con la [guida
all'installazione](/it/install), poi ripristina il **backup** JSON di ogni
utente sulla nuova istanza oppure — per una migrazione completa che include
gli utenti — sposta il dump di **Backup del server** lato admin. Dettagli in
[Backup e ripristino](/it/manual/backup).

## Può usarlo più di una persona?

Sì: è multi-utente per progettazione, con account reali e spazi di lavoro per
singolo — ognuno mantiene i propri portafogli. I ruoli
(**Amministratore** / **Editor** / **Visualizzatore**) decidono chi può
gestire l'istanza, e le registrazioni sono approvate (o auto-approvate) da un
amministratore. Il quadro completo è in
[Amministrazione](/it/manual/administration).
