import type { Dictionary } from './en'

/**
 * Italian dictionary — the product default language (decision D1). Kept
 * structurally identical to `en.ts` (canonical): `satisfies Dictionary`
 * fails the build on any missing, extra or misspelled key.
 *
 * Values are shown when the active locale is `it`; `{placeholders}` stay
 * ASCII so interpolated labels compose inside aria-labels.
 */
export const it = {
  common: {
    language: 'Lingua',
    /** Etichette generiche dei dialoghi di conferma (es. eliminazione portafoglio). */
    delete: 'Elimina',
    cancel: 'Annulla',
    /** Fallback affermativo del ConfirmDialog (in coppia con `cancel`). */
    confirm: 'Conferma',
    /** Chiusura di sheet/dialoghi (es. sheet transazione, K.4c). */
    close: 'Chiudi',
    /** Riga con il numero di asset detenuti (card portfolio in dashboard, dettaglio portafoglio). */
    assetCount: '{count} asset',
    /** Intestazioni di colonna condivise dalle tabelle posizioni e transazioni. */
    colAsset: 'Asset',
    colDate: 'Data',
    colType: 'Tipo',
    colQty: 'Qtà',
    colPrice: 'Prezzo',
    colValue: 'Valore',
    colTotal: 'Totale',
    colActions: 'Azioni',
    /** Etichetta Code condivisa tra form/tabella valute e tabella eventi health. */
    colCode: 'Codice',
    /** Pulsanti del piè di pagina di paginazione della tabella transazioni. */
    previous: 'Precedente',
    next: 'Successivo',
    /** Contatore di righe accanto ai pulsanti di paginazione ("1–20 di 137",
     *  variante vuota quando la finestra non ha righe). */
    rangeLabel: '{from}–{to} di {total}',
    rangeEmpty: '0 di {total}',
    /** Riga di avanzamento mentre una lista/tabella è in caricamento. */
    loading: 'Caricamento…',
    /** Titolo di stato vuoto predefinito della card asincrona generica. */
    nothingHere: "Non c'è ancora nulla",
    /** Azione di riprova degli stati di errore (superficie d'errore di AsyncCard). */
    retry: 'Riprova',
    /** Etichette generiche dei pulsanti dei form delle modali di creazione/modifica. */
    create: 'Crea',
    save: 'Salva',
    saving: 'Salvataggio…',
    saveChanges: 'Salva modifiche',
    /** Suggerimento sotto l'etichetta di un campo facoltativo. */
    optional: 'facoltativo',
    /** Fallback di errore generici quando l'API non porta un messaggio. */
    deleteFailed: 'Eliminazione non riuscita',
    saveFailed: 'Salvataggio non riuscito',
    createFailed: 'Creazione non riuscita',
    somethingWentWrong: 'Si è verificato un errore',
  },
  /**
   * Nomi accessibili incorporati nelle primitive generiche (spinner, modale,
   * toast, drawer/sheet, tab). Copy solo per screen reader, mai visibile.
   */
  a11y: {
    loading: 'Caricamento',
    closeDialog: 'Chiudi finestra di dialogo',
    dismissNotification: 'Chiudi notifica',
    /** Nome predefinito della ✕ della coppia drawer/sheet (override possibile). */
    closePanel: 'Chiudi pannello',
    /** Fallback del drawer/sheet senza titolo. */
    details: 'Dettagli',
    /** Fallback della striscia Tabs quando il chiamante non passa un nome accessibile. */
    sections: 'Sezioni',
  },
  nav: {
    main: 'Principale',
    dashboard: 'Dashboard',
    overview: 'Panoramica',
    portfolios: 'Portafogli',
    assets: 'Asset',
    more: 'Altro',
    bottomNav: 'Navigazione principale',
    dataSync: 'Dati e sincronizzazione',
    /** Voci dell'area Admin (#57 Fase A): renderizzate solo per amministratori. */
    users: 'Utenti',
    serverSettings: 'Impostazioni server',
    settings: 'Impostazioni',
    sectionAdmin: 'Admin',
    sectionSettings: 'Impostazioni',
    skipToContent: 'Vai al contenuto',
  },
  header: {
    expandSidebar: 'Espandi la barra laterale',
    collapseSidebar: 'Comprimi la barra laterale',
  },
  fab: {
    open: 'Apri le azioni rapide',
  },
  quickActions: {
    title: 'Azioni rapide',
    addTransaction: 'Aggiungi transazione',
    addTransactionHint: 'Registra un acquisto, una vendita o un dividendo',
    addAsset: 'Aggiungi asset',
    addAssetHint: 'Cerca su Yahoo e registralo',
    refreshPrices: 'Aggiorna prezzi',
    refreshPricesHint: 'Recupera subito le ultime quotazioni',
    enterPrice: 'Inserisci prezzo',
    comingSoon: 'In arrivo',
    refreshSuccess: 'Prezzi aggiornati',
    refreshError: 'Aggiornamento dei prezzi non riuscito',
    refreshRateLimited: 'Yahoo Finance sta limitando le richieste: alcuni prezzi potrebbero non essere aggiornati',
    refreshIssues: '{count} aggiornamenti prezzi non riusciti (Yahoo)',
  },
  user: {
    accountMenu: 'Menu account',
    signOut: 'Esci',
    fallbackName: 'Utente',
  },
  /**
   * Schermata di login / registrazione (pagina unica, toggle di modalità).
   * Il wordmark "Peculium" è un nome proprio e resta invariato; tutto il
   * resto — etichette di modalità, sottotitoli, label/hint dei campi,
   * validazioni, toast — è localizzato qui.
   */
  login: {
    tagline: 'Il tuo patrimonio, a casa tua',
    /** Opzioni del SegmentedControl di modalità, riusate tal quali dalla CTA. */
    signIn: 'Accedi',
    register: 'Registrati',
    /** Sottotitolo che cambia con la modalità, sotto il blocco brand. */
    createAccount: 'Crea un account',
    signInToAccount: 'Accedi al tuo account',
    /** Nome accessibile del SegmentedControl di modalità. */
    authMode: 'Modalità di autenticazione',
    /** Etichette dei campi. */
    name: 'Nome',
    email: 'Email',
    password: 'Password',
    confirmPassword: 'Conferma password',
    /** Hint in modalità registrazione, sotto il campo password. */
    passwordHint: 'Almeno 8 caratteri',
    /** Messaggi di validazione inline. */
    emailRequired: "L'email è obbligatoria",
    emailInvalid: 'Inserisci un indirizzo email valido',
    passwordRequired: 'La password è obbligatoria',
    passwordTooShort: 'La password deve essere lunga almeno 8 caratteri',
    passwordMismatch: 'Le password non coincidono',
    nameRequired: 'Il nome è obbligatorio',
    /** Toast di esito della registrazione (poi il form torna al login). */
    registered: 'Registrazione completata! Ora puoi accedere.',
    /** Toast di registrazione quando serve l'approvazione (auto-approvazione
     *  off, #57 Fase A). */
    registeredPending:
      'Registrazione completata! Un amministratore deve approvare il tuo account prima di accedere.',
    /** Accesso negato su account non attivo (403 del backend, #57 Fase A). */
    accountPending:
      'Il tuo account è in attesa di approvazione. Un amministratore deve approvarlo prima che tu possa accedere.',
    accountDisabled: 'Il tuo account è stato disattivato. Contatta un amministratore per riattivarlo.',
  },
  theme: {
    group: 'Tema',
    light: 'Chiaro',
    dark: 'Scuro',
    system: 'Sistema',
    aria: 'Tema: {theme}',
    ariaSystem: 'Tema: {theme}, attualmente {resolved}',
  },
  settingsTabs: {
    sections: 'Sezioni delle impostazioni',
    profile: 'Profilo',
    password: 'Password',
    preferences: 'Preferenze',
    currencies: 'Valute',
  },
  /**
   * Pannello comandi globale (EPIC K.5a, spec §8.1). Le etichette di
   * destinazioni/azioni sono riusate dai gruppi `nav.*`, `settingsTabs.*`,
   * `quickActions.*`, `theme.*` e `preferences.palette*`; qui c'è solo la
   * copy propria del pannello. `{query}` porta il testo digitato.
   */
  commandPalette: {
    /** Nome accessibile del dialog. */
    title: 'Pannello comandi',
    /** aria-label del pulsante di ricerca nell'header. */
    trigger: 'Apri il pannello comandi',
    /** Etichetta visibile (lg+) e aria-label dell'input di ricerca. */
    inputLabel: 'Cerca',
    placeholder: 'Cerca pagine, portafogli, asset…',
    sectionGoTo: 'Vai a',
    sectionAssets: 'Asset',
    sectionActions: 'Azioni',
    noResults: 'Nessun risultato',
    yahooRow: 'Cerca su Yahoo “{query}”',
    searching: 'Ricerca su Yahoo in corso…',
    /** Azioni (gli hint mostrano lo stato di destinazione dei toggle). */
    toggleTheme: 'Cambia tema',
    toggleCvd: 'Attiva/disattiva palette per daltonici',
    toggleSidebar: 'Mostra/nascondi barra laterale',
    /** Suggestioni da tastiera, dopo un glifo <kbd>. */
    hintNavigate: 'per navigare',
    hintSelect: 'per selezionare',
    hintClose: 'per chiudere',
  },
  preferences: {
    title: 'Preferenze',
    themeHint: 'Chiaro, scuro, oppure in base alle impostazioni del dispositivo (Sistema).',
    languageHint: 'Applicata subito e ricordata su questo dispositivo.',
    /** Controllo della palette utile/perdita (decisione D6, EPIC K.5c): vale
     *  anche come nome accessibile della tablist SegmentedControl. Le
     *  etichette restano corte ("Verde/Rosso" / "Blu/Arancione") per non far
     *  traboccare il controllo nella card a larghezza telefono (bug-fix
     *  EPIC K); il pannello comandi le riusa come hint dello stato di
     *  destinazione del toggle, e `paletteHint` porta la spiegazione estesa. */
    colorGroup: 'Colori utile/perdita',
    paletteClassic: 'Verde/Rosso',
    paletteCvd: 'Blu/Arancione',
    paletteHint: 'Sostituisce verde/rosso con blu/arancione in testi e grafici. Segni e frecce ▲▼ restano sempre.',
  },
  /** Card Profilo delle Impostazioni (riusa `nav.settings`,
   *  `settingsTabs.profile`, `chartView.colName`, `common.save/saving`). */
  profile: {
    email: 'Email',
    baseCurrency: 'Valuta base',
    baseCurrencyHint: 'Usata per consolidare i valori dei portafogli nella dashboard.',
    updated: 'Profilo aggiornato',
    updateFailed: 'Aggiornamento non riuscito',
  },
  /** Card Password delle Impostazioni: etichette, validazioni e toast. */
  password: {
    change: 'Cambia password',
    current: 'Password attuale',
    new: 'Nuova password',
    confirm: 'Conferma nuova password',
    minLengthHint: 'Almeno 8 caratteri',
    currentRequired: 'La password attuale è obbligatoria',
    tooShort: 'La password deve essere lunga almeno 8 caratteri',
    mismatch: 'Le password non coincidono',
    currentIncorrect: 'La password attuale non è corretta',
    changed: 'Password modificata',
    changeFailed: 'Modifica non riuscita',
  },
  /** Card Valute delle Impostazioni (riusa `common.colCode` per Code,
   *  `chartView.colName`/`common.colActions` per la tabella, `common.*` per
   *  caricamento e dialoghi; `{code}` porta il codice di valuta grezzo). */
  currencies: {
    title: 'Valute gestite',
    select: 'Seleziona una valuta',
    namePlaceholder: 'Facoltativo',
    allManaged: 'Tutte le valute disponibili sono già gestite.',
    empty: 'Nessuna valuta trovata.',
    add: 'Aggiungi',
    adding: 'Aggiunta…',
    added: 'Valuta {code} aggiunta',
    conversionUnavailable: 'Conversione USD->{code} non disponibile: valuta non gestibile',
    alreadyPresent: 'Valuta già presente',
    addFailed: 'Aggiunta della valuta non riuscita',
    removed: 'Valuta {code} rimossa',
    remove: 'Rimuovi valuta',
    removeNamed: 'Rimuovi la valuta {code}',
    removeFailed: 'Rimozione della valuta non riuscita',
    inUse: 'Valuta in uso o protetta',
    deleteTitle: 'Elimina valuta',
    deleteConfirm: 'Eliminare la valuta {code}?',
    loadFailed: 'Caricamento delle valute non riuscito',
  },
  /**
   * Superfici di allocazione (bug-fix EPIC K, sweep progressivo D1): la card
   * "Allocazione complessiva" del patrimonio in dashboard, il tab Allocazione del
   * portafoglio e il suo digest in Panoramica — titoli dei pannelli, stati di
   * errore isolati e nota di copertura dell'universo azionario (`{pct}` porta
   * una cifra decimale).
   */
  allocation: {
    title: 'Allocazione complessiva',
    unavailable: 'Allocazione non disponibile',
    classUnavailable: 'Allocazione per classi non disponibile',
    sectorUnavailable: 'Allocazione settoriale non disponibile',
    geoUnavailable: 'Allocazione geografica non disponibile',
    assetClasses: 'Classi di attività',
    sectorsEquity: 'Settori (solo azionario)',
    regionsEquity: 'Regioni (solo azionario)',
    countriesEquity: 'Paesi (solo azionario)',
    /** Nota di copertura dell'universo azionario mostrata quando c'era esclusioni. */
    equityUniverse: 'Universo azionario: {pct}% del portafoglio',
    /** Torta dashboard: valore del patrimonio ripartito per portafoglio. */
    byPortfolio: 'Allocazione per portafoglio',
    mixedCurrencies:
      'I portafogli usano valute diverse: i valori non sono confrontabili, le quote sono indicative.',
  },
  /**
   * Drill-down delle allocazioni (EPIC K.5, spec §6.5): il drawer/sheet
   * aperto al click su una fetta o una barra dei grafici di allocazione, che
   * elenca gli asset contribuenti del bucket. Il titolo visibile è l'etichetta
   * leggibile del bucket passata dal grafico chiamante; le intestazioni
   * Valore/Peso riusano `chartView.colValue`/`chartView.colWeight` e il nome
   * accessibile della ✕ riusa `common.close`.
   */
  drill: {
    /** `<caption>` sr-only della tabella asset contribuenti ({name} = etichetta del bucket). */
    caption: '{name} — asset contribuenti',
    /** Sottotitolo muted sopra la tabella; `{count}` = numero di righe. */
    contributingAssets: '{count} asset contribuiscono',
    colAsset: 'Asset',
    /** Quota dell'asset dentro il bucket (valore × peso di esposizione). */
    colContribution: 'Contributo',
    /** Contributo ÷ totale del bucket: la quota dell'asset nella fetta. */
    colShare: 'Quota della fetta',
    empty: 'Nessun asset in questa fetta',
    error: 'Impossibile caricare gli asset di questa fetta',
    retry: 'Riprova',
  },
  /**
   * Tab Esposizione degli asset e relative modali di modifica (bug-fix EPIC
   * K): titoli delle card, intestazioni dei pannelli, pulsanti di modifica e
   * banner riservato all'equity.
   */
  exposure: {
    geoTitle: 'Distribuzione geografica',
    sectorTitle: 'Distribuzione settoriale',
    countries: 'Paesi',
    regions: 'Regioni',
    sectors: 'Settori',
    noCountries: 'Nessun paese inserito',
    noCountriesHint: 'Aggiungi un paese qui sotto, oppure usa un prefill JustETF / Morningstar',
    modify: 'Modifica',
    editGeo: 'Modifica distribuzione geografica',
    editSectors: 'Modifica distribuzione settoriale',
    /** Banner che sostituisce le card per asset fuori dall'universo azionario. */
    universeTitle: 'Distribuzione geografica e settoriale',
    universeHint:
      'Questa distribuzione si applica solo agli asset azionari (azioni ed ETF/fondi di classe equity).',
    universeClassHint: "Imposta la classe 'Azioni' o 'Immobiliare' nelle Caratteristiche per attivarla.",
    /** Chrome delle modali di modifica: intestazioni di colonna, nomi
     *  accessibili delle righe e controllo di aggiunta (il piè di pagina
     *  Salva riusa `common.save`/`common.saving`/`common.colTotal`). */
    colCountry: 'Paese',
    colWeightPct: 'Peso %',
    geoArea: 'Area geografica',
    gicsSector: 'Settore GICS',
    weightAria: 'Peso di {name}',
    removeAria: 'Rimuovi {name}',
    addCountryAria: 'Paese da aggiungere',
    add: 'Aggiungi',
    /** Pulsanti di prefill dei provider: tooltip corti più aria specifica. */
    prefillJustEtf: 'Prefill da JustETF',
    prefillYahoo: 'Prefill da Yahoo',
    prefillMorningstar: 'Prefill da Morningstar',
    prefillCountriesJustEtf: 'Prefill paesi da JustETF',
    prefillCountriesMorningstar: 'Prefill paesi da Morningstar',
    prefillRegionsMorningstar: 'Prefill regioni da Morningstar',
    prefillSectorsJustEtf: 'Prefill settori da JustETF',
    prefillSectorsYahoo: 'Prefill settori da Yahoo',
    prefillSectorsMorningstar: 'Prefill settori da Morningstar',
    deriveTitle: 'Calcola da paesi',
    deriveAria: 'Calcola regioni dai paesi',
    /** Messaggi di validazione della somma dei pesi nei piè di pagina
     *  ({pct} porta la somma già formattata con due decimali). */
    overSum: 'La somma supera il 100% — attuale {pct}%. Riduci i pesi per salvare.',
    over100Title: 'La somma dei pesi supera il 100%: riduci i pesi per poter salvare',
    residualCountries: 'Residuo non attribuito: {pct}%.',
    residualRegions: 'Residuo non classificato: {pct}% — escluso dal grafico.',
    sumMustBe100: 'La somma dei pesi deve essere 100 (±0.5) — attuale: {pct}%',
  },
  /**
   * Badge di provenienza (bug-fix EPIC K): etichetta della pill, spiegazione
   * nel tooltip/aria connessa e connettore di data (`{date}` è il stamp
   * YYYY-MM-DD grezzo). Una chiave piatta per id di provenienza
   * (`manualLabel`/`manualDesc`, …) perché il runtime supporta esattamente
   * due livelli di annidamento.
   */
  provenance: {
    updatedTo: 'aggiornato al {date}',
    manualLabel: 'manuale',
    manualDesc: 'Dati modificati manualmente',
    justetfLabel: 'da JustETF',
    justetfDesc: 'Lista paesi importata da JustETF, non modificata manualmente',
    morningstarLabel: 'da Morningstar',
    morningstarDesc: 'Dati importati da Morningstar, non modificati manualmente',
    morningstarRegionsLabel: 'da Morningstar (regioni ufficiali)',
    morningstarRegionsDesc: 'Regioni ufficiali importate da Morningstar, non modificate manualmente',
    yahooLabel: 'da Yahoo',
    yahooDesc: 'Settori importati da Yahoo, non modificati manualmente',
    derivedLabel: 'calcolato dai paesi',
    derivedDesc: 'Regioni calcolate a partire dai pesi dei paesi',
    derivedEtfLabel: 'da JustETF via paesi',
    derivedEtfDesc: 'Regioni calcolate dai paesi importati da JustETF',
  },
  hero: {
    netValue: 'Valore netto',
    allTime: 'complessivo',
    valueVsInvested: 'Valore vs investito',
    breakdown: 'Dettaglio',
    realized: 'Realizzato',
    dividends: 'Dividendi',
    invested: 'Investito',
  },
  /**
   * Card Performance (condivisa tra la vista patrimonio della dashboard e la
   * Panoramica del dettaglio portafoglio): titolo della card e relativo
   * selettore di granularità Mensile/Annuale.
   */
  performance: {
    title: 'Performance',
    granularity: 'Granularità performance',
    monthly: 'Mensile',
    annual: 'Annuale',
  },
  /**
   * Copy delle zone della dashboard: titoli delle card non gestiti da altri
   * gruppi, la riga "chiuse" compatta sulle card portfolio e la tabella
   * consolidata "Asset investiti" (EPIC I.5) con intestazioni e badge
   * "senza prezzo".
   */
  dashboard: {
    closedPrefix: 'Chiuse:',
    investedAssets: 'Asset investiti',
    noInvestedAssets: 'Nessun asset investito',
    noInvestedAssetsHint: 'Le posizioni aperte compariranno qui non appena registri transazioni nei tuoi portafogli.',
    colAsset: 'Asset',
    colGainLoss: 'Guadagno/Perdita',
    colPnlPct: 'P/L %',
    noPrice: 'senza prezzo',
    noPriceHint: 'Nessun dato di prezzo: il valore è mantenuto al costo, quindi il suo P/L è 0',
  },
  /**
   * Card di riepilogo "Investimenti" (dettaglio Attive/Chiuse): condivisa
   * tra la disclosure dell'hero in dashboard e la Panoramica del dettaglio
   * portafoglio.
   */
  investments: {
    title: 'Investimenti',
    group: 'Gruppo',
    invested: 'Investito',
    valueProceeds: 'Valore / Incasso',
    gainLoss: 'Guadagno/Perdita',
    pct: '%',
    dividends: 'Dividendi',
    active: 'Attive',
    closed: 'Chiuse',
  },
  period: {
    oneYear: '1Y',
    threeYears: '3Y',
    all: 'TUTTO',
    group: 'Periodo del grafico',
  },
  quality: {
    fxMissing: '{amount} esclusi — cambio mancante ({count} posizioni)',
    rateLimited: 'Alcuni prezzi non aggiornati (limitazione Yahoo)',
    refreshIssues: '{count} aggiornamenti prezzi non riusciti',
    refreshFailed: 'Aggiornamento prezzi non riuscito: i valori potrebbero essere obsoleti',
  },
  freshness: {
    asOf: 'Prezzi alle {time}',
    refreshing: 'Aggiornamento dei prezzi…',
    hint: 'I valori consolidati usano questi prezzi',
    partialHint: 'Prezzi alle {time}: alcuni aggiornamenti non riusciti o limitati',
  },
  /**
   * Pagina di stato sincronizzazione prezzi (`/admin/health`, voce di nav
   * `nav.dataSync`). I valori backend (`event_type`, `code`, `message`) sono
   * mostrati verbatim; solo il badge di stato è localizzato dalle chiavi
   * `status*` con fallback al valore grezzo. `N/A` e l'unità `ms` restano.
   */
  health: {
    title: 'Stato sincronizzazione prezzi',
    subtitle: "Monitoraggio di connettività e prestazioni dell'API Yahoo Finance",
    periodToday: 'Oggi',
    periodLast24h: 'Ultime 24h',
    periodLast100: 'Ultimi 100',
    /** Nome accessibile del SegmentedControl + didascalia "Periodo: …". */
    periodAria: 'Periodo di monitoraggio',
    periodLabel: 'Periodo: {period}',
    refreshing: 'Aggiornamento…',
    refresh: 'Aggiorna ora',
    noData: 'Nessun dato di monitoraggio disponibile.',
    /** Card delle metriche. */
    successRate: 'Tasso di successo',
    totalSuccesses: 'Successi totali',
    totalFailures: 'Fallimenti totali',
    rateLimited: 'Richieste limitate',
    /** Tabella eventi (titolo, aria-label della tabella e intestazioni;
     *  Tipo/Stato/Code riusano `common.colType`, `positions.colStatus` e
     *  `common.colCode`). */
    recentEvents: 'Eventi recenti',
    colTimestamp: 'Data e ora',
    colMessage: 'Messaggio',
    colDuration: 'Durata',
    /** Etichette del badge di stato per i valori backend noti. */
    statusSuccess: 'Successo',
    statusRateLimited: 'Richieste limitate',
    statusFailure: 'Errore',
    loadFailed: 'Caricamento dei dati di monitoraggio non riuscito',
  },
  /**
   * Area Admin (#57 Fase A): stato di negato accesso condiviso, pagina Utenti
   * (badge di ruolo/statuse, azioni di riga, dialoghi e toast) e pagina delle
   * impostazioni server. Le etichette dei badge localizzano gli enum backend;
   * i valori inattesi mostrano la stringa grezza al punto d'uso.
   */
  admin: {
    /** Stato vuoto condiviso su ogni rotta `/admin/*` protetta. */
    forbiddenTitle: 'Solo amministratori',
    forbiddenHint: 'Non hai accesso a quest\'area.',
    /** Pagina Utenti (titolo che riusa `nav.users`, intestazioni `login.email`,
     *  `chartView.colName`, `positions.colStatus`, `common.colActions`). */
    usersSubtitle: 'Approva le registrazioni, gestisci i ruoli e reimposta le password.',
    colRole: 'Ruolo',
    colCreated: 'Creato il',
    /** Etichette badge/selettore di ruolo (`owner` è legacy: mostrato, mai
     *  assegnabile). */
    roleOwner: 'Proprietario',
    roleAdmin: 'Amministratore',
    roleEditor: 'Editor',
    roleViewer: 'Visualizzatore',
    /** Etichette badge di stato. */
    statusActive: 'Attivo',
    statusPending: 'In approvazione',
    statusDisabled: 'Disattivato',
    /** Tooltip delle azioni di riga; le chiavi `*Named` sono i nomi accessibili. */
    approve: 'Approva',
    disable: 'Disattiva',
    enable: 'Riattiva',
    resetPassword: 'Reimposta password',
    approveNamed: 'Approva {email}',
    disableNamed: 'Disattiva {email}',
    enableNamed: 'Riattiva {email}',
    resetNamed: 'Reimposta la password di {email}',
    roleNamed: 'Ruolo di {email}',
    /** Nota sotto la tabella sul ruolo legacy. */
    ownerNote: 'Il ruolo legacy Proprietario equivale ad Amministratore e non può essere assegnato a nuovi account.',
    /** Toast degli esiti delle azioni di riga. */
    approved: 'Utente approvato',
    userEnabled: 'Utente riattivato',
    userDisabled: 'Utente disattivato',
    roleUpdated: 'Ruolo aggiornato',
    passwordReset: 'Password reimpostata',
    /** Fallback di errore e codici API mappati (409/404). */
    loadFailed: 'Caricamento utenti non riuscito',
    loadFailedHint: 'Verifica la connessione e riprova.',
    empty: 'Nessun utente trovato.',
    updateFailed: 'Aggiornamento non riuscito',
    resetFailed: 'Reimposta password non riuscita',
    lastAdmin: 'Il server deve mantenere almeno un amministratore attivo.',
    userNotFound: 'Utente non trovato.',
    /** Dialogo di conferma disattivazione. */
    disableTitle: 'Disattiva utente',
    disableConfirm: 'Disattivare {email}? Non potrà accedere finché non sarà riattivato.',
    /** Dialogo di reimposta password (label/hint del campo riusano `password.*`). */
    resetTitle: 'Reimposta la password di {email}',
    resetHint: "Imposta una nuova password per questo account; l'utente dovrà cambiarla dopo l'accesso.",
    /** Pagina impostazioni server (titolo che riusa `nav.serverSettings`). */
    settingsSubtitle: 'Preferenze dell\'intero server, applicate subito.',
    autoApprove: 'Approvazione automatica delle nuove registrazioni',
    autoApproveHint:
      'Se attiva, i nuovi account possono accedere subito. Se disattiva, un amministratore deve approvarli dalla pagina Utenti.',
    lastUpdated: 'Ultimo aggiornamento: {time}',
    settingSaved: 'Impostazione salvata',
    settingSaveFailed: 'Salvataggio impostazione non riuscito',
    settingsLoadFailed: 'Caricamento impostazioni server non riuscito',
  },
  checklist: {
    title: 'Configura il tuo patrimonio',
    intro: 'Tre passi per iniziare a tracciare i tuoi investimenti.',
    stepPortfolio: 'Crea un portafoglio',
    stepPortfolioHint: 'Raggruppa gli investimenti per obiettivo o conto.',
    stepAsset: 'Aggiungi un asset',
    stepAssetHint: 'Cerca su Yahoo e registra ciò che possiedi.',
    stepTransaction: 'Registra una transazione',
    stepTransactionHint: 'Apri un portafoglio e inserisci un acquisto.',
    done: 'Completato',
    current: 'Passo corrente',
    pending: 'Non iniziato',
  },
  scope: {
    label: 'Ambito',
    all: 'Tutti i portafogli (Patrimonio)',
  },
  sparkline: {
    trend: 'Andamento del valore',
    valueTrend: 'Andamento del valore di {name}',
  },
  /**
   * Toggle "vedi come tabella" dei grafici (EPIC K.5b, spec §9.1):
   * etichette e nomi accessibili del `ui/ChartTableToggle` condiviso, il
   * template del `<caption>` sr-only delle viste tabellari e le intestazioni
   * di colonna. `{name}` porta il titolo proprio del grafico, se ce l'ha.
   */
  chartView: {
    chart: 'Grafico',
    table: 'Tabella',
    aria: 'Vista dati del grafico',
    ariaNamed: 'Vista dati del grafico — {name}',
    caption: '{name} — dati del grafico',
    captionGeneric: 'Dati del grafico',
    /** Intestazioni di colonna condivise dalle viste tabellari (usate anche
     *  come etichette `Valore:`/`Peso:` nei tooltip ECharts, bug-fix EPIC K). */
    colName: 'Nome',
    colValue: 'Valore',
    colWeight: 'Peso',
    colPeriod: 'Periodo',
    colReturn: 'Rendimento',
    colCumulative: 'TWR cumulativo',
    colInvested: 'Investito',
    namePerformance: 'Performance',
    nameCapital: 'Investito contro valore',
    /** Titoli predefiniti dei wrapper donut di allocazione/esposizione quando
     *  il chiamante non passa un `title` (nome serie, toggle e didascalia). */
    nameAllocation: 'Allocazione',
    nameDistribution: 'Distribuzione',
    /** Stati vuoti dei wrapper di grafici di allocazione/esposizione. */
    noData: 'Nessun dato',
    noDistribution: 'Nessuna distribuzione',
    noClassAllocation: 'Nessuna allocazione per classi',
    noAllocation: 'Nessuna allocazione',
    /** Nomi di serie di riserva (identità in legenda/tooltip) quando il
     *  wrapper non ha un titolo proprio da riusare. */
    seriesExposure: 'Esposizione',
    seriesClassAllocation: 'Allocazione per classi',
    /** Controllo di collasso per liste lunghe (es. paesi): espande in place
     *  invece di aprire un viewport con scorrimento interno. */
    showAll: 'Mostra tutti ({count})',
    showLess: 'Mostra meno',
    /** Nomi delle serie dei grafici posizione, capitale e performance
     *  (legenda/tooltip); `Invested`/`Value`/`Realized`/`Gain/Loss` riusano
     *  `hero.*`/`chartView.colValue`/`dashboard.colGainLoss` nel chiamante. */
    seriesCostBasis: 'Costo sostenuto',
    seriesMarketValue: 'Valore di mercato',
    seriesCumulative: 'Cumulato',
    /** Nome della serie dei prezzi di chiusura (minuscolo, come in grafico). */
    seriesClose: 'chiusura',
    /** Etichetta del marcatore di split ({ratio} tipo "2/1"). */
    splitRatio: 'Split {ratio}',
    /** Stato vuoto del grafico prezzi dell'asset. */
    noPriceData: 'Nessun dato prezzi disponibile',
  },
  portfolio: {
    tabsLabel: 'Sezioni del portafoglio',
    tabOverview: 'Panoramica',
    tabPositions: 'Posizioni',
    tabActivity: 'Attività',
    tabAllocation: 'Allocazione',
    back: 'Tutti i portafogli',
    addTransaction: 'Aggiungi transazione',
    actionsMenu: 'Azioni sul portafoglio',
    export: 'Esporta',
    import: 'Importa',
    delete: 'Elimina portafoglio',
    deleteConfirm: 'Eliminare questo portafoglio? Tutte le sue transazioni andranno perse.',
    deleted: 'Portafoglio eliminato',
    viewAllocation: 'Vedi allocazione completa',
    performanceHistory: 'Storico performance',
    seriesPortfolio: 'Portafoglio',
    /** Pagina lista portafogli e relativa modale di creazione (il titolo
     *  della pagina riusa `nav.portfolios`, il pulsante Importa
     *  `portfolio.import`). */
    new: 'Nuovo portafoglio',
    createTitle: 'Crea portafoglio',
    namePlaceholder: 'Nome del portafoglio',
    descriptionLabel: 'Descrizione',
    /** Nome accessibile del collegamento che apre un portafoglio. */
    openNamed: 'Apri {name}',
    emptyTitle: 'Nessun portafoglio',
    emptyHint: 'Crea un portafoglio per iniziare',
    /** Domanda breve del dialogo di eliminazione della lista (il tab
     *  dettaglio porta la `portfolio.deleteConfirm` più completa). */
    deleteQuestion: 'Eliminare questo portafoglio?',
    loadFailed: 'Caricamento dei portafogli non riuscito',
    created: 'Portafoglio creato',
    createFailed: 'Creazione del portafoglio non riuscita',
    /** Modale di import (le etichette riusano `chartView.colName`,
     *  `asset.factCurrency`, `activity.transactions` e `common.delete/cancel`). */
    importTitle: 'Importa portafoglio',
    importHint: 'Scegli un export (.json) di un portafoglio Peculium da importare.',
    chooseFile: 'Scegli file',
    changeFile: 'Cambia file',
    dateRange: 'Intervallo date',
    importModeNew: 'Crea come nuovo portafoglio',
    importModeOverwrite: 'Sovrascrivi portafoglio esistente',
    importTarget: 'Portafoglio di destinazione',
    importTargetPlaceholder: 'Seleziona il portafoglio da sovrascrivere',
    importInvalidFile: 'File non valido: formato di export non riconosciuto',
    importReadFailed: 'Impossibile leggere il file',
    imported: 'Portafoglio importato',
    importFailed: 'Importazione non riuscita',
    /** Fallback del titolo mentre il record del portafoglio è in caricamento. */
    fallbackName: 'Portafoglio',
    /** Fallback dei toast dei caricamenti dello shell e dell'export. */
    detailLoadFailed: 'Caricamento del portafoglio non riuscito',
    historyLoadFailed: 'Caricamento dello storico non riuscito',
    refreshFailed: 'Aggiornamento del portafoglio non riuscito',
    exportFailed: 'Esportazione non riuscita',
  },
  /**
   * Tab Posizioni del portafoglio: titolo della card (anche nome accessibile
   * della tabella delle posizioni), la riga di stato vuoto e le colonne non
   * coperte da `common.col*`, più il badge delle posizioni chiuse.
   */
  positions: {
    title: 'Posizioni',
    noPositions: 'Nessuna posizione',
    colTicker: 'Ticker',
    colCost: 'Costo',
    colRealized: 'Realizzato',
    colUnrealized: 'Non realizzato',
    colRoi: 'ROI',
    colStatus: 'Stato',
    closed: 'Chiusa',
  },
  activity: {
    typeGroup: 'Filtra per tipo di transazione',
    typeAll: 'Tutte',
    typeBuy: 'Acquisto',
    typeSell: 'Vendita',
    typeDividend: 'Dividendo',
    typeSplit: 'Split',
    typeFee: 'Commissione',
    asset: 'Asset',
    assetAll: 'Tutti gli asset',
    assetUnknown: 'Asset (non presente in questo portafoglio)',
    from: 'Dal',
    to: 'Al',
    clearFilters: 'Cancella filtri',
    emptyFiltered: 'Nessuna transazione corrisponde ai filtri',
    emptyFilteredHint: 'Prova a estendere le date o a rimuovere un filtro.',
    transactions: 'Transazioni',
    editTransaction: 'Modifica transazione',
    /** Fallback del toast del fetch della finestra transazioni dello shell. */
    loadFailed: 'Caricamento delle transazioni non riuscito',
  },
  tx: {
    titleNew: 'Nuova transazione',
    titleEdit: 'Modifica transazione',
    deleted: 'Transazione eliminata',
    undo: 'Annulla',
    undoFailed: 'Impossibile ripristinare la transazione',
    /** Toast di esito del flusso di aggiunta/modifica. */
    added: 'Transazione aggiunta',
    updated: 'Transazione aggiornata',
    /** Etichette e placeholder del form non coperti da `common.col*` (le
     *  opzioni del tipo riusano `activity.typeBuy/typeSell/typeDividend`). */
    amount: 'Importo',
    quantity: 'Quantità',
    fees: 'Commissioni',
    notes: 'Note',
    /** Messaggi di validazione inline. */
    selectAsset: 'Seleziona un asset',
    dateRequired: 'La data è obbligatoria',
    amountRequired: "L'importo deve essere maggiore di 0",
    quantityRequired: 'La quantità deve essere maggiore di 0',
    priceRequired: 'Il prezzo deve essere maggiore o uguale a 0',
  },
  asset: {
    tabsLabel: 'Sezioni dettaglio asset',
    tabOverview: 'Panoramica',
    tabExposure: 'Esposizione',
    tabData: 'Dati',
    back: 'Asset',
    refreshMeta: 'Aggiorna da Yahoo',
    backfillHistory: 'Backfill storico completo',
    delete: 'Elimina asset',
    deleteConfirm: 'Eliminare {ticker}? Questa azione non può essere annullata.',
    deleted: 'Asset eliminato',
    priceUpdated: 'Aggiornato il {date}',
    chip1d: '1G',
    chip1w: '1S',
    chip1m: '1M',
    chip1y: '1Y',
    chipYtd: 'YTD',
    whereHeld: 'Dove è detenuto',
    whereHeldEmpty: 'Non è detenuto in nessun portafoglio',
    whereHeldUnavailable: 'Le detenzioni nei portafogli non sono disponibili al momento',
    heldPortfolio: 'Portafoglio',
    heldQty: 'Quantità',
    heldCost: 'Costo',
    heldValue: 'Valore',
    heldGl: 'Guadagno/Perdita',
    quickFacts: 'Dati principali',
    factIsin: 'ISIN',
    factType: 'Tipo',
    factClass: 'Classe',
    factCurrency: 'Valuta',
    factExchange: 'Mercato',
    factPriceSource: 'Fonte prezzo',
    /** Opzioni del selettore fonte prezzo nel tab Dati (brand name tenuto). */
    priceSourceYahoo: 'Yahoo Finance',
    priceSourceManual: 'Prezzo manuale',
    priceSourceNone: 'Nessun prezzo',
    /** Etichette dei tipi asset (chip d'identità, dati principali, selettori
     *  Tipo). */
    typeStock: 'Azione',
    typeEtf: 'ETF',
    typeBond: 'Obbligazione',
    typeMutualFund: 'Fondo comune',
    typeCrypto: 'Cripto',
    typeCommodity: 'Materie prime',
    typeCash: 'Liquidità',
    /** Etichette delle classi di asset (chip d'identità, dati principali,
     *  selettori Classe, nomi delle fette e titoli del drill della donut
     *  classi). */
    classEquity: 'Azioni',
    classBond: 'Obbligazioni',
    classCommodity: 'Materie prime',
    classCurrency: 'Valute',
    classCrypto: 'Crypto',
    classRealEstate: 'Immobiliare',
    classMixed: 'Misto',
    classOther: 'Altro',
    dangerZone: 'Zona pericolosa',
    manualPrice: 'Inserimento prezzo manuale',
    manualPriceHint: 'Inserisci prezzi datati a mano per gli asset senza fonte prezzi automatica.',
    fixedIncome: 'Attributi obbligazionari',
    fixedIncomeHint: 'Dettagli di emittente, scadenza e cedola per le obbligazioni.',
    /** Pagina lista asset e relativa eliminazione (titolo della pagina,
     *  aria-label della tabella e collegamento di ritorno riusano
     *  `nav.assets`; le intestazioni riusano `positions.colTicker`,
     *  `chartView.colName`, `asset.factType`, `asset.factCurrency` e
     *  `common.colActions`). */
    add: 'Aggiungi asset',
    newTitle: 'Nuovo asset',
    lookupHint: 'Cerca un ticker per precompilare i dettagli.',
    /** Copy interna dei selettori di asset: placeholder e testo vuoto del
     *  combobox delle transazioni e dell'autocomplete di ricerca Yahoo. */
    comboboxPlaceholder: 'Seleziona un asset',
    noAssetsFound: 'Nessun asset trovato',
    tickerPlaceholder: 'Ticker (es. AAPL)',
    noResultsFound: 'Nessun risultato trovato',
    colCountry: 'Paese',
    /** Nome accessibile del pulsante di eliminazione della riga
     *  ({ticker} riportato verbatim). */
    deleteNamed: 'Elimina {ticker}',
    /** Domanda breve del dialogo di eliminazione della lista (lo shell del
     *  dettaglio porta la `asset.deleteConfirm` più completa). */
    deleteQuestion: 'Eliminare {ticker}?',
    loadFailed: 'Caricamento degli asset non riuscito',
    created: 'Asset creato',
    /** Titolo della card dello storico prezzi in Panoramica. */
    priceHistory: 'Storico prezzo',
    /** Titolo del form delle caratteristiche nel tab Dati. */
    characteristics: 'Caratteristiche',
    /** Stato vuoto della striscia quotazione e avviso per fonte non-Yahoo
     *  ({source} porta l'etichetta della fonte già localizzata). */
    noPriceData: 'Nessun dato prezzo',
    noAutoSync: '{source} — nessun sync automatico',
    /** Toast delle azioni gestite dallo shell (salvataggio identità,
     *  aggiornamento meta da Yahoo, backfill, prefill/salvataggi dalle
     *  modali); i fallimenti generici riusano `common.saveFailed`/
     *  `common.deleteFailed`. */
    detailLoadFailed: "Caricamento dell'asset non riuscito",
    formRequiredFields: 'Ticker, Nome e Valuta sono obbligatori',
    updated: 'Asset aggiornato',
    metaRefreshed: 'Campi aggiornati da Yahoo',
    metaRefreshFailed: 'Aggiornamento non riuscito',
    backfillDone: 'Storico prezzi aggiornato',
    backfillFailed: 'Backfill non riuscito',
    countriesPrefilledJustEtf: 'Paesi precompilati da JustETF',
    countriesSectorsPrefilledMorningstar: 'Paesi e settori precompilati da Morningstar',
    regionsPrefilledMorningstar: 'Regioni precompilate da Morningstar',
    sectorsPrefilledJustEtf: 'Distribuzione settoriale precompilata da JustETF',
    sectorsPrefilledYahoo: 'Distribuzione settoriale precompilata da Yahoo',
    sectorsPrefilledMorningstar: 'Distribuzione settoriale precompilata da Morningstar',
    yahooNoResponse: 'Yahoo non ha risposto',
    prefillFailed: 'Prefill non riuscito',
    downloadFailed: 'Download non riuscito',
    noWeightedCountries: 'Nessun paese con peso: aggiungi prima dei paesi',
    regionsRecomputed: 'Regioni ricalcolate dai paesi',
    computeFailed: 'Calcolo non riuscito',
    geoSaved: 'Distribuzione geografica salvata',
    sectorsSaved: 'Distribuzione settoriale salvata',
    countriesSaved: 'Distribuzione paesi salvata',
  },
} satisfies Dictionary
