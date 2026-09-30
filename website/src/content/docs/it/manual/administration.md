---
title: Amministrazione
order: 8
description: Ruoli, gestione degli utenti e interruttori a livello di server — l'area riservata agli amministratori.
---

Peculium è multi-utente, e qualcuno deve pure tenere le chiavi. Quel qualcuno
è l'amministratore — questo capitolo racconta cosa offre la sua area.

## Chi è un amministratore

Su un server nuovo di zecca, il **primo account registrato** — quello che
crei nell'[Avvio rapido](/it/manual/quickstart) — diventa amministratore. I
ruoli sono **Proprietario**, **Amministratore**, **Editor** e
**Visualizzatore**; il **Proprietario** legacy si comporta esattamente come
l'**Amministratore** e non può essere assegnato a nuovi account. L'area è
visibile solo agli amministratori: chiunque altro provi ad aprirla legge
«Solo amministratori — non hai accesso a quest'area».

## Utenti — Admin → Utenti

Il sottotitolo della pagina dice il mestiere: *"Approva le registrazioni,
gestisci i ruoli e reimposta le password."* La tabella elenca ogni account
con nome, email, **Ruolo**, **Stato** — Attivo, **In approvazione**,
Disattivato — e data di creazione. Per riga puoi:

- **Approvare** una registrazione in attesa, permettendo al titolare di accedere;
- **disattivare** un account (dopo una conferma — non potrà accedere finché
  un amministratore non lo **riattiva**);
- cambiare il **ruolo** dal menu a tendina della riga;
- **reimpostare la password** di un utente: ne imposti una nuova per
  l'account, e l'utente dovrà cambiarla dopo l'accesso.

Un guardrail tiene al sicuro il server: deve restare sempre **almeno un
amministratore attivo**, e l'ultimo non può essere disattivato o declassato.

![La pagina Utenti](/screenshots/it/manual/admin-users.png)

*Admin → Utenti — ogni account con ruolo e stato, e le azioni approva / disattiva / ruolo / reimposta.*

## Impostazioni server — Admin → Impostazioni server

Preferenze dell'intero server, applicate subito. Oggi la decisione che pesa
di più è **Approvazione automatica delle nuove registrazioni**: se attiva, i
nuovi account possono accedere nell'istante in cui si registrano; se
disattiva, un amministratore deve approvarli dalla pagina Utenti. Su
un'istanza condivisa con la famiglia conviene approvare ogni nuovo account
uno alla volta: lasciala disattiva e tieni d'occhio il badge "In approvazione".

![La pagina Impostazioni server](/screenshots/it/manual/admin-settings.png)

*Admin → Impostazioni server — l'interruttore di approvazione automatica delle nuove registrazioni.*

---

L'aspetto più pesante di quest'area — il **Backup del server** a livello di
database — è trattato in [Backup e ripristino](/it/manual/backup): ricorda
che il suo dump contiene **tutti gli account** dell'istanza. Il tuo account
personale (nome, valuta base, tema, password) resta sul lato
[Impostazioni](/it/manual/settings).
