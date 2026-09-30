---
title: Backup e ripristino
order: 7
description: Due reti di sicurezza con ambiti diversi — i tuoi dati e l'intero server.
---

Peculium offre il backup su due livelli, e conviene non confonderli: uno
protegge il **tuo spazio di lavoro**, l'altro protegge **l'intera istanza**.
Come regola pratica, fai un backup fresco prima di ogni aggiornamento della
stack (la [guida all'installazione](/it/install) copre l'aggiornamento in sé).

## I tuoi dati — Impostazioni → Backup e ripristino

Chiunque può salvare ciò che possiede. La card **Scarica i tuoi dati** esporta
ogni portafoglio con le sue transazioni, gli asset correlati (metadati,
esposizione e i [prezzi manuali](/it/manual/daily-use) inseriti) e le valute
in uso, in un unico file JSON. I dati dei provider — prezzi Yahoo, storico
cambi — sono esclusi di proposito: vengono recuperati di nuovo dopo il
ripristino, come descritto in [Dati e sincronizzazione](/it/manual/data-sync).

**Ripristina da un backup** accetta uno di questi file e offre due modalità:

- **Aggiungi ai dati attuali** — non distruttiva: i portafogli del backup
  vengono importati come nuovi e nulla di ciò che già hai viene toccato.
- **Sostituisci i dati attuali** — prima elimina i tuoi portafogli esistenti e
  le relative transazioni, poi importa il file; l'operazione va confermata in
  una finestra di dialogo e non è reversibile. In nessuna delle due modalità
  gli asset vengono eliminati: il catalogo è condiviso, le voci esistenti
  vengono riusate e solo quelle mancanti ricreate ([Concetti](/it/manual/concepts)).

Un ripristino riuscito riporta i conteggi: portafogli e transazioni create,
asset creati e riusati.

![Impostazioni → Backup e ripristino](/screenshots/it/manual/backup-user.png)

*Impostazioni → Backup e ripristino — un download JSON, ripristino in modalità Aggiungi o Sostituisci.*

## L'intero server — Admin → Backup del server (solo admin)

Gli amministratori hanno lo strumento pesante sotto **Admin → Backup del
server**. **Scarica un dump completo** trasferisce l'intero database del
server — ogni utente, portafoglio, asset e impostazione — come archivio
PostgreSQL in formato custom. Custodisci quel file con cura: a differenza del
backup personale, contiene *tutti gli account* dell'istanza.

**Ripristina da un dump** ne è l'immagine speculare, ed è
**distruttivo**: riscrive ogni tabella a partire dall'archivio, utenti
inclusi. Qui non esiste una modalità "aggiungi" — sostituisce il database
dell'intero server, non solo il tuo account — e bisogna scrivere la parola
**SOSTITUISCI** per confermare. Le sessioni e i token correnti potrebbero
smettere di funzionare, quindi mettiti in conto di **effettuare di nuovo il
login**. È un'operazione lato server: i database grandi richiedono tempo —
tieni aperta la pagina fino al risultato.

![Admin → Backup del server](/screenshots/it/manual/backup-server.png)

*Admin → Backup del server — il dump completo del database e il ripristino distruttivo, bloccato da una conferma scritta.*

---

Le card a livello utente fanno parte delle [Impostazioni](/it/manual/settings);
l'area di amministrazione attorno al backup del server — utenti, impostazioni
server, salute della sincronizzazione — è trattata in
[Amministrazione](/it/manual/administration).
