/**
 * English dictionary (EPIC K.1b, decision D1) — the canonical shape every
 * locale must match. Keys live in a two-level nested structure (`group.key`
 * in the file); `index.svelte.ts` flattens them into dot-joined lookup keys
 * (`nav.dashboard`) exposed through the `MessageKey` union, so `t()` calls
 * are checked at compile time and `it.ts` cannot drift (it is verified with
 * `satisfies Dictionary`).
 *
 * Conventions:
 * - add new strings here first, then mirror them in `it.ts`;
 * - every new string from K.1 on ships in both languages; legacy pages keep
 *   their mixed EN/IT copy until their own migration sweep (progressive);
 * - values are plain strings with optional `{name}` placeholders; labels
 *   use sentence case without trailing punctuation.
 */
export const en = {
  common: {
    language: 'Language',
    /** Generic confirm-dialog labels (used e.g. by the portfolio delete). */
    delete: 'Delete',
    cancel: 'Cancel',
    /** Positive ConfirmDialog fallback (paired with `cancel`). */
    confirm: 'Confirm',
    /** Close affordance on sheets/dialogs (e.g. the transaction sheet, K.4c). */
    close: 'Close',
    /** Held-asset count line (dashboard portfolio cards, portfolio detail). */
    assetCount: '{count} assets',
    /** Column headers shared by the positions and transactions tables. */
    colAsset: 'Asset',
    colDate: 'Date',
    colType: 'Type',
    colQty: 'Qty',
    colPrice: 'Price',
    colValue: 'Value',
    colTotal: 'Total',
    colActions: 'Actions',
    /** Code label shared by the currencies form/table and the health
     *  events table. */
    colCode: 'Code',
    /** Pagination footer buttons of the transactions table. */
    previous: 'Previous',
    next: 'Next',
    /** Row-range readout next to the pagination buttons ("1–20 of 137",
     *  empty variant when the window has no rows). */
    rangeLabel: '{from}–{to} of {total}',
    rangeEmpty: '0 of {total}',
    /** Progress line shown while a list/table is still loading. */
    loading: 'Loading…',
    /** Default empty-state title of the generic async card shell. */
    nothingHere: 'Nothing here yet',
    /** Retry action on error states (AsyncCard error surface). */
    retry: 'Retry',
    /** Generic form-button labels shared by the create/edit dialogs. */
    create: 'Create',
    save: 'Save',
    saving: 'Saving…',
    saveChanges: 'Save changes',
    /** Hint rendered under an optional form-field label. */
    optional: 'optional',
    /** Generic failure fallbacks when the API carries no message. */
    deleteFailed: 'Delete failed',
    saveFailed: 'Save failed',
    createFailed: 'Create failed',
    somethingWentWrong: 'Something went wrong',
  },
  /**
   * Accessible names baked into the generic primitives (spinner, modal,
   * toast, drawer/sheet, tabs). Screen-reader-only copy, never shown.
   */
  a11y: {
    loading: 'Loading',
    closeDialog: 'Close dialog',
    dismissNotification: 'Dismiss notification',
    /** Default ✕ name of the drawer/sheet pair (callers may override). */
    closePanel: 'Close panel',
    /** Untitled drawer/sheet fallback. */
    details: 'Details',
    /** Tabs strip fallback when the caller passes no accessible name. */
    sections: 'Sections',
  },
  nav: {
    /** Accessible name of the sidebar/drawer `<nav>` landmark. */
    main: 'Main',
    dashboard: 'Dashboard',
    /** Phone bottom-nav destination for `/` (EPIC K.2, decision D2). */
    overview: 'Overview',
    portfolios: 'Portfolios',
    assets: 'Assets',
    /** Bottom-nav item that opens the off-canvas "More" sheet (D2). */
    more: 'More',
    /** Accessible name of the fixed phone bottom navigation landmark. */
    bottomNav: 'Primary navigation',
    /**
     * Price-sync health page (decision D7). Referenced once, from the
     * `adminItems` config in `SidebarNav.svelte`, so the whole entry —
     * label included — can be relocated to an Administration menu later.
     */
    dataSync: 'Data & Sync',
    settings: 'Settings',
    /** Sidebar section headers (visible only when the sidebar is expanded). */
    sectionAdmin: 'Admin',
    sectionSettings: 'Settings',
    skipToContent: 'Skip to content',
  },
  header: {
    expandSidebar: 'Expand sidebar',
    collapseSidebar: 'Collapse sidebar',
  },
  fab: {
    /** Accessible name of the phone-only quick-actions floating button. */
    open: 'Open quick actions',
  },
  quickActions: {
    /** Title of the sheet opened by the Fab (decision D2). */
    title: 'Quick actions',
    addTransaction: 'Add transaction',
    addTransactionHint: 'Record a buy, sell or dividend',
    addAsset: 'Add asset',
    addAssetHint: 'Search Yahoo and register it',
    refreshPrices: 'Refresh prices',
    refreshPricesHint: 'Fetch the latest quotes now',
    /** Reserved slot until EPIC J.1 ships: rendered disabled. */
    enterPrice: 'Enter price',
    comingSoon: 'Coming soon',
    /** Fab refresh feedback (toasts). */
    refreshSuccess: 'Prices updated',
    refreshError: 'Price refresh failed',
    refreshRateLimited: 'Yahoo Finance is rate-limiting requests: some prices may be stale',
    refreshIssues: '{count} price updates failed (Yahoo)',
  },
  user: {
    /** Compact icon-only trigger (rail / mobile header). */
    accountMenu: 'Account menu',
    signOut: 'Sign out',
    /** Fallback shown when the account has no name/email. */
    fallbackName: 'User',
  },
  /**
   * Login / register screen (single page, mode toggle). The "Peculium"
   * wordmark is a proper noun and stays as-is; everything else — mode
   * labels, headings, field labels/hints, validation, toasts — lives here.
   */
  login: {
    tagline: 'Your wealth, self-hosted',
    /** SegmentedControl mode options, reused verbatim by the submit CTA. */
    signIn: 'Sign in',
    register: 'Register',
    /** Subtitle switching with the mode, under the brand block. */
    createAccount: 'Create an account',
    signInToAccount: 'Sign in to your account',
    /** Accessible name of the mode SegmentedControl. */
    authMode: 'Authentication mode',
    /** Field labels. */
    name: 'Name',
    email: 'Email',
    password: 'Password',
    confirmPassword: 'Confirm password',
    /** Register-mode hint under the password field. */
    passwordHint: 'At least 8 characters',
    /** Inline validation messages. */
    emailRequired: 'Email is required',
    emailInvalid: 'Enter a valid email address',
    passwordRequired: 'Password is required',
    passwordTooShort: 'Password must be at least 8 characters',
    passwordMismatch: 'Passwords do not match',
    nameRequired: 'Name is required',
    /** Success toast of the register flow (then the form switches to Sign in). */
    registered: 'Registered! You can now log in.',
  },
  theme: {
    /** Field label on Preferences and group name of the header popup. */
    group: 'Theme',
    light: 'Light',
    dark: 'Dark',
    system: 'System',
    /** Trigger aria-labels; `{theme}`/`{resolved}` carry a lower-cased label. */
    aria: 'Theme: {theme}',
    ariaSystem: 'Theme: {theme}, currently {resolved}',
  },
  settingsTabs: {
    /** Accessible name of the Settings section tab bar. */
    sections: 'Settings sections',
    profile: 'Profile',
    password: 'Password',
    preferences: 'Preferences',
    currencies: 'Currencies',
  },
  /**
   * Global command palette (EPIC K.5a, spec §8.1). Destination/action labels
   * reused from `nav.*`, `settingsTabs.*`, `quickActions.*`, `theme.*` and
   * `preferences.palette*` live there; this group only holds the palette's
   * own copy. `{query}` carries the raw typed text.
   */
  commandPalette: {
    /** Accessible name of the dialog. */
    title: 'Command palette',
    /** aria-label of the header search trigger. */
    trigger: 'Open command palette',
    /** Visible pill text (lg+) and aria-label of the search input. */
    inputLabel: 'Search',
    placeholder: 'Search pages, portfolios, assets…',
    sectionGoTo: 'Go to',
    sectionAssets: 'Assets',
    sectionActions: 'Actions',
    noResults: 'No results',
    yahooRow: 'Search Yahoo for “{query}”',
    searching: 'Searching Yahoo…',
    /** Action rows (the hints preview the target state of the toggles). */
    toggleTheme: 'Toggle theme',
    toggleCvd: 'Toggle color-blind palette',
    toggleSidebar: 'Toggle sidebar',
    /** Footer key hints, composed after a <kbd> glyph. */
    hintNavigate: 'to navigate',
    hintSelect: 'to select',
    hintClose: 'to close',
  },
  preferences: {
    title: 'Preferences',
    themeHint: 'Light, dark, or follow the device setting (System).',
    languageHint: 'Applied immediately and remembered on this device.',
    /** Gain/loss palette control (decision D6, EPIC K.5c): also the
     *  accessible name of the SegmentedControl tablist. The option labels
     *  stay short ("Green/Red" / "Blue/Orange") so the control cannot
     *  overflow its card at phone widths (EPIC K bug-fix); the command
     *  palette reuses them as the toggle's target-state hint, and
     *  `paletteHint` carries the full explanation. */
    colorGroup: 'Gain/loss colors',
    paletteClassic: 'Green/Red',
    paletteCvd: 'Blue/Orange',
    paletteHint: 'Swaps green/red for blue/orange in text and charts. Signs and ▲▼ arrows stay either way.',
  },
  /**
   * Settings → Profile card (the page title reuses `nav.settings`, the
   * card heading `settingsTabs.profile`, the Name label `chartView.colName`
   * and the save button `common.save`/`common.saving`).
   */
  profile: {
    email: 'Email',
    baseCurrency: 'Base currency',
    baseCurrencyHint: 'Used to consolidate values across portfolios on the dashboard.',
    updated: 'Profile updated',
    updateFailed: 'Update failed',
  },
  /** Settings → Password card (labels, inline validation and toasts). */
  password: {
    change: 'Change password',
    current: 'Current password',
    new: 'New password',
    confirm: 'Confirm new password',
    minLengthHint: 'At least 8 characters',
    currentRequired: 'Current password is required',
    tooShort: 'Password must be at least 8 characters',
    mismatch: 'Passwords do not match',
    currentIncorrect: 'Current password is incorrect',
    changed: 'Password changed',
    changeFailed: 'Change failed',
  },
  /**
   * Settings → Currencies card (the Code label reuses `common.colCode`,
   * Name/Actions `chartView.colName`/`common.colActions`, the loading line
   * and dialog chrome `common.*`). `{code}` carries the plain currency code.
   */
  currencies: {
    title: 'Managed currencies',
    select: 'Select a currency',
    namePlaceholder: 'Optional',
    allManaged: 'All listed currencies are already managed.',
    empty: 'No currencies found.',
    add: 'Add',
    adding: 'Adding…',
    added: 'Currency {code} added',
    conversionUnavailable: 'USD->{code} conversion not available; currency not manageable',
    alreadyPresent: 'Currency already present',
    addFailed: 'Failed to add currency',
    removed: 'Currency {code} removed',
    remove: 'Remove currency',
    removeNamed: 'Remove currency {code}',
    removeFailed: 'Failed to remove currency',
    inUse: 'Currency in use or protected',
    deleteTitle: 'Delete currency',
    deleteConfirm: 'Delete currency {code}?',
    loadFailed: 'Failed to load currencies',
  },
  /**
   * Allocation surfaces (EPIC K bug-fix, progressive D1 sweep): the wealth
   * "Overall allocation" card on the dashboard, the portfolio Allocation
   * tab and its Overview digest — panel headings, isolated error states and
   * the equity-universe coverage note (`{pct}` carries one decimal).
   */
  allocation: {
    title: 'Overall allocation',
    unavailable: 'Allocation unavailable',
    classUnavailable: 'Class allocation unavailable',
    sectorUnavailable: 'Sector allocation unavailable',
    geoUnavailable: 'Geographic allocation unavailable',
    assetClasses: 'Asset classes',
    sectorsEquity: 'Sectors (equity only)',
    regionsEquity: 'Regions (equity only)',
    countriesEquity: 'Countries (equity only)',
    /** Equity-universe coverage note shown when non-equity was excluded. */
    equityUniverse: 'Equity universe: {pct}% of the portfolio',
    /** Dashboard donut: wealth value split per portfolio. */
    byPortfolio: 'Allocation by portfolio',
    mixedCurrencies:
      'Portfolios use different currencies: values are not comparable, shares are indicative.',
  },
  /**
   * Allocation drill-down (EPIC K.5, spec §6.5): the drawer/sheet opened by
   * clicking an allocation slice or bar, listing the contributing assets of
   * that bucket. The visible title is the bucket's friendly label passed in
   * by the calling chart; the Value/Weight headers reuse `chartView.colValue`
   * /`chartView.colWeight` and the ✕ accessible name reuses `common.close`.
   */
  drill: {
    /** Sr-only `<caption>` of the contributing-assets table ({name} = bucket label). */
    caption: '{name} — contributing assets',
    /** Muted subtitle above the table; `{count}` = number of rows. */
    contributingAssets: '{count} contributing assets',
    colAsset: 'Asset',
    /** The asset's amount inside the bucket (value × exposure weight). */
    colContribution: 'Contribution',
    /** Contribution ÷ bucket total: the asset's share of the slice. */
    colShare: 'Share of slice',
    empty: 'No assets in this slice',
    error: 'Could not load the assets of this slice',
    retry: 'Retry',
  },
  /**
   * Asset Exposure tab and its edit modals (EPIC K bug-fix): card headings,
   * panel titles, the edit buttons and the equity-only banner.
   */
  exposure: {
    geoTitle: 'Geographic distribution',
    sectorTitle: 'Sector distribution',
    countries: 'Countries',
    regions: 'Regions',
    sectors: 'Sectors',
    noCountries: 'No countries added',
    noCountriesHint: 'Add a country below, or use a JustETF / Morningstar prefill',
    modify: 'Edit',
    editGeo: 'Edit geographic distribution',
    editSectors: 'Edit sector distribution',
    /** Banner replacing the cards for assets outside the equity universe. */
    universeTitle: 'Geographic and sector distribution',
    universeHint:
      'This distribution only applies to equity assets (stocks and equity-class ETFs/funds).',
    universeClassHint: "Set the 'Stocks' or 'Real estate' class in Characteristics to enable it.",
    /** Edit-modal chrome: column headers, row accessible names and the add
     *  control (the Save footer reuses `common.save`/`common.saving`/
     *  `common.colTotal`). */
    colCountry: 'Country',
    colWeightPct: 'Weight %',
    geoArea: 'Geographic area',
    gicsSector: 'GICS sector',
    weightAria: 'Weight of {name}',
    removeAria: 'Remove {name}',
    addCountryAria: 'Country to add',
    add: 'Add',
    /** Provider prefill buttons: short tooltips plus row-specific aria. */
    prefillJustEtf: 'Prefill from JustETF',
    prefillYahoo: 'Prefill from Yahoo',
    prefillMorningstar: 'Prefill from Morningstar',
    prefillCountriesJustEtf: 'Prefill countries from JustETF',
    prefillCountriesMorningstar: 'Prefill countries from Morningstar',
    prefillRegionsMorningstar: 'Prefill regions from Morningstar',
    prefillSectorsJustEtf: 'Prefill sectors from JustETF',
    prefillSectorsYahoo: 'Prefill sectors from Yahoo',
    prefillSectorsMorningstar: 'Prefill sectors from Morningstar',
    deriveTitle: 'Compute from countries',
    deriveAria: 'Compute regions from countries',
    /** Weight-sum validation messages of the modal footers ({pct} carries
     *  the already-formatted two-decimal sum). */
    overSum: 'The sum exceeds 100% — currently {pct}%. Lower the weights to save.',
    over100Title: 'The weights sum to over 100%: lower them to be able to save',
    residualCountries: 'Unallocated residual: {pct}%.',
    residualRegions: 'Unclassified residual: {pct}% — excluded from the chart.',
    sumMustBe100: 'The weights must sum to 100 (±0.5) — current: {pct}%',
  },
  /**
   * Provenance badges (EPIC K bug-fix): the pill label, the tooltip/aria
   * explanation and the date joiner (`{date}` is the raw YYYY-MM-DD stamp).
   * One flat key per provenance id (`manualLabel`/`manualDesc`, …) because
   * the dictionary runtime supports exactly two nesting levels.
   */
  provenance: {
    updatedTo: 'updated {date}',
    manualLabel: 'manual',
    manualDesc: 'Data edited manually',
    justetfLabel: 'from JustETF',
    justetfDesc: 'Country list imported from JustETF, not edited manually',
    morningstarLabel: 'from Morningstar',
    morningstarDesc: 'Data imported from Morningstar, not edited manually',
    morningstarRegionsLabel: 'from Morningstar (official regions)',
    morningstarRegionsDesc: 'Official regions imported from Morningstar, not edited manually',
    yahooLabel: 'from Yahoo',
    yahooDesc: 'Sectors imported from Yahoo, not edited manually',
    derivedLabel: 'derived from countries',
    derivedDesc: 'Regions derived from the country weights',
    derivedEtfLabel: 'from JustETF via countries',
    derivedEtfDesc: 'Regions derived from the countries imported from JustETF',
  },
  /**
   * Wealth Overview hero zone (EPIC K.3a, redesign spec §6.1 zone A). The
   * `Investments` roll-up heading lives in `investments.title` and the
   * Performance card copy in `performance.*`.
   */
  hero: {
    netValue: 'Net value',
    allTime: 'all-time',
    /** Headline of the value-vs-invested chart next to the hero number. */
    valueVsInvested: 'Value vs invested',
    /** Progressive disclosure wrapping the Active/Closed table. */
    breakdown: 'Breakdown',
    realized: 'Realized',
    dividends: 'Dividends',
    invested: 'Invested',
  },
  /**
   * Performance card (shared by the dashboard wealth view and the portfolio
   * detail Overview): the card heading and its Monthly/Annual granularity
   * toggle.
   */
  performance: {
    title: 'Performance',
    granularity: 'Performance granularity',
    monthly: 'Monthly',
    annual: 'Annual',
  },
  /**
   * Dashboard zone copy: card headings not owned by another group, the
   * compact closed line on the portfolio cards and the consolidated
   * "Invested assets" table (EPIC I.5) with its headers and no-price badge.
   */
  dashboard: {
    closedPrefix: 'Closed:',
    investedAssets: 'Invested assets',
    noInvestedAssets: 'No invested assets yet',
    noInvestedAssetsHint: 'Open positions will appear here once you record transactions in your portfolios.',
    colAsset: 'Asset',
    colGainLoss: 'Gain/Loss',
    colPnlPct: 'P/L %',
    noPrice: 'no price',
    noPriceHint: 'No price data: value is carried at cost, so its P/L is 0',
  },
  /**
   * "Investments" roll-up card (Active/Closed breakdown): shared by the
   * dashboard hero disclosure and the portfolio detail Overview.
   */
  investments: {
    title: 'Investments',
    group: 'Group',
    invested: 'Invested',
    valueProceeds: 'Value / Proceeds',
    gainLoss: 'Gain/Loss',
    pct: '%',
    dividends: 'Dividends',
    active: 'Active',
    closed: 'Closed',
  },
  /** Period chips on the hero chart (decision D10: bucket-driven ranges). */
  period: {
    oneYear: '1Y',
    threeYears: '3Y',
    all: 'ALL',
    group: 'Chart period',
  },
  /**
   * Data-quality strip chips (spec §6.1/§8.5). Each chip is a link to the
   * fixing surface, so labels name the problem, not the destination.
   */
  quality: {
    fxMissing: '{amount} excluded — missing FX ({count} holdings)',
    rateLimited: 'Some prices not updated (Yahoo rate limit)',
    refreshIssues: '{count} price updates failed',
    refreshFailed: 'Price refresh failed — values may be stale',
  },
  /** Freshness stamp near the hero (spec §8.5), from the session refresh. */
  freshness: {
    asOf: 'Prices as of {time}',
    refreshing: 'Refreshing prices…',
    /** Plain-text tooltip on the completed stamp. */
    hint: 'Consolidated values use these prices',
    partialHint: 'Prices as of {time} — some updates failed or were rate-limited',
  },
  /**
   * Admin price-sync health page (`/admin/health`, nav entry `nav.dataSync`).
   * Table values fetched from the backend (`event_type`, `code`, `message`)
   * are rendered verbatim; only the status badge is localised through the
   * `status*` keys below, falling back to the raw value for unknown
   * statuses. `N/A` and the `ms` duration unit are kept as technical terms.
   */
  health: {
    title: 'Price Sync Health',
    subtitle: 'Monitoring Yahoo Finance API connectivity and performance',
    periodToday: 'Today',
    periodLast24h: 'Last 24h',
    periodLast100: 'Last 100',
    /** SegmentedControl accessible name + "Period: …" caption below. */
    periodAria: 'Health period',
    periodLabel: 'Period: {period}',
    refreshing: 'Refreshing…',
    refresh: 'Refresh Now',
    noData: 'No health data available.',
    /** Metric cards. */
    successRate: 'Success Rate',
    totalSuccesses: 'Total Successes',
    totalFailures: 'Total Failures',
    rateLimited: 'Rate Limited',
    /** Events table (heading, table aria-label and headers; Type/Status/Code
     *  reuse `common.colType`, `positions.colStatus` and `common.colCode`). */
    recentEvents: 'Recent Events',
    colTimestamp: 'Timestamp',
    colMessage: 'Message',
    colDuration: 'Duration',
    /** Status-badge labels for the known backend values. */
    statusSuccess: 'Success',
    statusRateLimited: 'Rate limited',
    statusFailure: 'Failure',
    loadFailed: 'Failed to fetch health data',
  },
  /** First-run checklist replacing the empty-wealth EmptyState (D8). */
  checklist: {
    title: 'Set up your wealth',
    intro: 'Three steps to start tracking your investments.',
    stepPortfolio: 'Create a portfolio',
    stepPortfolioHint: 'Group investments by goal or account.',
    stepAsset: 'Add an asset',
    stepAssetHint: 'Search Yahoo and register what you own.',
    stepTransaction: 'Record a transaction',
    stepTransactionHint: 'Open a portfolio and log a buy.',
    done: 'Done',
    current: 'Current step',
    pending: 'Not started',
  },
  /** Wealth ⇄ portfolio scope switcher in the Overview header (D3). */
  scope: {
    label: 'Scope',
    all: 'All portfolios (Wealth)',
  },
  /**
   * Portfolio-card sparklines on the wealth Overview (EPIC K.3b, spec §6.1
   * zone C). The shape is supplementary (the card already carries value and
   * P/L), so these are accessible names, not visible copy.
   */
  sparkline: {
    /** Generic fallback when the caller has no better label. */
    trend: 'Value trend',
    /** Per-card aria-label with the portfolio name interpolated. */
    valueTrend: '{name} value trend',
  },
  /**
   * "View as table" chart toggle (EPIC K.5b, redesign spec §9.1): labels
   * and accessible names of the shared `ui/ChartTableToggle`, the sr-only
   * `<caption>` template of the table views and their column headers.
   * `{name}` carries the chart's own heading when it has one.
   */
  chartView: {
    chart: 'Chart',
    table: 'Table',
    /** Tablist accessible name (with/without the interpolated chart title). */
    aria: 'Chart data view',
    ariaNamed: 'Chart data view — {name}',
    /** Screen-reader caption of the table view. */
    caption: '{name} — chart data',
    captionGeneric: 'Chart data',
    /** Column headers shared by the table views (also interpolated into the
     *  ECharts tooltips as the `Value:`/`Weight:` labels, EPIC K bug-fix). */
    colName: 'Name',
    colValue: 'Value',
    colWeight: 'Weight',
    colPeriod: 'Period',
    colReturn: 'Return',
    colCumulative: 'Cumulative TWR',
    colInvested: 'Invested',
    /** Accessible names for the bucket charts, which carry no own heading. */
    namePerformance: 'Performance',
    nameCapital: 'Invested vs value',
    /** Fallback titles of the allocation/exposure donut wrappers when the
     *  caller passes no `title` (series name, toggle and caption). */
    nameAllocation: 'Allocation',
    nameDistribution: 'Distribution',
    /** Empty states of the allocation/exposure chart wrappers. */
    noData: 'No data',
    noDistribution: 'No distribution',
    noClassAllocation: 'No class allocation',
    noAllocation: 'No allocation',
    /** Fallback series names (legend/tooltip identity) when the wrapper has
     *  no own heading to reuse. */
    seriesExposure: 'Exposure',
    seriesClassAllocation: 'Class allocation',
    /** Collapse control of long bar lists (e.g. countries): expands in place
     *  instead of opening an inner scroll viewport. */
    showAll: 'Show all ({count})',
    showLess: 'Show less',
    /** Series names (legend/tooltip identity) of the position, capital and
     *  performance charts; `Invested`/`Value`/`Realized`/`Gain/Loss` reuse
     *  `hero.*`/`chartView.colValue`/`dashboard.colGainLoss` at the call site. */
    seriesCostBasis: 'Cost basis',
    seriesMarketValue: 'Market value',
    seriesCumulative: 'Cumulative',
    /** Price-chart close-series name (kept lower-case, as plotted). */
    seriesClose: 'close',
    /** Stock-split marker label ({ratio} like "2/1"). */
    splitRatio: 'Split {ratio}',
    /** Empty state of the asset price chart. */
    noPriceData: 'No price data available',
  },
  /**
   * Portfolio-detail shell (EPIC K.4a, spec §6.2): sticky-header chrome and
   * the four tier-2 tabs, plus the shell's load/export toast fallbacks.
   */
  portfolio: {
    /** Accessible name of the portfolio tab bar. */
    tabsLabel: 'Portfolio sections',
    tabOverview: 'Overview',
    tabPositions: 'Positions',
    tabActivity: 'Activity',
    tabAllocation: 'Allocation',
    /** Back link in the sticky header, to the portfolios list. */
    back: 'All portfolios',
    addTransaction: 'Add transaction',
    /** Accessible name of the `⋯` actions menu in the sticky header. */
    actionsMenu: 'Portfolio actions',
    export: 'Export',
    import: 'Import',
    delete: 'Delete portfolio',
    deleteConfirm: 'Delete this portfolio? All its transactions will be lost.',
    deleted: 'Portfolio deleted',
    /** Overview allocation digest → link to the Allocation tab. */
    viewAllocation: 'View full allocation',
    /** Overview capital-history card heading (below the percentage chart). */
    performanceHistory: 'Performance history',
    /** Aggregate-series option of that card's asset selector. */
    seriesPortfolio: 'Portfolio',
    /** Portfolios list page and its create dialog (the page title reuses
     *  `nav.portfolios`, the Import trigger `portfolio.import`). */
    new: 'New Portfolio',
    createTitle: 'Create Portfolio',
    namePlaceholder: 'Portfolio name',
    descriptionLabel: 'Description',
    /** Accessible name of the card link opening a portfolio. */
    openNamed: 'Open {name}',
    emptyTitle: 'No portfolios yet',
    emptyHint: 'Create one to get started',
    /** Short list-page delete-dialog question (the detail tab carries the
     *  fuller `portfolio.deleteConfirm`). */
    deleteQuestion: 'Delete this portfolio?',
    loadFailed: 'Failed to load portfolios',
    created: 'Portfolio created',
    createFailed: 'Failed to create portfolio',
    /** Import dialog (labels reuse `chartView.colName`, `asset.factCurrency`,
     *  `activity.transactions` and `common.delete/cancel`). */
    importTitle: 'Import portfolio',
    importHint: 'Choose a Peculium portfolio export (.json) to import.',
    chooseFile: 'Choose file',
    changeFile: 'Change file',
    dateRange: 'Date range',
    importModeNew: 'Create as new portfolio',
    importModeOverwrite: 'Overwrite existing portfolio',
    importTarget: 'Target portfolio',
    importTargetPlaceholder: 'Select portfolio to overwrite',
    importInvalidFile: 'Invalid file: export format not recognized',
    importReadFailed: 'Could not read the file',
    imported: 'Portfolio imported',
    importFailed: 'Import failed',
    /** Heading fallback while the portfolio record is still loading. */
    fallbackName: 'Portfolio',
    /** Toast fallbacks of the shell's loads and the export action. */
    detailLoadFailed: 'Failed to load portfolio',
    historyLoadFailed: 'Failed to load history',
    refreshFailed: 'Failed to refresh portfolio',
    exportFailed: 'Export failed',
  },
  /**
   * Portfolio Positions tab: the card heading (also the accessible name of
   * the positions table), the empty line and the columns not covered by
   * `common.col*`, plus the closed-position badge.
   */
  positions: {
    title: 'Positions',
    noPositions: 'No positions',
    colTicker: 'Ticker',
    colCost: 'Cost',
    colRealized: 'Realized',
    colUnrealized: 'Unrealized',
    colRoi: 'ROI',
    colStatus: 'Status',
    closed: 'Closed',
  },
  /**
   * Portfolio Activity tab (EPIC K.4c, spec §6.2/§8.2): the "Transactions"
   * card heading (also the accessible name of the transactions table), the
   * row edit button and the filter row — type chips, asset picker and date
   * range, persisted in the tab's URL query.
   */
  activity: {
    /** Accessible name of the transaction-type chip radiogroup. */
    typeGroup: 'Filter by transaction type',
    typeAll: 'All',
    typeBuy: 'Buy',
    typeSell: 'Sell',
    typeDividend: 'Dividend',
    typeSplit: 'Split',
    typeFee: 'Fee',
    /** Visible inline label of the asset picker. */
    asset: 'Asset',
    assetAll: 'All assets',
    /** Placeholder row for a deep-linked asset outside this portfolio. */
    assetUnknown: 'Asset (not in this portfolio)',
    /** Visible inline labels of the inclusive date-range inputs. */
    from: 'From',
    to: 'To',
    clearFilters: 'Clear filters',
    /** Filtered-empty state (the row exists but matches no filter). */
    emptyFiltered: 'No transactions match these filters',
    emptyFilteredHint: 'Try widening the date range or clearing a filter.',
    /** Activity card heading (also the transactions table's accessible name). */
    transactions: 'Transactions',
    /** Accessible name of the row edit button in the transactions table. */
    editTransaction: 'Edit transaction',
    /** Toast fallback of the shell's transactions-window fetch. */
    loadFailed: 'Failed to load transactions',
  },
  /**
   * Transaction form + delete feedback (EPIC K.4c): titles are shared by
   * the `ui/Modal` (≥ `sm`) and the `ui/Sheet` (phone) containers; the
   * delete strings drive the undo-based flow (decision D11). The form's
   * inner field labels, placeholders and validation messages live here too;
   * the shared ones come from `common.*` and `activity.*`.
   */
  tx: {
    titleNew: 'New Transaction',
    titleEdit: 'Edit Transaction',
    deleted: 'Transaction deleted',
    /** Undo action inside the delete toast (5 s window). */
    undo: 'Undo',
    undoFailed: 'The transaction could not be restored',
    /** Success toasts of the add/edit flow. */
    added: 'Transaction added',
    updated: 'Transaction updated',
    /** Form labels and placeholders not covered by `common.col*` (the type
     *  options reuse `activity.typeBuy/typeSell/typeDividend`). */
    amount: 'Amount',
    quantity: 'Quantity',
    fees: 'Fees',
    notes: 'Notes',
    /** Inline validation messages. */
    selectAsset: 'Select an asset',
    dateRequired: 'Date is required',
    amountRequired: 'Amount must be greater than 0',
    quantityRequired: 'Quantity must be greater than 0',
    priceRequired: 'Price must be 0 or greater',
  },
  /**
   * Asset-detail shell (EPIC K.4b, spec §6.3): sticky-header chrome, the
   * three tier-2 tabs, the "Where held" Overview block, the danger-zone
   * and reserved-section labels, plus the shell's action toasts.
   */
  asset: {
    /** Accessible name of the asset tab bar. */
    tabsLabel: 'Asset sections',
    tabOverview: 'Overview',
    tabExposure: 'Exposure',
    tabData: 'Data',
    /** Back link in the sticky header, to the assets library list. */
    back: 'Assets',
    refreshMeta: 'Update from Yahoo',
    backfillHistory: 'Backfill full history',
    delete: 'Delete asset',
    deleteConfirm: 'Delete {ticker}? This cannot be undone.',
    deleted: 'Asset deleted',
    /** Muted date under the quote chips: `{date}` is locale-formatted. */
    priceUpdated: 'Updated {date}',
    /** Compact quote-delta chips (1D/1W/1M/1Y/YTD). */
    chip1d: '1D',
    chip1w: '1W',
    chip1m: '1M',
    chip1y: '1Y',
    chipYtd: 'YTD',
    /** Overview "Where held" block (spec §4.2 decision 5). */
    whereHeld: 'Where held',
    whereHeldEmpty: 'Not held in any portfolio',
    whereHeldUnavailable: 'Holding portfolios are unavailable right now',
    heldPortfolio: 'Portfolio',
    heldQty: 'Qty',
    heldCost: 'Cost',
    heldValue: 'Value',
    heldGl: 'Gain/Loss',
    /** Overview read-only identity grid. */
    quickFacts: 'Quick facts',
    factIsin: 'ISIN',
    factType: 'Type',
    factClass: 'Class',
    factCurrency: 'Currency',
    factExchange: 'Exchange',
    factPriceSource: 'Price source',
    /** Options of the Data-tab price-source select (brand name kept). */
    priceSourceYahoo: 'Yahoo Finance',
    priceSourceManual: 'Manual price',
    priceSourceNone: 'No price',
    /** Asset-type labels (identity chips, quick facts, Type selects). */
    typeStock: 'Stock',
    typeEtf: 'ETF',
    typeBond: 'Bond',
    typeMutualFund: 'Mutual fund',
    typeCrypto: 'Crypto',
    typeCommodity: 'Commodity',
    typeCash: 'Cash',
    /** Asset-class labels (identity chips, quick facts, Class selects,
     *  class-donut slice names and drill titles). */
    classEquity: 'Equities',
    classBond: 'Bonds',
    classCommodity: 'Commodities',
    classCurrency: 'Currencies',
    classCrypto: 'Crypto',
    classRealEstate: 'Real estate',
    classMixed: 'Mixed',
    classOther: 'Other',
    /** Data tab danger zone (same actions as the header `⋯` menu). */
    dangerZone: 'Danger zone',
    /** Reserved EPIC J placeholders (no behaviour yet). */
    manualPrice: 'Manual price entry',
    manualPriceHint: 'Record dated prices by hand for assets without an automatic feed.',
    fixedIncome: 'Fixed-income attributes',
    fixedIncomeHint: 'Issuer, maturity and coupon details for bonds.',
    /** Assets list page and its delete flows (the page title, the table
     *  aria-label and the `⋯` back link all reuse `nav.assets`; headers reuse
     *  `positions.colTicker`, `chartView.colName`, `asset.factType`,
     *  `asset.factCurrency` and `common.colActions`). */
    add: 'Add Asset',
    newTitle: 'New Asset',
    lookupHint: 'Look up a ticker to prefill the details.',
    /** Asset pickers' built-in copy: the transaction combobox placeholder
     *  and empty text, the Yahoo lookup placeholder and empty text. */
    comboboxPlaceholder: 'Select asset',
    noAssetsFound: 'No assets found',
    tickerPlaceholder: 'Ticker (e.g. AAPL)',
    noResultsFound: 'No results found',
    colCountry: 'Country',
    /** Accessible name of the row trash button ({ticker} verbatim). */
    deleteNamed: 'Delete {ticker}',
    /** Short list-page delete-dialog question (the detail shell carries the
     *  fuller `asset.deleteConfirm`). */
    deleteQuestion: 'Delete {ticker}?',
    loadFailed: 'Failed to load assets',
    created: 'Asset created',
    /** Overview price-history card heading. */
    priceHistory: 'Price history',
    /** Data tab characteristics form heading. */
    characteristics: 'Characteristics',
    /** Header quote-strip empty state and the non-Yahoo source warning
     *  ({source} carries the already-localised price-source label). */
    noPriceData: 'No price data',
    noAutoSync: '{source} — no automatic sync',
    /** Toasts of the shell actions the layout orchestrates (identity save,
     *  Yahoo meta refresh, backfill, modal prefills/saves); generic failures
     *  reuse `common.saveFailed`/`common.deleteFailed`. */
    detailLoadFailed: 'Failed to load asset',
    formRequiredFields: 'Ticker, Name and Currency are required',
    updated: 'Asset updated',
    metaRefreshed: 'Fields updated from Yahoo',
    metaRefreshFailed: 'Update failed',
    backfillDone: 'Price history updated',
    backfillFailed: 'Backfill failed',
    countriesPrefilledJustEtf: 'Countries prefilled from JustETF',
    countriesSectorsPrefilledMorningstar: 'Countries and sectors prefilled from Morningstar',
    regionsPrefilledMorningstar: 'Regions prefilled from Morningstar',
    sectorsPrefilledJustEtf: 'Sector distribution prefilled from JustETF',
    sectorsPrefilledYahoo: 'Sector distribution prefilled from Yahoo',
    sectorsPrefilledMorningstar: 'Sector distribution prefilled from Morningstar',
    yahooNoResponse: 'Yahoo did not respond',
    prefillFailed: 'Prefill failed',
    downloadFailed: 'Download failed',
    noWeightedCountries: 'No countries with a weight: add countries first',
    regionsRecomputed: 'Regions recomputed from countries',
    computeFailed: 'Compute failed',
    geoSaved: 'Geographic distribution saved',
    sectorsSaved: 'Sector distribution saved',
    countriesSaved: 'Country distribution saved',
  },
}

/** Canonical dictionary shape derived from the English source of truth. */
export type Dictionary = typeof en
