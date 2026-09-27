<script lang="ts">
  import { onMount } from 'svelte'
  import { resolve } from '$app/paths'
  import {
    portfolioApi,
    type AllocationDrill,
    type AllocationDrillDim,
    type Dashboard,
    type DashboardAllocation,
    type DashboardPerformance,
    type PortfolioPerformanceSummary,
  } from '$lib/services/api'
  import { priceRefresh } from '$lib/stores/priceRefresh.svelte'
  import { applyDashboardStatus } from '$lib/stores/dashboardStatus.svelte'
  import { t } from '$lib/i18n/index.svelte'
  import AllocationDonut from '$lib/components/domain/AllocationDonut.svelte'
  import CapitalChart from '$lib/components/domain/CapitalChart.svelte'
  import FirstRunChecklist from '$lib/components/domain/FirstRunChecklist.svelte'
  import InvestmentsTable from '$lib/components/domain/InvestmentsTable.svelte'
  import PerformanceChart from '$lib/components/domain/PerformanceChart.svelte'
  import ScopeSwitcher from '$lib/components/domain/ScopeSwitcher.svelte'
  import Sparkline, { type SparklinePoint } from '$lib/components/domain/Sparkline.svelte'
  import ClassDonut from '$lib/components/domain/ClassDonut.svelte'
  import ExposureBarChart, { type ExposureBarRow } from '$lib/components/domain/ExposureBarChart.svelte'
  import AllocationDrillPanel from '$lib/components/domain/AllocationDrillPanel.svelte'
  import { countryDisplayName } from '$lib/countryNames'
  import { chartSemanticColors } from '$lib/chartPalette'
  import { resolved } from '$lib/stores/theme.svelte'
  import Card from '$lib/components/ui/Card.svelte'
  import PeriodChips from '$lib/components/ui/PeriodChips.svelte'
  import PnlValue from '$lib/components/ui/PnlValue.svelte'
  import SegmentedControl from '$lib/components/ui/SegmentedControl.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import EmptyState from '$lib/components/ui/EmptyState.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
  import { ChevronDown } from 'lucide-svelte'
  import { formatCurrency, formatPercent, assetClassLabel } from '$lib/format'
  import { pnlColorClass } from '$lib/ui-colors'

  let dash = $state<Dashboard | null>(null)
  let alloc = $state<DashboardAllocation | null>(null)
  let loading = $state(true)

  // Performance buckets (EPIC I.3): vault-wide percentage return and
  // invested-capital series in the base currency, monthly by default,
  // switchable to annual. One fetch feeds BOTH the hero value-vs-invested
  // chart (K.3a zone A) and the Performance card (zone B). Isolated like the
  // allocation card: a failed fetch just renders the charts' empty states.
  let perf = $state<DashboardPerformance | null>(null)
  let perfLoading = $state(true)
  let granularity = $state<'month' | 'year'>('month')
  const perfItems = $derived([
    { value: 'month', label: t('performance.monthly') },
    { value: 'year', label: t('performance.annual') },
  ])

  // SegmentedControl binds a plain string; the accessors keep the union type.
  function getGranularity(): string {
    return granularity
  }
  function setGranularity(value: string): void {
    if (value === 'month' || value === 'year') granularity = value
  }

  // Monotonic request id: when the toggle is flipped quickly, only the last
  // issued request may write the state.
  let perfReq = 0
  async function loadPerformance(g: 'month' | 'year'): Promise<void> {
    const req = ++perfReq
    perfLoading = true
    try {
      const res = await portfolioApi.dashboardPerformance(g)
      if (req === perfReq) perf = res
    } catch {
      if (req === perfReq) perf = null
    } finally {
      if (req === perfReq) perfLoading = false
    }
  }

  // Runs once on mount with the default granularity and refetches whenever
  // the Monthly/Annual toggle changes.
  $effect(() => {
    void loadPerformance(granularity)
  })

  // Hero chart ranges (EPIC K.3a, decision D10): BUCKET-DRIVEN — with monthly
  // buckets the chips window the existing series client-side to the last 12
  // (1Y) / 36 (3Y) buckets; with annual buckets only ALL is offered and the
  // chip row hides itself. Note this windows *buckets*, not calendar time:
  // the backend omits empty months, and true daily 1M/3M ranges need the
  // daily vault-series endpoint already logged as a K.3 fast-follow ask.
  type HeroPeriod = '1Y' | '3Y' | 'ALL'
  const HERO_PERIOD_KEY = 'peculium-hero-period'
  // Legacy storage key: when the new key is absent, the value stored here is
  // adopted and written forward once, so no saved preference is lost.
  const LEGACY_HERO_PERIOD_KEY = 'vaultlab-hero-period'
  function readStoredPeriod(): HeroPeriod {
    try {
      let v = localStorage.getItem(HERO_PERIOD_KEY)
      if (v === null) {
        // One-time migration: adopt the legacy value and write it forward.
        const legacy = localStorage.getItem(LEGACY_HERO_PERIOD_KEY)
        if (legacy !== null) {
          localStorage.setItem(HERO_PERIOD_KEY, legacy)
          v = legacy
        }
      }
      if (v === '1Y' || v === '3Y' || v === 'ALL') return v
    } catch {
      /* storage unavailable (private mode): default below */
    }
    return 'ALL'
  }
  // §8.2: the last-used period persists per scope; only the wealth scope
  // exists today, so one key. Storage errors must never break the page.
  let heroPeriod = $state<HeroPeriod>(readStoredPeriod())
  function selectHeroPeriod(value: string): void {
    if (value !== '1Y' && value !== '3Y' && value !== 'ALL') return
    heroPeriod = value
    try {
      localStorage.setItem(HERO_PERIOD_KEY, value)
    } catch {
      /* in-memory selection still applies for this visit */
    }
  }

  // Chips are derived from the granularity the buckets actually carry (the
  // payload, not the toggle) so they never offer a window the data can't fill.
  const heroPeriods = $derived<HeroPeriod[]>(
    perf?.granularity === 'month' ? ['1Y', '3Y', 'ALL'] : ['ALL'],
  )
  const heroPeriodChips = $derived(
    (
      [
        { value: '1Y', label: t('period.oneYear') },
        { value: '3Y', label: t('period.threeYears') },
        { value: 'ALL', label: t('period.all') },
      ] as { value: HeroPeriod; label: string }[]
    ).filter((p) => heroPeriods.includes(p.value)),
  )
  const heroBuckets = $derived.by(() => {
    const rows = perf?.buckets ?? []
    if (perf?.granularity !== 'month' || heroPeriod === 'ALL') return rows
    return rows.slice(Math.max(rows.length - (heroPeriod === '1Y' ? 12 : 36), 0))
  })

  onMount(async () => {
    try {
      dash = await portfolioApi.dashboard()
      // Feed the global quality strip counters (the shell seeds them on
      // mount; here they stay as fresh as the payload below).
      if (dash) applyDashboardStatus(dash)
    } catch {
      dash = null
    } finally {
      loading = false
    }

    // L'allocazione complessiva (classi, regioni, settori, paesi) è isolata:
    // se l'endpoint non è disponibile la card viene omessa senza bloccare il
    // resto della dashboard.
    try {
      alloc = await portfolioApi.dashboardAllocation()
    } catch {
      alloc = null
    }
  })

  // The once-per-session price refresh now lives in the shell; this page
  // just listens for its completion (any trigger — auto, header, Fab,
  // palette — bumps `revision`) and refetches the price-derived payloads:
  // the POST cleared the GET cache and new prices move both the dashboard
  // totals and the performance buckets. `seenRefresh` is deliberately NOT
  // `$state`: it is the effect's private baseline, seeded at page init so
  // the mount never self-refetches.
  async function reloadAfterRefresh(): Promise<void> {
    try {
      const fresh = await portfolioApi.dashboard()
      dash = fresh
      if (fresh) applyDashboardStatus(fresh)
    } catch {
      // Keep current data; the strip/header already reflect the outcome.
    }
    void loadPerformance(granularity)
  }

  let seenRefresh = priceRefresh.revision
  $effect(() => {
    const rev = priceRefresh.revision
    if (rev === seenRefresh) return
    seenRefresh = rev
    void reloadAfterRefresh()
  })

  const hasMultipleCurrencies = $derived((dash?.by_currency?.length ?? 0) > 1)
  const portfolioSlices = $derived(
    (dash?.portfolios ?? []).map((p) => ({ name: p.portfolio_name, value: Number(p.active.value) })),
  )

  // Zone C sparklines (EPIC K.3b, spec §6.1/§9.2): per-portfolio market-value
  // history rendered inside the portfolio cards. Fetched in the background
  // once the dashboard payload exists — the cards render immediately and the
  // sparklines drop in as responses land. At family scale a handful of
  // parallel `history(id)` GETs is acceptable (and the 60s GET cache dedupes
  // the post-refresh round); a batched vault-history endpoint is the backend
  // fast-follow if the portfolio count ever grows.
  let sparklines = $state<Record<string, SparklinePoint[]>>({})

  // Monotonic round guard (last-write-wins): whenever `dash` is replaced
  // (initial load, refetch after the session price refresh) a new round is
  // issued and only that round may write, so a late response can never land
  // after a newer one. Responses also key on the portfolio id, so they can
  // never attach to the wrong card. Failed fetches resolve silently: the
  // card simply renders without its sparkline (decorative data, no toast).
  let sparkRound = 0
  $effect(() => {
    const ids = (dash?.portfolios ?? []).map((p) => p.portfolio_id)
    const round = ++sparkRound
    ids.forEach((id) => {
      portfolioApi
        .history(id)
        .then((res) => {
          if (round === sparkRound) {
            sparklines[id] = res.series.map((pt) => ({
              date: pt.date,
              value: Number(pt.market_value),
            }))
          }
        })
        .catch(() => {
          if (round === sparkRound) sparklines[id] = []
        })
    })
  })

  // EPIC I.5 consolidated "Invested assets" table: open positions aggregated
  // across portfolios in the base currency, already sorted by descending
  // value server-side (kept as-is, no client re-sort). `?? []` also covers
  // older backends that still omit the field.
  const investedAssets = $derived(dash?.invested_assets ?? [])

  // First-run checklist progress (D8), built from the dashboard payload only.
  // The branch renders while `portfolios` is empty, so in practice the first
  // step is current and the rest pending; the derivations keep the card
  // honest if the payload is fresher than the branch decision (60s GET
  // cache). Positions are the only asset signal the vault payload carries,
  // so steps 2/3 share it: an asset alone (no portfolio) stays pending.
  const firstRun = $derived({
    portfolioCount: dash?.portfolios?.length ?? 0,
    assetCount: (dash?.invested_assets ?? []).length,
    transactionSeen: (dash?.portfolios ?? []).some(
      (p) => Number(p.active.invested) !== 0 || Number(p.closed.invested) !== 0,
    ),
  })

  // EPIC I.4 "Allocazione complessiva" card: region, sector and country
  // payloads are mapped onto the generic ExposureBarRow shape consumed by
  // ExposureBarChart (countries keep their raw ISO codes as row identity; the
  // chart maps them to full names via `labelFor`, see the country panel below).
  const regionBarRows = $derived<ExposureBarRow[]>(
    (alloc?.regions ?? []).map((r) => ({ name: r.region, value: r.value, weight: r.weight })),
  )
  const sectorBarRows = $derived<ExposureBarRow[]>(
    (alloc?.sectors ?? []).map((s) => ({ name: s.sector, value: s.value, weight: s.weight })),
  )
  const countryBarRows = $derived<ExposureBarRow[]>(
    (alloc?.countries ?? []).map((c) => ({ name: c.country, value: c.value, weight: c.weight })),
  )

  // Equity-universe coverage note for the bar charts, mirroring the long
  // covered/excluded note the donut charts render themselves; shown only when
  // non-equity holdings were actually excluded.
  const equityUniverseNote = $derived.by(() => {
    const covered = Number(alloc?.covered_value || 0)
    const excluded = Number(alloc?.excluded_value || 0)
    const total = covered + excluded
    if (total <= 0 || excluded <= 0) return undefined
    return t('allocation.equityUniverse', { pct: ((covered / total) * 100).toFixed(1) })
  })

  // Aggregated "Other" buckets are muted grey, like the slice treatment in the
  // donut charts. Read via chartSemanticColors so it re-evaluates on theme
  // flips (the {#key} blocks inside the charts re-init them anyway).
  function otherGrey(name: string): string | undefined {
    return name === 'Other' || name === 'Other / Not Classified'
      ? chartSemanticColors(resolved()).other
      : undefined
  }

  // The compact per-card closed line is dropped when the portfolio never sold
  // a lot. Closed dividends are now folded into `proceeds` by the backend, so
  // they no longer drive the visibility on their own.
  function hasClosedActivity(p: PortfolioPerformanceSummary): boolean {
    return Number(p.closed.invested) !== 0
  }

  // ── Allocation drill-down (EPIC K.5, spec §6.5) ─────────────────────────
  // One shared panel for the whole "Allocazione complessiva" card: clicking
  // a class slice or a sector/region/country bar sets the bucket and opens
  // it (drawer ≥ lg / sheet < lg, D4). The `key` is the RAW value the chart
  // carries (class key / ISO code / region / sector name — may contain
  // spaces); the `title` is the label the same chart displays. The vault
  // scope fetches `dashboardAllocationDrill`, in the base currency.
  let drillOpen = $state(false)
  let drillDim = $state<AllocationDrillDim>('class')
  let drillKey = $state('')
  let drillTitle = $state('')

  function openDrill(dim: AllocationDrillDim, key: string, label = key): void {
    drillDim = dim
    drillKey = key
    drillTitle = label
    drillOpen = true
  }
  function closeDrill(): void {
    drillOpen = false
  }
  // Stable per page instance (Svelte 5 script bodies run once): the panel's
  // fetch effect depends on the identity, not on the bucket values.
  function drillFetch(dim: AllocationDrillDim, key: string): Promise<AllocationDrill> {
    return portfolioApi.dashboardAllocationDrill(dim, key)
  }
</script>

<div class="p-6">
  <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
    <h1 class="text-2xl font-bold">{t('nav.dashboard')}</h1>
    {#if !loading && dash?.portfolios?.length}
      <!-- Scope switcher (D3): navigation, not a filter — picking a portfolio
           leaves for its detail page, which is the same analytics at
           portfolio scope. Built from the payload's own list. -->
      <ScopeSwitcher portfolios={dash.portfolios} class="w-full max-w-64 sm:w-64" />
    {/if}
  </div>

  {#if loading}
    <div class="flex justify-center py-24 text-muted-foreground">
      <Spinner size="lg" />
    </div>
  {:else if !dash?.portfolios?.length}
    <!-- Empty vault: guided first-run checklist (D8), replaces the plain
         EmptyState; auto-hides as soon as the first portfolio exists. -->
    <FirstRunChecklist
      portfolioCount={firstRun.portfolioCount}
      assetCount={firstRun.assetCount}
      transactionSeen={firstRun.transactionSeen}
    />
  {:else}
    <div class="space-y-6">
      {#if dash.summary}
        <!-- Zone A — hero (K.3a): ONE number (net market value of the active
             breakdown in the user's base currency), the signed P/L beneath it
             via `PnlValue` (never colour alone, D6), muted secondary chips
             (realized/dividends/invested). Desktop:
             2-up with the compact value-vs-invested chart (the former
             "Capital invested" card content, folded in here with the D10
             period chips); phones stack it under the number. -->
        <Card class="p-4 lg:p-6">
          <div class="grid items-center gap-6 lg:grid-cols-2">
            <div class="min-w-0">
              <h2 class="text-sm text-muted-foreground">{t('hero.netValue')}</h2>
              <p class="text-hero tabular-nums">
                {formatCurrency(dash.summary.active.value, dash.base_currency)}
              </p>
              <p class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-0.5">
                <PnlValue
                  value={dash.summary.active.gain_loss}
                  kind="currency"
                  currency={dash.base_currency}
                  size="lg"
                />
                <PnlValue value={dash.summary.active.gain_loss_pct} kind="percent" size="lg" />
                <span class="text-xs text-muted-foreground">{t('hero.allTime')}</span>
              </p>
              <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span class="inline-flex items-center gap-1.5">
                  {t('hero.realized')}
                  <PnlValue
                    value={dash.summary.closed.realized}
                    kind="currency"
                    currency={dash.base_currency}
                    size="sm"
                  />
                </span>
                <span class="inline-flex items-center gap-1.5 tabular-nums">
                  {t('hero.dividends')}
                  {formatCurrency(dash.summary.active.dividends, dash.base_currency)}
                </span>
                <span class="inline-flex items-center gap-1.5 tabular-nums">
                  {t('hero.invested')}
                  {formatCurrency(dash.summary.active.invested, dash.base_currency)}
                </span>
              </div>
              <!-- The Active/Closed roll-up stays available on demand instead
                   of dominating the top of the page (progressive density). -->
              <details class="group mt-4">
                <summary
                  class="focus-ring inline-flex cursor-pointer list-none items-center gap-1.5 rounded-control px-1 py-1 text-sm font-medium text-muted-foreground transition-colors duration-fast ease-standard hover:text-foreground [&::-webkit-details-marker]:hidden"
                >
                  <ChevronDown
                    class="h-4 w-4 transition-transform duration-base ease-standard group-open:rotate-180"
                    aria-hidden="true"
                  />
                  {t('hero.breakdown')}
                </summary>
                <div class="mt-3">
                  <InvestmentsTable
                    active={dash.summary.active}
                    closed={dash.summary.closed}
                    currency={dash.base_currency}
                  />
                </div>
              </details>
            </div>

            <div class="min-w-0">
              <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                <h3 class="text-sm font-semibold">{t('hero.valueVsInvested')}</h3>
                {#if heroPeriodChips.length > 1}
                  <PeriodChips
                    periods={heroPeriodChips}
                    value={heroPeriod}
                    onchange={selectHeroPeriod}
                    ariaLabel={t('period.group')}
                  />
                {/if}
              </div>
              {#if perfLoading}
                <div class="flex h-[240px] items-center justify-center text-muted-foreground">
                  <Spinner />
                </div>
              {:else}
                <CapitalChart
                  buckets={heroBuckets}
                  currency={perf?.currency || dash.base_currency || 'USD'}
                  granularity={perf?.granularity ?? 'month'}
                  compact
                />
              {/if}
            </div>
          </div>
        </Card>
      {/if}

      <div class="grid gap-4 lg:grid-cols-2">
        <!-- Zone B — percentage return card: bars = per-bucket time-weighted
             return, line = cumulative TWR. The Monthly/Annual control lives
             here and drives `granularity`, which also feeds the hero chart
             above (both share the one `dashboardPerformance` fetch). -->
        <Card class="p-4">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
            <h2 class="font-semibold">{t('performance.title')}</h2>
            <SegmentedControl
              items={perfItems}
              bind:value={getGranularity, setGranularity}
              ariaLabel={t('performance.granularity')}
            />
          </div>
          {#if perfLoading}
            <div class="flex h-[340px] items-center justify-center text-muted-foreground">
              <Spinner />
            </div>
          {:else}
            <PerformanceChart buckets={perf?.buckets ?? []} granularity={granularity} />
          {/if}
        </Card>

        <Card class="p-4">
          <h2 class="mb-4 font-semibold">{t('allocation.byPortfolio')}</h2>
          <AllocationDonut
            data={portfolioSlices}
            title={t('allocation.byPortfolio')}
            currency={dash.base_currency || dash.by_currency[0]?.currency || 'USD'}
            showValue={!hasMultipleCurrencies}
          />
          {#if hasMultipleCurrencies}
            <p class="mt-2 text-xs text-muted-foreground">
              {t('allocation.mixedCurrencies')}
            </p>
          {/if}
        </Card>
      </div>

      <div>
        <h2 class="mb-4 font-semibold">{t('nav.portfolios')}</h2>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {#each dash.portfolios as p (p.portfolio_id)}
            {@const spark = sparklines[p.portfolio_id]}
            <Card class="transition-colors hover:border-accent">
              <a href={resolve(`/portfolios/${p.portfolio_id}`)} class="block p-4">
                <div class="flex items-baseline justify-between gap-2">
                  <span class="truncate font-semibold">{p.portfolio_name}</span>
                  <span class="shrink-0 text-xs text-muted-foreground">{p.currency}</span>
                </div>
                <p class="mt-2 text-lg font-bold tabular-nums">{formatCurrency(p.active.value, p.currency)}</p>
                <div class="mt-1 flex items-center justify-between text-sm">
                  <span class="font-medium tabular-nums {pnlColorClass(p.active.gain_loss)}">
                    {formatCurrency(p.active.gain_loss, p.currency)}
                  </span>
                  <span class="font-medium tabular-nums {pnlColorClass(p.active.gain_loss_pct)}">
                    {formatPercent(p.active.gain_loss_pct)}
                  </span>
                </div>
                {#if hasClosedActivity(p)}
                  <p class="mt-2 text-xs tabular-nums text-muted-foreground">
                    {t('dashboard.closedPrefix')} {formatCurrency(p.closed.invested, p.currency)} ·
                    {formatCurrency(p.closed.proceeds, p.currency)} ·
                    <span class="font-medium {pnlColorClass(p.closed.realized)}">
                      {formatCurrency(p.closed.realized, p.currency)}
                    </span>
                  </p>
                {/if}
                <p class="mt-2 text-xs text-muted-foreground">{t('common.assetCount', { count: p.asset_count })}</p>
                {#if spark && spark.length > 1}
                  <!-- Value-history sparkline as a bottom strip (K.3b): only
                       present once the background fetch has landed, so the
                       card never reserves space or blocks on it. -->
                  <Sparkline
                    class="mt-3"
                    points={spark}
                    ariaLabel={t('sparkline.valueTrend', { name: p.portfolio_name })}
                  />
                {/if}
              </a>
            </Card>
          {/each}
        </div>
      </div>

      <div class="rounded-card border-border bg-surface p-4 shadow-card">
        <h2 class="mb-4 font-semibold">{t('allocation.title')}</h2>
        {#if alloc == null}
          <p class="text-sm text-muted-foreground">{t('allocation.unavailable')}</p>
        {:else}
          <!-- EPIC I.4 layout: asset-class donut over the whole vault plus the
               equity-only breakdown (sector, region and country bars), in a
               responsive 2-column grid; regions sit next to countries in the
               bottom row. The inner panels reuse the shared card surface
               styling; the portfolio detail "Allocazione" section mirrors this
               exact layout in EPIC I.7 (#86). -->
          <div class="grid gap-4 lg:grid-cols-2">
            <div class="rounded-card border-border bg-surface p-4 shadow-card">
              <ClassDonut
                data={alloc.classes ?? []}
                currency={alloc.currency}
                label={t('allocation.assetClasses')}
                onDrill={(cls) => openDrill('class', cls, assetClassLabel(cls))}
              />
            </div>
            <div class="rounded-card border-border bg-surface p-4 shadow-card">
              <ExposureBarChart
                rows={sectorBarRows}
                currency={alloc.currency}
                label={t('allocation.sectorsEquity')}
                note={equityUniverseNote}
                colorFor={otherGrey}
                onDrill={(name) => openDrill('sector', name)}
              />
            </div>
            <div class="rounded-card border-border bg-surface p-4 shadow-card">
              <ExposureBarChart
                rows={regionBarRows}
                currency={alloc.currency}
                label={t('allocation.regionsEquity')}
                note={equityUniverseNote}
                colorFor={otherGrey}
                onDrill={(name) => openDrill('region', name)}
              />
            </div>
            <div class="rounded-card border-border bg-surface p-4 shadow-card">
              <ExposureBarChart
                rows={countryBarRows}
                currency={alloc.currency}
                label={t('allocation.countriesEquity')}
                note={equityUniverseNote}
                colorFor={otherGrey}
                labelFor={countryDisplayName}
                maxVisibleRows={10}
                onDrill={(code) => openDrill('country', code, countryDisplayName(code))}
              />
            </div>
          </div>
          <!-- One drill panel per page (D4): the clicked chart fills the
               bucket and the fetcher runs the wealth-scope request. -->
          <AllocationDrillPanel
            open={drillOpen}
            onClose={closeDrill}
            title={drillTitle}
            dim={drillDim}
            key={drillKey}
            fetcher={drillFetch}
            currencyHint={alloc.currency}
          />
        {/if}
      </div>

      <!-- EPIC I.5: consolidated invested-assets table replacing the old
           per-portfolio accordions. One row per open asset merged across all
           portfolios, in the base currency, ordered by value descending as
           returned by the backend. -->
      <Card class="p-4">
        <h2 class="mb-3 font-semibold">{t('dashboard.investedAssets')}</h2>
        {#if investedAssets.length === 0}
          <EmptyState
            dashed
            title={t('dashboard.noInvestedAssets')}
            description={t('dashboard.noInvestedAssetsHint')}
          />
        {:else}
          <div class="overflow-x-auto">
            <Table aria-label={t('dashboard.investedAssets')}>
              <THead>
                <Tr>
                  <Th>{t('dashboard.colAsset')}</Th>
                  <Th align="right">{t('chartView.colInvested')}</Th>
                  <Th align="right">{t('chartView.colValue')}</Th>
                  <Th align="right">{t('dashboard.colGainLoss')}</Th>
                  <Th align="right">{t('dashboard.colPnlPct')}</Th>
                </Tr>
              </THead>
              <TBody>
                {#each investedAssets as a (a.asset_id)}
                  <Tr>
                    <Td>
                      <a
                        href={resolve(`/assets/${a.asset_id}`)}
                        class="font-medium text-accent-text hover:underline"
                      >
                        {a.ticker}
                      </a>
                      {#if !a.has_price}
                        <Badge
                          variant="neutral"
                          class="ml-1.5 align-middle"
                          title={t('dashboard.noPriceHint')}
                        >
                          {t('dashboard.noPrice')}
                        </Badge>
                      {/if}
                      <span class="block text-xs text-muted-foreground">{a.name}</span>
                    </Td>
                    <Td align="right">{formatCurrency(a.invested, dash.base_currency)}</Td>
                    <Td align="right">{formatCurrency(a.value, dash.base_currency)}</Td>
                    <Td align="right" class="font-medium {pnlColorClass(a.gain_loss)}">
                      {formatCurrency(a.gain_loss, dash.base_currency)}
                    </Td>
                    <Td align="right" class={pnlColorClass(a.gain_loss_pct)}>
                      {formatPercent(a.gain_loss_pct)}
                    </Td>
                  </Tr>
                {/each}
              </TBody>
            </Table>
          </div>
        {/if}
      </Card>
    </div>
  {/if}
</div>
