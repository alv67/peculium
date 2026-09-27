<script lang="ts">
  import type { Snippet } from 'svelte'
  import { onMount } from 'svelte'
  import { afterNavigate, goto } from '$app/navigation'
  import { resolve } from '$app/paths'
  import { page } from '$app/state'
  import { toast } from '$lib/stores/toast.svelte'
  import { priceRefresh } from '$lib/stores/priceRefresh.svelte'
  import { t } from '$lib/i18n/index.svelte'
  import {
    portfolioApi,
    transactionApi,
    assetApi,
    type Portfolio,
    type PortfolioSummary,
    type PortfolioHistory,
    type DashboardPerformance,
    type Transaction,
    type Asset,
    type PortfolioClassAllocation,
    type PortfolioGeographyAllocation,
    type PortfolioSectorAllocation,
  } from '$lib/services/api'
  import { formatCurrency } from '$lib/format'
  import { ArrowLeft, Download, MoreHorizontal, Plus, Trash2, Upload } from 'lucide-svelte'
  import AddTransactionModal from '$lib/components/domain/AddTransactionModal.svelte'
  import ImportPortfolioModal from '$lib/components/domain/ImportPortfolioModal.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte'
  import PnlValue from '$lib/components/ui/PnlValue.svelte'
  import Tabs from '$lib/components/ui/Tabs.svelte'
  import { setPortfolioPage, type PortfolioPageContext } from './context'
  import {
    parseTransactionFilters,
    transactionFiltersSignature,
    applyTransactionFilters,
    type TransactionFilters,
  } from './tx-filters'

  /**
   * Portfolio shell (EPIC K.4a, spec §4.2.2/§6.2): the single long portfolio
   * page split into a sticky header + four deep-linkable nested-route tabs
   * (Overview `/`, Positions `/positions`, Activity `/activity`,
   * Allocation `/allocation`). The header carries the identity row (back
   * link, name + currency, `[+ Transaction]` and the `⋯` actions menu:
   * export / import / delete — portfolio actions live in the menu, not in a
   * fifth tab, per the §6.2 wireframe), the KPI/hero strip in the portfolio
   * currency (same composition as the vault hero zone A, K.3a), and the
   * route-linked `ui/Tabs` bar; the tabs swap below via `{@render
   * children()}` while this layout — and all of its data — stays mounted.
   *
   * Data ownership: every fetch that the old page performed is performed
   * here unchanged and handed to the tab pages through the typed context in
   * `./context.ts` — no tab re-fetches anything on its own.
   */
  let { children }: { children: Snippet } = $props()

  const id = $derived(page.params.id as string | undefined)

  let portfolio = $state<Portfolio | null>(null)
  let summary = $state<PortfolioSummary | null>(null)
  let transactions = $state<Transaction[] | null>(null)
  let assets = $state<Asset[] | null>(null)
  let history = $state<PortfolioHistory | null>(null)
  let classAlloc = $state<PortfolioClassAllocation | null>(null)
  let classAllocError = $state(false)
  let geoAlloc = $state<PortfolioGeographyAllocation | null>(null)
  let sectorAlloc = $state<PortfolioSectorAllocation | null>(null)
  let geoAllocError = $state(false)
  let sectorAllocError = $state(false)
  let showTx = $state(false)
  let editingTx = $state<Transaction | null>(null)

  // EPIC I.9 (#88): the Transactions table is paginated through the
  // `TransactionPage` envelope of `GET /portfolios/{id}/transactions`
  // (order date desc). `txPage` is the 1-based page index, `txOffset` the
  // row window start; the page size mirrors the backend default (20). The
  // window now survives tab switches too (the layout never unmounts between
  // tabs), so Activity reopens exactly where it was left.
  // Every fetch below additionally applies the K.4c URL-persisted filters
  // (see `txFilters`), so the pagination total is the FILTERED count.
  const TX_PAGE_SIZE = 20
  let txPage = $state(1)
  let txLimit = $state(TX_PAGE_SIZE)
  let txOffset = $state(0)
  let txTotal = $state(0)
  let txLoading = $state(false)

  // EPIC K.4c (spec §6.2/§6.4/§8.2): the Activity filters — transaction
  // type, asset and date range — live in the tab's URL query
  // (`?type=sell&asset=<id>&from=YYYY-MM-DD&to=YYYY-MM-DD`) so a filtered
  // window is shareable, survives reloads and deep links
  // (`/portfolios/7/activity?type=sell` loads filtered from the start) and
  // back/forward restores exactly the previous view. The URL is their ONLY
  // storage: this derived reads it back on every change (including plain
  // history navigations, which bypass `setTxFilters`), and every
  // transaction fetch below funnels through it.
  const txFilters = $derived(parseTransactionFilters(page.url.searchParams))

  /** `txFilters` shaped as `transactionApi.list` params for the given row
   * window (unset dimensions stay `undefined` and the client omits them).
   * Read at CALL time, so the post-mutation refetch always honours the
   * filters still in the URL. */
  function txListQuery(offset: number) {
    return {
      limit: txLimit,
      offset,
      type: txFilters.type,
      asset_id: txFilters.assetId,
      from: txFilters.from,
      to: txFilters.to,
    }
  }

  /** Persist a new filter set into the URL (replaceState: chip clicks are
   * refinements of the current view, not history landmarks — the back
   * button therefore leaves the Activity tab instead of stepping through
   * every chip). The watcher below performs the actual reset + refetch, so
   * the flow stays identical for back/forward navigations too. */
  function setTxFilters(next: TransactionFilters): void {
    const url = new URL(page.url)
    applyTransactionFilters(url, next)
    if (url.search === page.url.search) return
    // The URL is cloned from the CURRENT `page.url` (already resolved by
    // SvelteKit; only the query is rewritten), same convention as the
    // keyboard nav in `ui/Tabs`.
    // eslint-disable-next-line svelte/no-navigation-without-resolve
    void goto(url, { replaceState: true, keepFocus: true, noScroll: true })
  }

  // Changing any filter resets the window to the first page of the filtered
  // result and refetches it — the only fetches a filter change invalidates
  // (the KPIs/allocations/performance don't depend on the filter).
  // `appliedFiltersSig` is a plain (non-reactive) baseline seeded from the
  // mount-time URL: the initial window already comes back filtered from
  // `load()`, and only a REAL change from that baseline re-queries.
  let appliedFiltersSig = transactionFiltersSignature(
    parseTransactionFilters(page.url.searchParams),
  )
  $effect(() => {
    const sig = transactionFiltersSignature(txFilters)
    if (sig === appliedFiltersSig) return
    appliedFiltersSig = sig
    txPage = 1
    txOffset = 0
    void loadTransactions()
  })

  // Same "1–20 of 137" range label and footer layout as the health page.
  const txRangeLabel = $derived(
    (transactions?.length ?? 0) === 0
      ? t('common.rangeEmpty', { total: txTotal })
      : t('common.rangeLabel', {
          from: txOffset + 1,
          to: txOffset + (transactions?.length ?? 0),
          total: txTotal,
        }),
  )

  // Monotonic request id (same guard as the performance card): rapid page
  // flips must never let a stale response overwrite the current window.
  let txReq = 0

  /** Fetch the current transaction page (window + K.4c URL filters) into
   * `transactions`/`txTotal`, touching nothing else on the page. If the
   * window comes back empty while rows still exist (the last row of the
   * last page was just deleted), step back to the previous page — clamped
   * against the fresh total — and refetch it within the same call. */
  async function loadTransactions(): Promise<void> {
    if (!id) return
    const req = ++txReq
    txLoading = true
    try {
      let res = await transactionApi.list(id, txListQuery(txOffset))
      if (req === txReq && res.transactions.length === 0 && res.total > 0 && txOffset > 0) {
        const maxPage = Math.max(1, Math.ceil(res.total / txLimit))
        txPage = Math.min(Math.max(1, txPage - 1), maxPage)
        txOffset = (txPage - 1) * txLimit
        res = await transactionApi.list(id, txListQuery(txOffset))
      }
      if (req === txReq) {
        transactions = res.transactions
        txTotal = res.total
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('activity.loadFailed')
      if (req === txReq) toast.error(message)
    } finally {
      if (req === txReq) txLoading = false
    }
  }

  /** Jump to a 1-based page (clamped to the last page of `txTotal`) and
   * refetch only the transactions window — never the whole portfolio page. */
  function gotoTxPage(target: number): void {
    const maxPage = Math.max(1, Math.ceil(txTotal / txLimit))
    txPage = Math.min(Math.max(1, target), maxPage)
    txOffset = (txPage - 1) * txLimit
    void loadTransactions()
  }

  const currency = $derived(portfolio?.currency || 'USD')

  // EPIC I.8 (#87): the portfolio's own percentage performance card — same
  // bucket model as the dashboard "Performance" card (EPIC I.3) but in the
  // PORTFOLIO currency, fed by `performanceBuckets(id, granularity)`
  // (`GET /portfolios/{id}/performance/buckets`). Isolated like the
  // allocations: a failed fetch just shows the chart's "No data" empty
  // state, and a `Spinner` covers every load. The card lives on the
  // Overview tab; the fetch stays here because the session price refresh
  // and every transaction mutation refetch the buckets too (E.9).
  let perf = $state<DashboardPerformance | null>(null)
  let perfLoading = $state(true)
  let granularity = $state<'month' | 'year'>('month')

  // SegmentedControl binds a plain string; the accessors keep the union type.
  function getGranularity(): string {
    return granularity
  }
  function setGranularity(value: string): void {
    if (value === 'month' || value === 'year') granularity = value
  }

  // Monotonic request id: when the toggle is flipped quickly, only the last
  // issued request may write the state (same guard as the dashboard).
  let perfReq = 0
  async function loadPerformance(g: 'month' | 'year'): Promise<void> {
    if (!id) return
    const req = ++perfReq
    perfLoading = true
    try {
      const res = await portfolioApi.performanceBuckets(id, g)
      if (req === perfReq) perf = res
    } catch {
      if (req === perfReq) perf = null
    } finally {
      if (req === perfReq) perfLoading = false
    }
  }

  // Runs once on mount with the default granularity, refetches on every
  // Monthly/Annual toggle change (and on portfolio navigation, since `id`
  // is read reactively inside `loadPerformance`).
  $effect(() => {
    void loadPerformance(granularity)
  })

  $effect(() => {
    if (!showTx) editingTx = null
  })

  onMount(load)

  async function load(): Promise<void> {
    if (!id) return
    try {
      // The transactions window rides `loadTransactions()` so the deep
      // linked K.4c filters apply to the FIRST fetch too and the guarded
      // write there stays the single owner of `transactions`/`txTotal`.
      const [p, s, , a] = await Promise.all([
        portfolioApi.get(id),
        portfolioApi.summary(id),
        loadTransactions(),
        assetApi.list(),
      ])
      portfolio = p
      summary = s
      assets = a
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('portfolio.detailLoadFailed')
      toast.error(message)
    }

    await loadAllocations()

    try {
      history = await portfolioApi.history(id)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('portfolio.historyLoadFailed')
      toast.error(message)
    }
  }

  // Refetch the price-derived data when the (globally triggered) session
  // refresh completes: the POST cleared the GET cache, so the summary comes
  // back with the fresh prices, and new prices can move the TWR buckets too
  // (same refetch pair the dashboard runs — see its `revision` watcher).
  async function refetchAfterRefresh(): Promise<void> {
    if (!id) return
    try {
      summary = await portfolioApi.summary(id)
    } catch {
      // Keep current data.
    }
    void loadPerformance(granularity)
  }

  // Plain (non-`$state`) baseline seeded at component init: only refresh
  // completions that happen while this layout is mounted refetch.
  let seenRefresh = priceRefresh.revision
  $effect(() => {
    const rev = priceRefresh.revision
    if (rev === seenRefresh) return
    seenRefresh = rev
    void refetchAfterRefresh()
  })

  async function loadAllocations(): Promise<void> {
    if (!id) return
    classAllocError = false
    geoAllocError = false
    sectorAllocError = false

    // L'allocazione per classi è isolata: se il backend non la espone ancora
    // (es. asset_class non popolati) non blocca il resto della pagina.
    try {
      classAlloc = await portfolioApi.classAllocation(id)
    } catch {
      classAllocError = true
    }

    // Anche geografia e settore sono isolati: un errore qui (endpoint non
    // disponibile o dati mancanti) non deve bloccare il resto della pagina.
    try {
      geoAlloc = await portfolioApi.geographyAllocation(id)
    } catch {
      geoAllocError = true
    }

    try {
      sectorAlloc = await portfolioApi.sectorAllocation(id)
    } catch {
      sectorAllocError = true
    }
  }

  async function reloadAfterMutation(): Promise<void> {
    if (!id) return
    try {
      // Refetch the CURRENT transactions page (plus total, with the
      // empty-page step-back) in parallel with the summary/history.
      // `loadTransactions` never rejects (it toasts its own errors), so a
      // transactions failure cannot block the other refreshes. The
      // mutation already cleared the GET cache.
      const [s, h] = await Promise.all([
        portfolioApi.summary(id),
        portfolioApi.history(id),
        loadTransactions(),
      ])
      summary = s
      history = h
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('portfolio.refreshFailed')
      toast.error(message)
    }
    // New/edited transactions change the flows behind the TWR buckets too
    // (the mutation already cleared the GET cache) — refresh the card.
    void loadPerformance(granularity)
    await loadAllocations()
  }

  // --- Header actions ------------------------------------------------------

  /** Open the modal fresh (no stale edit) — the `[+ Transaction]` CTA and
   * the Activity tab's row-edit button both funnel through here. */
  function openAddTransaction(): void {
    editingTx = null
    showTx = true
  }

  function editTransaction(tx: Transaction): void {
    editingTx = tx
    showTx = true
  }

  async function exportPortfolio(): Promise<void> {
    if (!id) return
    try {
      const doc = await portfolioApi.exportDoc(id)
      const blob = new Blob([JSON.stringify(doc, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `peculium-${doc.portfolio.name.toLowerCase().replace(/\s+/g, '-') || 'portfolio'}.json`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('portfolio.exportFailed')
      toast.error(message)
    }
  }

  // Import lives in the `⋯` menu too (K.4a). The shared modal keeps both
  // modes: overwrite THIS portfolio (the only target it is given — the
  // all-portfolios picker stays on the `/portfolios` list page) or register
  // the document as a new portfolio. Success reloads the whole shell, from
  // the first transactions page (an overwrite can change everything the
  // tabs show).
  let showImport = $state(false)

  async function reloadAfterImport(): Promise<void> {
    txPage = 1
    txOffset = 0
    await load()
    void loadPerformance(granularity)
  }

  // Delete mirrors the list page: confirm dialog → API → toast → leave the
  // (now gone) portfolio. `goto` is deliberately not awaited so this
  // handler never races the dialog's own close-then-unmount sequence.
  let showDeleteDialog = $state(false)
  let deleting = $state(false)

  function requestDelete(): void {
    menuOpen = false
    showDeleteDialog = true
  }

  async function deletePortfolio(): Promise<void> {
    if (!id) return
    deleting = true
    try {
      await portfolioApi.delete(id)
      toast.success(t('portfolio.deleted'))
      void goto(resolve('/portfolios'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.deleteFailed')
      toast.error(message)
    } finally {
      deleting = false
    }
  }

  // `⋯` popup: same close contract as the shell's UserMenu (Escape back to
  // the trigger / outside pointerdown / route change — tab clicks navigate,
  // so the strip swapping tabs also dismisses the menu).
  let menuOpen = $state(false)
  let menuRoot = $state<HTMLDivElement | null>(null)
  let menuTrigger = $state<HTMLButtonElement | null>(null)

  function handleMenuKeydown(event: KeyboardEvent): void {
    if (!menuOpen) return
    if (event.key === 'Escape') {
      menuOpen = false
      menuTrigger?.focus()
    }
  }

  afterNavigate(() => (menuOpen = false))

  $effect(() => {
    if (!menuOpen) return
    function handlePointerdown(event: PointerEvent): void {
      if (menuRoot && event.target instanceof Node && !menuRoot.contains(event.target)) {
        menuOpen = false
      }
    }
    window.addEventListener('pointerdown', handlePointerdown)
    return () => window.removeEventListener('pointerdown', handlePointerdown)
  })

  // --- Tabs (spec §4.2.2: nested routes, real URLs) ------------------------
  // hrefs resolved per the repo convention; `ui/Tabs` derives the active
  // tab from the URL, so deep links and the back button just work. The
  // portfolio-level actions (export/import/delete) stay in the `⋯` menu of
  // the header rather than becoming a fifth tab (§6.2 puts them in the
  // ".." slot next to `[+ Transaction]`).
  const tabItems = $derived(
    id
      ? [
          { href: resolve(`/portfolios/${id}`), label: t('portfolio.tabOverview') },
          { href: resolve(`/portfolios/${id}/positions`), label: t('portfolio.tabPositions') },
          { href: resolve(`/portfolios/${id}/activity`), label: t('portfolio.tabActivity') },
          { href: resolve(`/portfolios/${id}/allocation`), label: t('portfolio.tabAllocation') },
        ]
      : [],
  )

  // --- Context for the tab pages (see ./context.ts) -------------------------
  setPortfolioPage({
    get id() {
      return id
    },
    get portfolio() {
      return portfolio
    },
    get summary() {
      return summary
    },
    get currency() {
      return currency
    },
    get perf() {
      return perf
    },
    get perfLoading() {
      return perfLoading
    },
    get granularity() {
      return granularity
    },
    getGranularity,
    setGranularity,
    get history() {
      return history
    },
    get transactions() {
      return transactions
    },
    get txPage() {
      return txPage
    },
    get txLimit() {
      return txLimit
    },
    get txOffset() {
      return txOffset
    },
    get txTotal() {
      return txTotal
    },
    get txLoading() {
      return txLoading
    },
    get txRangeLabel() {
      return txRangeLabel
    },
    gotoTxPage,
    get txFilters() {
      return txFilters
    },
    setTxFilters,
    openAddTransaction,
    editTransaction,
    get classAlloc() {
      return classAlloc
    },
    get geoAlloc() {
      return geoAlloc
    },
    get sectorAlloc() {
      return sectorAlloc
    },
    get classAllocError() {
      return classAllocError
    },
    get geoAllocError() {
      return geoAllocError
    },
    get sectorAllocError() {
      return sectorAllocError
    },
  } satisfies PortfolioPageContext)
</script>

<svelte:window onkeydown={handleMenuKeydown} />

<div class="p-4 lg:p-6">
  <!-- Sticky entity header (spec §5.1/§6.2): `top` follows the live
       --app-header-h published by AppShell, so it stays flush while the app
       header condenses 56px → 44px (no gap strip); the same transition makes
       the two move in lockstep. The negative margins + padding are the
       responsive mirror of the wrapper padding (= `<main>`'s), so the opaque
       `bg-background` band spans the full content width at every size. -->
  <header
    class="sticky top-[var(--app-header-h)] z-10 -mx-4 -mt-4 mb-6 flex flex-col gap-3 bg-background px-4 pb-3 pt-4 transition-[top] duration-base ease-standard lg:-mx-6 lg:-mt-6 lg:px-6 lg:pt-6"
  >
    <div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
      <div class="min-w-0">
        <a
          href={resolve('/portfolios')}
          class="focus-ring -ml-1 inline-flex items-center gap-1 rounded-control text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft class="h-4 w-4 shrink-0" aria-hidden="true" />
          {t('portfolio.back')}
        </a>
        <h1 class="mt-1 text-2xl font-bold">
          {portfolio?.name ?? t('portfolio.fallbackName')}
          {#if portfolio}
            <span class="text-sm font-medium text-muted-foreground">({portfolio.currency})</span>
          {/if}
        </h1>
        {#if portfolio?.description}
          <p class="text-sm text-muted-foreground">{portfolio.description}</p>
        {/if}
      </div>
      <!-- `ml-auto` keeps the actions right-aligned when the wrap puts them
           on their own line (`justify-between` alone would push the lone
           trigger to the left, #117). -->
      <div class="ml-auto flex shrink-0 items-center gap-2">
        <Button onclick={openAddTransaction}>
          <Plus class="h-4 w-4" />
          {t('portfolio.addTransaction')}
        </Button>
        <div class="relative" bind:this={menuRoot}>
          <button
            bind:this={menuTrigger}
            type="button"
            aria-label={t('portfolio.actionsMenu')}
            aria-haspopup="true"
            aria-expanded={menuOpen}
            onclick={() => (menuOpen = !menuOpen)}
            class="focus-ring inline-flex h-9 w-9 items-center justify-center rounded-control border border-input text-foreground transition-colors hover:bg-muted"
          >
            <MoreHorizontal class="h-4 w-4" aria-hidden="true" />
          </button>
          {#if menuOpen}
            <div
              class="absolute right-0 top-full z-20 mt-2 w-52 max-w-[calc(100vw-2rem)] rounded-card border border-border bg-surface p-1 shadow-raised"
            >
              <Button
                variant="ghost"
                class="w-full"
                onclick={() => {
                  menuOpen = false
                  void exportPortfolio()
                }}
              >
                <span class="flex w-full items-center gap-2">
                  <Download class="h-4 w-4 shrink-0" aria-hidden="true" />
                  {t('portfolio.export')}
                </span>
              </Button>
              <Button
                variant="ghost"
                class="w-full"
                onclick={() => {
                  menuOpen = false
                  showImport = true
                }}
              >
                <span class="flex w-full items-center gap-2">
                  <Upload class="h-4 w-4 shrink-0" aria-hidden="true" />
                  {t('portfolio.import')}
                </span>
              </Button>
              <Button
                variant="ghost"
                class="w-full"
                onclick={requestDelete}
              >
                <span class="flex w-full items-center gap-2 text-negative">
                  <Trash2 class="h-4 w-4 shrink-0" aria-hidden="true" />
                  {t('portfolio.delete')}
                </span>
              </Button>
            </div>
          {/if}
        </div>
      </div>
    </div>

    {#if summary}
      <!-- KPI/hero strip (spec §6.2 "value + P/L always visible"): the same
           zone-A composition the vault hero uses (K.3a) — headline value,
           signed P/L via `PnlValue` (never colour alone, D6) and muted
           secondary chips — in the PORTFOLIO currency. It stays mounted
           while tabs swap; the full Active/Closed roll-up remains on the
           Overview tab. -->
      <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <span class="text-xl font-bold tabular-nums">
          {formatCurrency(summary.active.value, currency)}
        </span>
        <PnlValue value={summary.active.gain_loss} kind="currency" {currency} />
        <PnlValue value={summary.active.gain_loss_pct} kind="percent" />
        <span class="text-xs text-muted-foreground">{t('hero.allTime')}</span>
      </div>
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span class="inline-flex items-center gap-1.5">
          {t('hero.invested')}
          <span class="tabular-nums text-foreground">
            {formatCurrency(summary.active.invested, currency)}
          </span>
        </span>
        <span class="inline-flex items-center gap-1.5">
          {t('hero.realized')}
          <PnlValue value={summary.closed.realized} kind="currency" {currency} size="sm" />
        </span>
        <span class="inline-flex items-center gap-1.5">
          {t('hero.dividends')}
          <span class="tabular-nums text-foreground">
            {formatCurrency(summary.active.dividends, currency)}
          </span>
        </span>
      </div>
    {/if}

    <Tabs items={tabItems} ariaLabel={t('portfolio.tabsLabel')} />
  </header>

  {@render children()}
</div>

<AddTransactionModal
  bind:open={showTx}
  portfolioId={id ?? ''}
  assets={assets ?? []}
  {currency}
  editing={editingTx}
  onsuccess={reloadAfterMutation}
/>

<ImportPortfolioModal
  bind:open={showImport}
  portfolios={portfolio ? [portfolio] : []}
  onsuccess={reloadAfterImport}
/>

<ConfirmDialog
  bind:open={showDeleteDialog}
  variant="danger"
  title={t('portfolio.delete')}
  message={t('portfolio.deleteConfirm')}
  confirmLabel={t('common.delete')}
  cancelLabel={t('common.cancel')}
  loading={deleting}
  onconfirm={deletePortfolio}
/>
