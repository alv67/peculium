# Release Notes

## v0.6.1 — 23 Set 2026

### Nuove funzionalità
- Lo stato di aggiornamento dei prezzi ora vive nell'header dell'app, su ogni pagina: il timbro "Prezzi alle HH:MM" è sempre visibile e cliccabile per aggiornare le quotazioni su richiesta, e gli avvisi sulla qualità dei dati (rate-limit o aggiornamenti falliti, posizioni senza tasso di cambio) ti seguono in una fascia sotto l'header invece di stare nella dashboard

### Correzioni
- L'intera interfaccia ora segue la lingua scelta: dashboard, dettaglio portafoglio (Panoramica/Posizioni/Attività e le relative tabelle), liste portafogli e asset, tab del dettaglio asset, tutte le modali (crea portafoglio/asset, importa portafoglio, aggiungi transazione, modifica esposizione), Impostazioni, la pagina Stato sincronizzazione prezzi e le etichette di tipo/classe/fonte prezzo passano tra italiano e inglese
- Sul telefono il menu ⋯ dell'header dell'asset non c'è più: le sue azioni (aggiorna da Yahoo, recupera lo storico prezzi, elimina) erano già ripetute nella zona pericolosa del tab Dati, e il menu finiva fuori schermo quando l'header andava a capo
- Sul telefono il menu ⋯ dell'header del portafoglio resta allineato a destra quando l'header va a capo, e il suo pannello non esce più dallo schermo
- Sul telefono il drill-down delle allocazioni non si chiude più appena si apre: la sheet dal basso resta visibile e puoi leggere gli asset che compongono la fetta
- Il pannello di ricerca (⌘K/Ctrl+K, o il pulsante di ricerca nell'header) non si blocca più dopo il primo uso: su un vault senza portafogli o asset ora si apre ogni volta invece di funzionare una sola volta
- Sui dispositivi touch il tooltip dei grafici di allocazione non resta più bloccato sullo schermo: scompare quando apri il pannello di drill-down o tocchi fuori dal grafico

## v0.6.0 — 21 Set 2026

### Nuove funzionalità
- Clicca una fetta o una barra delle allocazioni — la ciambella delle classi di attività o le barre di settori, regioni e paesi, in dashboard e nel tab Allocazione di un portafoglio — per aprire un pannello laterale (una sheet dal basso sul telefono) con gli asset che la compongono, il loro valore, contributo e quota di quella fetta
- Ogni grafico ora si può passare alla vista tabella degli stessi dati con il nuovo controllo Grafico/Tabella sulla card: i numeri che prima si leggevano solo passando sul grafico, ora riga per riga, su qualsiasi dispositivo e anche con gli screen reader
- Apri il nuovo pannello comandi con ⌘K/Ctrl+K (o con il pulsante di ricerca nell'header, su ogni dispositivo): vai a una qualsiasi pagina, portafoglio o asset, cerca su Yahoo ed esegui le azioni rapide — aggiungi transazione, aggiorna prezzi, cambia tema, attiva la palette per daltonici — senza lasciare la pagina in cui sei
- Palette utile/perdita opzionale per daltonici: da Impostazioni → Preferenze puoi sostituire i colori verde/rosso con una coppia blu/arancione in tutta l'app, testi e grafici inclusi — i segni +/− e le frecce ▲▼ ci sono sempre, e il tema scelto non cambia
- La scheda Attività del portafoglio ora si può filtrare per tipo di transazione, asset e intervallo di date; i filtri restano nel link, quindi puoi condividere o salvare una vista filtrata e il pulsante indietro la riporta esattamente com'era
- L'eliminazione di una transazione ora si può annullare direttamente dal toast di conferma (5 secondi); la finestra di conferma dell'eliminazione non c'è più
- Sul telefono il form della transazione si apre come una sheet dal basso, con gli stessi campi del dialog su desktop
- La pagina dell'asset ora è divisa in schede Panoramica, Esposizione e Dati, ognuna con un link proprio da condividere o salvare; ticker, dati identificativi e ultima quotazione restano fissi in alto mentre cambi scheda
- La Panoramica dell'asset ora mostra "Dove è detenuto": quali dei tuoi portafogli possiedono l'asset, con quantità, costo, valore e guadagno/perdita — ogni portafoglio rimanda direttamente alla sua pagina
- Le azioni sull'asset (aggiorna da Yahoo, recupera lo storico prezzi completo, elimina) ora si trovano nel menu ⋯ dell'header dell'asset, e sono ripetute nella zona pericolosa del tab Dati
- La pagina del portafoglio ora è divisa in schede Panoramica, Posizioni, Attività e Allocazione, ognuna con un link proprio da condividere o salvare; una fascia compatta con valore e utile resta fissa in alto mentre cambi scheda
- Le azioni sul portafoglio (esporta, importa, elimina) ora si trovano nel menu ⋯ dell'header del portafoglio, accanto al pulsante "Aggiungi transazione" sempre visibile
- Ogni card dei portafogli nella dashboard ora mostra un piccolo grafico dell'andamento del valore accanto ai numeri attuali
- La dashboard ora si apre con il tuo valore netto come numero principale, un grafico valore contro investito con intervalli 1Y/3Y/TUTTO, un dettaglio attivo/chiuso apribile su richiesta, l'orario di aggiornamento dei prezzi e una nota sulla qualità dei dati quando qualcosa richiede attenzione (ad esempio posizioni escluse per il cambio mancante)
- Su un vault nuovo la dashboard mostra una checklist guidata di primo avvio: crea un portafoglio, aggiungi un asset, registra una transazione — ogni passo rimanda direttamente alla pagina giusta
- Dalla header della dashboard puoi ora passare tra l'intero vault e un singolo portafoglio
- Nuova navigazione adattiva: sul telefono una barra in basso (Panoramica, Portafogli, Asset, Altro) più un pulsante ⊕ con le azioni rapide (aggiungi transazione, aggiungi asset, aggiorna prezzi), sul tablet una sottile rail di icone e sul desktop la familiare barra laterale espandibile — con un header compatto che si restringe durante lo scroll
- La pagina di stato della sincronizzazione prezzi ora si chiama "Dati e sincronizzazione"
- L'app ora segue di default il tema del sistema: chiaro e scuro sono entrambi di prima classe, e puoi comunque fissare il tuo preferito dall'header
- Nuovo font per l'interfaccia (Inter) e un font monospazio dedicato per ticker e codici — entrambi incorporati nell'app, nulla viene scaricato da servizi di terzi
- Nuova pagina Impostazioni → Preferenze: scegli il tema (Chiaro/Scuro/Sistema) e la lingua dell'interfaccia (Italiano/Inglese); l'app ora usa di default il tema del sistema e la lingua italiana, e ricorda le tue scelte su questo dispositivo

### Correzioni
- Il recupero dell'esposizione di un asset da Morningstar ora funziona anche per i fondi quotati su più mercati: se la quotazione di un mercato non ha dati, viene usata automaticamente la quotazione successiva dello stesso ISIN

## v0.5.0 — 17 Set 2026

### Nuove funzionalità
- L'elenco delle transazioni del portafoglio ora è paginato: viene restituito pagina per pagina (di default 20 alla volta, fino a 100) in un ordine stabile dalla più recente, insieme al conteggio totale, così le storici lunghi restano rapidi e prevedibili
- Il dettaglio del portafoglio ha ora un proprio grafico di performance percentuale mensile/annuale: lo stesso time-weighted return della dashboard, misurato per il singolo portafoglio nella sua valuta, accanto alla serie del capitale investito rispetto al valore attuale
- La sezione allocazione del dettaglio portafoglio ora replica la card "Allocazione complessiva" della dashboard: una ciambella per classe di attività e barre decrescenti per regioni, settori e paesi, tutto nella valuta del portafoglio (solo azioni, ordinate per valore)
- Anche nel dettaglio del portafoglio compare lo stesso riepilogo degli investimenti attivi e chiusi della dashboard, nella valuta del portafoglio: le posizioni aperte con investito, valore attuale, guadagno/perdita e dividendi, e la parte chiusa con il costo dei lotti venduti, l'incasso e il guadagno/perdita realizzata
- La dashboard mostra ora un'unica tabella degli asset investiti: una riga per ciascun asset, aggregata su tutti i tuoi portafogli ed espressa nella tua valuta base, con importo investito, valore attuale e P/L %, ordinata per valore
- Scegli la tua valuta base in Impostazioni → Profilo: la dashboard ora consolida totali e allocazioni in quella valuta
- L'allocazione della dashboard ora mostra anche la ripartizione per classe di investimento e per singolo paese, accanto a quelle già disponibili per settore e macro-regione: i valori sono aggregati su tutti i tuoi portafogli ed espressi nella tua valuta base
- La dashboard ora separa investimenti attivi e chiusi, a livello di vault e per singolo portafoglio: per la parte chiusa vedi il costo dei lotti venduti, l'incasso e il guadagno/perdita realizzata sul capitale; le righe dei dividendi compaiono insieme agli investimenti attivi e, una volta chiusa completamente una posizione, confluiscono nell'incasso
- Nuovo grafico delle performance aggregato nella dashboard: un unico grafico per tutti i tuoi portafogli, nella tua valuta base, che mostra il rendimento percentuale time-weighted di ogni mese o anno (con una linea del TWR cumulativo) accanto a un grafico del capitale che confronta il denaro investito con il valore attuale, con selettore mensile/annuale. Il rendimento è misurato giorno per giorno, quindi versamenti, vendite, dividendi e commissioni non lo distorcono più e chiudere o riaprire completamente una posizione non produce più picchi assurdi; le posizioni senza prezzo di mercato sono mantenute al costo e non compaiono mai come falsa perdita

### Correzioni
- L'importazione di un portafoglio esportato da una versione precedente dell'app non fallisce più: le informazioni mancanti vengono completate con valori predefiniti sensati
- I portafogli con sole posizioni chiuse non mostrano più un guadagno/perdita fuorviante del -100%: le posizioni chiuse non entrano nei valori attivi e gli importi non hanno più residui di arrotondamento

## v0.4.0 — 13 Set 2026

### Nuove funzionalità
- Nuovo tema scuro, attivo di default, con le opzioni Chiaro / Scuro / Sistema
- Navigazione ridisegnata: sidebar collassabile, header in alto con selettore del tema e menu utente, e drawer a scomparsa su mobile
- Colori coerenti con il tema su tutte le pagine e i grafici, così l'interfaccia è leggibile sia in chiaro sia in scuro
- Le azioni distruttive ora usano una finestra di conferma dell'app invece del prompt nativo del browser
- Le notifiche (toast) sono state ridisegnate in linea col tema e rese accessibili agli screen reader
- La creazione di asset/portafogli e l'import di un portafoglio ora avvengono in finestre modali coerenti con il design dell'app
- Ridisegnata la schermata di accesso/registrazione con il logo Peculium, lo switch Sign in / Register, la validazione inline dei campi e la conferma password in registrazione
- Dashboard ridisegnata: card KPI per valuta, donut dell'allocazione e card dei portafogli cliccabili
- L'header della dashboard mostra quando i prezzi sono stati aggiornati l'ultima volta
- Dettaglio portafoglio: aggiungere/modificare una transazione ora avviene in una finestra con validazione inline e totale live, e l'eliminazione è confermata nell'app
- Le posizioni e le transazioni del portafoglio usano le tabelle condivise del design system (le posizioni ora mostrano anche l'ultimo prezzo), e le azioni della pagina stanno in un header sticky
- Impostazioni riorganizzate in tab: Profile, Password, Currencies e Health
- Le valute gestite ora si scelgono da una lista, con il nome compilato automaticamente
- Il cambio password ora valida inline ed evidenzia il campo in errore (es. password corrente sbagliata)
- La lista degli eventi di health della sincronizzazione prezzi ora è paginata, così si può consultare tutto lo storico
- La pagina di health della sincronizzazione prezzi è stata spostata in una sezione Admin dedicata nella sidebar

### Correzioni
- Le allocazioni del portafoglio si aggiornano subito dopo aver aggiunto, modificato o eliminato una transazione, senza ricaricare la pagina
- Il grafico storico dei portafogli in dashboard disegna ogni portafoglio come una linea continua su una timeline reale, e si può zoomare e spostare come i grafici del portafoglio
- Health della sincronizzazione prezzi: le card Success Rate e Rate Limited mostrano ora i valori reali (il rate poteva restare bloccato su `N/A`, o mostrare `NaN%` senza dati)
- Health della sincronizzazione prezzi: i messaggi di errore ora indicano il tipo di richiesta (chart / spark / search / fx) e il relativo ticker o valuta
- Gli asset non gestiti da Yahoo (manual / none) non generano più errori di sincronizzazione nella dashboard di health
- Health della sincronizzazione prezzi: i totali non si azzerano più al riavvio e si possono limitare a Today / Last 24h / Last 100 eventi

## v0.3.0 — 11 Set 2026

### Nuove funzionalità
- Scegli come ogni asset riceve i suoi prezzi: `Yahoo`, `Manual` o `None` (evita errori Yahoo per ticker non-Yahoo come alcuni bond)
- Il grafico storico dell'asset ora carica tutto lo storico e fa lo zoom in-place usando i selettori 1M/3M/1Y/YTD/MAX (nessun ricaricamento inutile)
- Nuovo range `YTD` (da inizio anno) sul grafico storico dell'asset
- Marcatori degli split mostrati sul grafico storico dell'asset (es. `Split 4:1`)
- L'esposizione (regioni/settori) dell'asset si modifica in una modale dedicata, con tabelle dei pesi validate e compilazione automatica da JustETF e Yahoo
- Modifica la distribuzione geografica con un elenco per-paese (aggiungi/rimuovi paesi e imposta ogni peso), oltre a regioni e settori
- L'esposizione geografica e settoriale può essere compilata anche da Morningstar (regioni ufficiali), oltre a JustETF e Yahoo
- Ogni distribuzione mostra da dove arrivano i dati e quando sono stati aggiornati l'ultima volta (es. `da Morningstar (2026-09-05)`)
- Le letture dai provider sono in cache, quindi riaprire i prefill è immediato

### Correzioni
- Riaprire un editor di esposizione ora riparte sempre dai dati salvati: le modifiche non salvate vengono scartate

## v0.2.0 — 30 Ago 2026

### Nuove funzionalità
- Valori coerenti nel riepilogo del portafoglio anche quando manca un tasso di cambio
- Chart di distribuzione geografica e settoriale per portafogli e dashboard
- Classi di asset e allocazione per classe di investimento
- Storico dei tassi di cambio, così serie e chart restano corretti nel tempo
- Pagina dettaglio asset con riferimenti, esposizione e storico prezzi completo
- Esposizione ETF automatica (paesi/regioni e settori) con ricerca del codice ISIN partendo dal ticker

## v0.1.0 — 25 Ago 2026 (prima release ufficiale)

### Nuove funzionalità
- Registrazione e login multi-utente (JWT con access e refresh token in rotazione)
- Impostazioni account: modifica di nome ed email e cambio password
- Portafogli: creazione, modifica, eliminazione, esportazione e importazione
- Asset: creazione, modifica, eliminazione con autocomplete del ticker e sincronizzazione automatica dei prezzi Yahoo
- Transazioni (acquisto / vendita / dividendo / split / commissione)
- Dashboard: valore totale, gain/loss, allocazione, performance e ROI per asset
- Grafico storico del valore del portafoglio nella dashboard, per vedere come cambia nel tempo
- Grafico storico di performance del singolo portafoglio con carico investito (cost basis), valore degli asset ancora investiti e andamento storico del P/L realizzato
- Lista portafogli con il valore corrente di ogni portafoglio/asset
- Supporto multi-valuta (EUR / USD / GBP / CHF) con whitelist delle valute configurabile
- Importo investito per valuta nella dashboard
- Aggiornamento prezzi automatico e manuale con dashboard di health della sincronizzazione
- Aggiornamento automatico periodico di prezzi asset e tassi di cambio, che aggiorna valori e storico fino all'ultima esecuzione
