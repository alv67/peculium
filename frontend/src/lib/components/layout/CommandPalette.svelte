<script lang="ts">
  import {
    Activity,
    Banknote,
    Briefcase,
    CircleDollarSign,
    Eye,
    FolderClosed,
    HandCoins,
    KeyRound,
    LayoutDashboard,
    Monitor,
    Moon,
    PanelLeft,
    RefreshCw,
    Search,
    Settings,
    SlidersHorizontal,
    Sun,
    User,
  } from 'lucide-svelte'
  import { fade } from 'svelte/transition'
  import { afterNavigate, goto } from '$app/navigation'
  import { resolve } from '$app/paths'
  import { t, type MessageKey } from '$lib/i18n/index.svelte'
  import {
    assetApi,
    portfolioApi,
    type Asset,
    type AssetLookupResult,
    type Portfolio,
  } from '$lib/services/api'
  import { palette, setCvd } from '$lib/stores/palette.svelte'
  import { refreshPrices } from '$lib/stores/priceRefresh.svelte'
  import { setThemeMode, theme, type ThemeMode } from '$lib/stores/theme.svelte'
  import { viewport } from '$lib/stores/viewport.svelte'
  import { focusTrap } from '../ui/focus-trap'
  import { backdropFade } from '../ui/transitions'
  import { cx } from '../ui/utils'

  /**
   * Global command palette (EPIC K.5a, spec §8.1): ⌘K / Ctrl+K opens a
   * keyboard-first dialog to navigate and act without leaving the current
   * context. Mounted once in `AppShell.svelte` (inside the auth gate, so the
   * global chord never fires on Login); the window-level keydown handler and
   * the `<svelte:window>` host live here, while `open` is `$bindable` so the
   * shell (and the `AppHeader` search trigger) can drive it too.
   *
   * Sections, rendered in this order and only when non-empty:
   * - *Go to*: the static destinations (Overview, Portfolios, Assets,
   *   Data & Sync, Settings and its four sub-sections) plus every portfolio
   *   from `portfolioApi.list()`;
   * - *Assets*: registered assets from `assetApi.list()`, plus a live
   *   "Search Yahoo for …" row fed by a debounced `assetApi.lookup()` —
   *   selecting it navigates to `/assets` (creating assets stays out of
   *   the palette's scope);
   * - *Actions*: Add transaction (the same single-portfolio shortcut the
   *   K.2 `Fab` uses), Refresh prices (the shared `priceRefresh` store path,
   *   Toggle theme (cycles light → dark → system on the theme store),
   *   Toggle CVD palette (palette store) and Toggle sidebar (only when a
   *   sidebar exists to toggle, i.e. desktop — the callback lives in the
   *   shell, which owns the persisted state).
   *
   * Data is fetched lazily, only while the dialog is open: the first open
   * populates both lists, every later open re-fires them for free thanks to
   * the 60 s GET cache in `services/api.ts` (any mutation clears it, so the
   * lists self-refresh after edits without a dedicated invalidation hook).
   * Failures stay silent — the static sections keep the palette useful.
   *
   * Matching is a ~15-line local matcher (prefix > substring > subsequence,
   * case-insensitive, no fuzzy library) over a per-item haystack
   * (label + hint + aliases). Keyboard model: ↑/↓ (wrapping), Home/End move
   * the active option through `aria-activedescendant`, Enter runs it, Esc
   * closes — the input is the single Tab stop and the shared K.1c
   * `focusTrap` handles capture/restore, so focus always returns to the
   * trigger and never strands. ARIA is the APG *combobox with a grouped
   * listbox*: dialog → input `role="combobox"` → `role="listbox"` of
   * `role="group"` sections of `role="option"` rows. Motion sticks to the
   * `prefers-reduced-motion`-aware backdrop fade (no panel animation).
   */
  type IconType = typeof RefreshCw

  /** Below this many characters the Yahoo row never appears (and never fires). */
  const LOOKUP_MIN_CHARS = 2

  /** Keystroke-settling window before the Yahoo lookup fires (ms). */
  const LOOKUP_DEBOUNCE_MS = 300

  /** Cap per dynamic section (portfolios / assets) while the query is empty. */
  const MAX_ITEMS_WHEN_EMPTY = 8

  /** Global DOM guard for pathological queries. */
  const MAX_MATCHES = 50

  /** Theme cycle order for the *Toggle theme* action (matches ThemeToggle). */
  const THEME_CYCLE: ThemeMode[] = ['light', 'dark', 'system']

  interface PaletteItem {
    id: string
    section: 'go' | 'assets' | 'actions'
    label: string
    /** Muted right-aligned supplement (ticker, currency, target state…). */
    hint?: string
    icon: IconType
    /** Extra search aliases beyond the visible copy (English words). */
    keywords?: string
    run: () => void | Promise<void>
    /** Lower-cased haystack (`label hint keywords`) the matcher scores. */
    hay: string
  }

  let {
    open = $bindable(false),
    ontogglesidebar = undefined,
  }: {
    /** Bindable so the shell header trigger, the chord and Esc share one state. */
    open?: boolean
    /** Shell sidebar toggle; the *Toggle sidebar* row renders only with it. */
    ontogglesidebar?: (() => void) | undefined
  } = $props()

  const uid = $props.id()
  const listboxId = `${uid}-listbox`

  let query = $state('')
  let rawActive = $state(0)
  let portfolios = $state<Portfolio[]>([])
  let assets = $state<Asset[]>([])
  let lookup = $state<{ status: 'idle' | 'loading' | 'ready'; results: AssetLookupResult[] }>({
    status: 'idle',
    results: [],
  })
  let inputEl = $state<HTMLInputElement | null>(null)

  // Plain flags (never rendered): request de-duplication and race guards.
  let listsLoading = false
  let lookupSeq = 0

  const normalizedQuery = $derived(
    query.trim().toLowerCase().replace(/\s+/g, ' '),
  )

  /** Monotonic counter: a lookup response is applied only if it is still the latest. */
  function setLookup(next: { status: 'idle' | 'loading' | 'ready'; results: AssetLookupResult[] }): void {
    if (open) lookup = next
  }

  function fetchLists(): void {
    if (listsLoading) return
    listsLoading = true
    void Promise.all([portfolioApi.list(), assetApi.list()])
      .then(([pf, as]) => {
        // Defensive (issue #121): a list endpoint can resolve to JSON `null`
        // instead of `[]`; storing it would poison the `$state` and make the
        // `view` derived throw on every later flush (palette unopenable).
        portfolios = Array.isArray(pf) ? pf : []
        assets = Array.isArray(as) ? as : []
      })
      .catch(() => {
        // Silent: a transient failure keeps whatever was loaded before; the
        // static sections stay useful either way, and the next open retries.
      })
      .finally(() => {
        listsLoading = false
      })
  }

  /** Add Transaction shortcut, identical to the K.2 `Fab` action (spec §6.4). */
  async function addTransaction(): Promise<void> {
    try {
      const data = await portfolioApi.list()
      // Coerce like `fetchLists` (issue #121): a `null` list must fall back to
      // the portfolios page, not throw through the array access below.
      const rows = Array.isArray(data) ? data : []
      await goto(
        rows.length === 1 ? resolve(`/portfolios/${rows[0].id}`) : resolve('/portfolios'),
      )
    } catch {
      await goto(resolve('/portfolios'))
    }
  }

  /** Manual session refresh through the shared store path (same semantics
   * as the Fab): toasts + global stamp/strip, pages refetch on `revision`. */
  function runRefreshPrices(): void {
    void refreshPrices({ announceSuccess: true })
  }

  /** Score of a haystack vs the normalized query; `null` = no match. */
  function matchScore(hay: string, q: string): number | null {
    if (hay.startsWith(q)) return 0
    if (hay.includes(q)) return 1
    let i = 0
    for (const ch of hay) {
      if (ch === q[i]) i += 1
      if (i === q.length) return 2
    }
    return null
  }

  const nextThemeMode = $derived(
    THEME_CYCLE[(THEME_CYCLE.indexOf(theme.mode) + 1) % THEME_CYCLE.length],
  )

  const view = $derived.by(() => {
    const q = normalizedQuery
    const rawQuery = query.trim()
    const empty = q === ''

    const row = (partial: Omit<PaletteItem, 'hay'>): PaletteItem => ({
      ...partial,
      hay: `${partial.label} ${partial.hint ?? ''} ${partial.keywords ?? ''}`
        .toLowerCase()
        .replace(/\s+/g, ' ')
        .trim(),
    })

    // While unfiltered the dynamic sections are capped (a 200-asset vault
    // should not turn the palette into a scroll wall); under a query every
    // match counts, ranked by score.
    const portfolioSource = empty ? portfolios.slice(0, MAX_ITEMS_WHEN_EMPTY) : portfolios
    const assetSource = empty ? assets.slice(0, MAX_ITEMS_WHEN_EMPTY) : assets

    const goItems: PaletteItem[] = [
      row({
        id: 'go-overview',
        section: 'go',
        label: t('nav.overview'),
        icon: LayoutDashboard,
        keywords: 'dashboard home patrimonio wealth',
        run: () => goto(resolve('/')),
      }),
      row({
        id: 'go-portfolios',
        section: 'go',
        label: t('nav.portfolios'),
        icon: Briefcase,
        keywords: 'portfolio',
        run: () => goto(resolve('/portfolios')),
      }),
      row({
        id: 'go-assets',
        section: 'go',
        label: t('nav.assets'),
        icon: Banknote,
        keywords: 'asset securities instruments',
        run: () => goto(resolve('/assets')),
      }),
      row({
        id: 'go-data-sync',
        section: 'go',
        label: t('nav.dataSync'),
        icon: Activity,
        keywords: 'admin health sync prices',
        run: () => goto(resolve('/admin/health')),
      }),
      row({
        id: 'go-settings',
        section: 'go',
        label: t('nav.settings'),
        icon: Settings,
        keywords: 'preferences profile',
        run: () => goto(resolve('/settings')),
      }),
      row({
        id: 'go-settings-profile',
        section: 'go',
        label: t('settingsTabs.profile'),
        hint: t('nav.settings'),
        icon: User,
        run: () => goto(resolve('/settings')),
      }),
      row({
        id: 'go-settings-password',
        section: 'go',
        label: t('settingsTabs.password'),
        hint: t('nav.settings'),
        icon: KeyRound,
        keywords: 'security',
        run: () => goto(resolve('/settings/password')),
      }),
      row({
        id: 'go-settings-preferences',
        section: 'go',
        label: t('settingsTabs.preferences'),
        hint: t('nav.settings'),
        icon: SlidersHorizontal,
        keywords: 'theme language palette',
        run: () => goto(resolve('/settings/preferences')),
      }),
      row({
        id: 'go-settings-currencies',
        section: 'go',
        label: t('settingsTabs.currencies'),
        hint: t('nav.settings'),
        icon: CircleDollarSign,
        keywords: 'fx exchange rates',
        run: () => goto(resolve('/settings/currencies')),
      }),
      ...portfolioSource.map((p) =>
        row({
          id: `go-portfolio-${p.id}`,
          section: 'go',
          label: p.name,
          hint: p.currency,
          icon: FolderClosed,
          run: () => goto(resolve(`/portfolios/${p.id}`)),
        }),
      ),
    ]

    const assetItems: PaletteItem[] = assetSource.map((a) =>
      row({
        id: `asset-${a.id}`,
        section: 'assets',
        label: a.name || a.ticker,
        hint: a.ticker,
        icon: Banknote,
        keywords: `${a.ticker} ${a.type} ${a.asset_class}`,
        run: () => goto(resolve(`/assets/${a.id}`)),
      }),
    )

    // Live Yahoo affordance: never filtered locally (it mirrors the raw
    // query); the debounced effect below feeds `lookup` for its hint.
    // Selection navigates to the assets page — creation stays out of scope.
    if (rawQuery.length >= LOOKUP_MIN_CHARS) {
      const top = lookup.results[0]
      assetItems.push(
        row({
          id: 'assets-yahoo',
          section: 'assets',
          label: t('commandPalette.yahooRow', { query: rawQuery }),
          hint:
            lookup.status === 'loading'
              ? t('commandPalette.searching')
              : top
                ? `${top.ticker} · ${top.name}`
                : undefined,
          icon: Search,
          keywords: 'yahoo symbol search',
          run: () => goto(resolve('/assets')),
        }),
      )
    }

    const actionItems: PaletteItem[] = [
      row({
        id: 'act-add-transaction',
        section: 'actions',
        label: t('quickActions.addTransaction'),
        hint: t('quickActions.addTransactionHint'),
        icon: HandCoins,
        keywords: 'tx buy sell dividend',
        run: addTransaction,
      }),
      row({
        id: 'act-refresh-prices',
        section: 'actions',
        label: t('quickActions.refreshPrices'),
        hint: t('quickActions.refreshPricesHint'),
        icon: RefreshCw,
        keywords: 'quote yahoo sync',
        run: runRefreshPrices,
      }),
      row({
        id: 'act-toggle-theme',
        section: 'actions',
        label: t('commandPalette.toggleTheme'),
        // The hint previews the cycle target, so the row always says what it does.
        hint: t(`theme.${nextThemeMode}`),
        icon: nextThemeMode === 'light' ? Sun : nextThemeMode === 'dark' ? Moon : Monitor,
        keywords: 'light dark system appearance',
        run: () => setThemeMode(nextThemeMode),
      }),
      row({
        id: 'act-toggle-cvd',
        section: 'actions',
        label: t('commandPalette.toggleCvd'),
        hint: t(palette.cvd ? 'preferences.paletteClassic' : 'preferences.paletteCvd'),
        icon: Eye,
        keywords: 'color blind accessibility green red blue orange',
        run: () => setCvd(!palette.cvd),
      }),
    ]
    // "Where applicable": the sidebar only exists from `sm` up, and the
    // tablet rail ignores the persisted flag — offer it on desktop only.
    if (ontogglesidebar && viewport.isDesktop) {
      actionItems.push(
        row({
          id: 'act-toggle-sidebar',
          section: 'actions',
          label: t('commandPalette.toggleSidebar'),
          icon: PanelLeft,
          keywords: 'rail expand collapse',
          run: () => ontogglesidebar(),
        }),
      )
    }

    const filter = (items: PaletteItem[]): PaletteItem[] => {
      if (empty) return items
      return items
        .map((item) => ({ item, score: matchScore(item.hay, q) }))
        .filter((m): m is { item: PaletteItem; score: number } => m.score !== null)
        // Stable Array#sort keeps the defined order inside each score tier.
        .sort((a, b) => a.score - b.score)
        .slice(0, MAX_MATCHES)
        .map((m) => m.item)
    }

    let index = 0
    const sections = [
      { labelKey: 'commandPalette.sectionGoTo' as MessageKey, items: filter(goItems) },
      { labelKey: 'commandPalette.sectionAssets' as MessageKey, items: filter(assetItems) },
      { labelKey: 'commandPalette.sectionActions' as MessageKey, items: filter(actionItems) },
    ]
      .filter((section) => section.items.length > 0)
      .map((section) => ({
        labelKey: section.labelKey,
        rows: section.items.map((item) => ({ id: item.id, index: index++, item })),
      }))

    return { sections, count: index }
  })

  /** Roving active option (clamped, so a shrinking list can never strand it). */
  const activeIndex = $derived(
    view.count > 0 ? Math.min(Math.max(rawActive, 0), view.count - 1) : 0,
  )

  function runItem(item: PaletteItem): void {
    open = false
    try {
      void Promise.resolve(item.run()).catch(() => {
        // Palette already closed; the action's own error surfacing applies.
      })
    } catch {
      // Defensive: never trap the user in a half-run action.
    }
  }

  function handleKeydown(event: KeyboardEvent): void {
    // Enter while an IME composition is pending confirms the composition, not
    // the palette.
    if (event.isComposing) return
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault()
        if (view.count > 0) rawActive = (activeIndex + 1) % view.count
        break
      case 'ArrowUp':
        event.preventDefault()
        if (view.count > 0) rawActive = (activeIndex - 1 + view.count) % view.count
        break
      case 'Home':
        if (view.count > 0) {
          event.preventDefault()
          rawActive = 0
        }
        break
      case 'End':
        if (view.count > 0) {
          event.preventDefault()
          rawActive = view.count - 1
        }
        break
      case 'Enter': {
        const current = view.sections.flatMap((section) => section.rows)[activeIndex]
        if (current) {
          event.preventDefault()
          runItem(current.item)
        }
        break
      }
      // Escape closes on the overlay handler (it sees the bubbled key too).
      default:
        break
    }
  }

  function handleOverlayKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') open = false
  }

  function handleBackdropClick(event: MouseEvent): void {
    // Only clicks that land on the overlay itself (not bubbled from the panel).
    if (event.target === event.currentTarget) open = false
  }

  /** ⌘K / Ctrl+K from anywhere (the shell mounts this outside the dialog). */
  function handleGlobalKeydown(event: KeyboardEvent): void {
    if ((event.metaKey || event.ctrlKey) && !event.altKey && event.key.toLowerCase() === 'k') {
      event.preventDefault()
      open = !open
    }
  }

  // Open lifecycle: body scroll lock, lazy list fetch (the 60 s GET cache in
  // services/api.ts makes the every-open call free unless the data is stale),
  // focus into the input (after the focus trap grabbed the panel) and full
  // reset on close/unmount — the cleanup form also covers the shell
  // unmounting while open (logout).
  $effect(() => {
    if (!open) return
    document.body.style.overflow = 'hidden'
    void fetchLists()
    inputEl?.focus()
    return () => {
      document.body.style.overflow = ''
      query = ''
      rawActive = 0
      lookup = { status: 'idle', results: [] }
    }
  })

  // Debounced Yahoo lookup: fires only past LOOKUP_MIN_CHARS chars and only
  // after the keystrokes settle; the sequence counter drops stale responses
  // (and responses that arrived after the dialog closed).
  $effect(() => {
    const q = query.trim()
    if (!open || q.length < LOOKUP_MIN_CHARS) {
      lookup = { status: 'idle', results: [] }
      return
    }
    setLookup({ status: 'loading', results: [] })
    const seq = ++lookupSeq
    const timer = setTimeout(() => {
      assetApi
        .lookup(q)
        .then((results) => {
          // Same guard as `fetchLists` (issue #121): the derived below reads
          // `lookup.results[0]` during the reactive flush.
          if (seq === lookupSeq) {
            setLookup({ status: 'ready', results: Array.isArray(results) ? results : [] })
          }
        })
        .catch(() => {
          if (seq === lookupSeq) setLookup({ status: 'ready', results: [] })
        })
    }, LOOKUP_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  })

  // Any query edit restarts the tour of the list from the top row.
  $effect(() => {
    void query
    rawActive = 0
  })

  // Follow the active option on arrow/Home/End (no-op when already in view;
  // `block: 'nearest'` also skips the scroll under reduced motion).
  $effect(() => {
    const id = `${uid}-opt-${activeIndex}`
    if (!open) return
    document.getElementById(id)?.scrollIntoView({ block: 'nearest' })
  })

  // Back/forward (or any other navigation) while open: dismiss the dialog.
  afterNavigate(() => {
    if (open) open = false
  })
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

{#if open}
  <div
    class="fixed inset-0 z-40 flex items-center justify-center bg-overlay/50 p-4"
    role="dialog"
    aria-modal="true"
    aria-label={t('commandPalette.title')}
    tabindex="-1"
    transition:fade={backdropFade()}
    onclick={handleBackdropClick}
    onkeydown={handleOverlayKeydown}
  >
    <div
      use:focusTrap
      tabindex="-1"
      class="flex max-h-[70dvh] w-full max-w-lg flex-col overflow-hidden rounded-card bg-surface-raised shadow-raised outline-none"
    >
      <div class="flex items-center gap-2 border-b border-border px-4">
        <Search class="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden="true" />
        <input
          id={`${uid}-input`}
          bind:this={inputEl}
          bind:value={query}
          role="combobox"
          type="text"
          autocomplete="off"
          autocapitalize="none"
          spellcheck={false}
          aria-label={t('commandPalette.inputLabel')}
          aria-expanded={view.count > 0}
          aria-haspopup="listbox"
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={view.count > 0 ? `${uid}-opt-${activeIndex}` : undefined}
          placeholder={t('commandPalette.placeholder')}
          class="h-12 min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
          onkeydown={handleKeydown}
        />
      </div>

      <div
        id={listboxId}
        role="listbox"
        aria-labelledby={`${uid}-input`}
        class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-1.5"
      >
        {#if view.count === 0}
          <p class="px-3 py-8 text-center text-sm text-muted-foreground">
            {t('commandPalette.noResults')}
          </p>
        {:else}
          {#each view.sections as section (section.labelKey)}
            <div role="group" aria-labelledby={`${uid}-section-${section.labelKey}`}>
              <div
                id={`${uid}-section-${section.labelKey}`}
                class="px-3 pb-1 pt-2 text-micro font-semibold uppercase tracking-wide text-muted-foreground"
              >
                {t(section.labelKey)}
              </div>
              <!-- The list is the input's `aria-activedescendant` target: the
                   options are deliberately not focusable (APG grouped-listbox
                   pattern), so the click handler needs no sibling key handler —
                   the keyboard path is the input above — and a `tabindex` on
                   the rows would be actively wrong (Tab must not enter the
                   list). The a11y rules below cannot see that contract. -->
              <!-- eslint-disable svelte/a11y-click-events-have-key-events, svelte/a11y-no-noninteractive-element-interactions, svelte/a11y-interactive-supports-focus -->
              {#each section.rows as row (row.id)}
                {@const Icon = row.item.icon}
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_interactive_supports_focus -->
                <!-- eslint-disable-next-line svelte/no-unused-svelte-ignore -- the two suppressions above answer the Svelte *compiler* diagnostics, which the ESLint parser config does not reproduce -->
                <div
                  id={`${uid}-opt-${row.index}`}
                  role="option"
                  aria-selected={row.index === activeIndex}
                  class={cx(
                    'flex w-full cursor-pointer items-center gap-3 rounded-control px-3 py-2 text-left text-sm text-foreground',
                    row.index === activeIndex ? 'bg-accent/10' : 'hover:bg-muted',
                  )}
                  onmousedown={(event: MouseEvent) => event.preventDefault()}
                  onmousemove={() => (rawActive = row.index)}
                  onclick={() => runItem(row.item)}
                >
                  <Icon class="h-4 w-4 shrink-0 text-accent-text" />
                  <span class="min-w-0 flex-1 truncate">{row.item.label}</span>
                  {#if row.item.hint}
                    <span class="max-w-[40%] shrink-0 truncate text-xs text-muted-foreground">
                      {row.item.hint}
                    </span>
                  {/if}
                </div>
              {/each}
              <!-- eslint-enable svelte/a11y-click-events-have-key-events, svelte/a11y-no-noninteractive-element-interactions, svelte/a11y-interactive-supports-focus -->
            </div>
          {/each}
        {/if}
      </div>

      <!-- Key hints are a mouse-user affordance: hidden below `sm` where the
           palette is tapped, not keyboard-driven. -->
      <div
        class="hidden items-center gap-4 border-t border-border px-4 py-2 text-xs text-muted-foreground sm:flex"
      >
        <span class="flex items-center gap-1.5">
          <kbd class="rounded border border-border bg-muted px-1.5 font-mono text-micro">↑↓</kbd>
          {t('commandPalette.hintNavigate')}
        </span>
        <span class="flex items-center gap-1.5">
          <kbd class="rounded border border-border bg-muted px-1.5 font-mono text-micro">↵</kbd>
          {t('commandPalette.hintSelect')}
        </span>
        <span class="flex items-center gap-1.5">
          <kbd class="rounded border border-border bg-muted px-1.5 font-mono text-micro">Esc</kbd>
          {t('commandPalette.hintClose')}
        </span>
      </div>
    </div>
  </div>
{/if}
