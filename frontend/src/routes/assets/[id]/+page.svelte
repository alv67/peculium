<script lang="ts">
  import { resolve } from '$app/paths'
  import { t } from '$lib/i18n/index.svelte'
  import {
    formatCurrency,
    assetTypeLabel,
    assetClassLabel,
    priceSourceLabel,
  } from '$lib/format'
  import PriceChart from '$lib/components/PriceChart.svelte'
  import PnlValue from '$lib/components/ui/PnlValue.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import EmptyState from '$lib/components/ui/EmptyState.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'
  import { getAssetPage } from './context'

  /**
   * Overview tab (EPIC K.4b, spec §6.3): the price chart with its
   * 1M/3M/1Y/YTD/MAX in-place zoom and split markers (exactly the card the
   * old single page held, minus the "Metriche quote" card whose values moved
   * into the sticky header), the NEW "Where held" block (spec §4.2 decision
   * 5) and a read-only quick-facts grid of the asset identity. The chart's
   * range selection is card-level control state (like the portfolio
   * Overview's history selector in K.4a), so it lives here; every payload —
   * prices, splits, the dashboard-derived `held` rows and the asset itself —
   * comes from the shell context. Nothing is fetched here.
   */
  const ctx = getAssetPage()

  const RANGES = [
    { key: '1M', days: 30 },
    { key: '3M', days: 90 },
    { key: '1Y', days: 365 },
    { key: 'YTD', days: -1 },
    { key: 'MAX', days: Infinity },
  ] as const
  type RangeKey = (typeof RANGES)[number]['key']

  let range = $state<RangeKey | null>('1Y')
  let programmaticallyZooming = $state(false)

  const chartSeries = $derived.by(() => {
    return [...ctx.prices]
      .sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
      .map((p) => ({ date: p.date, close: p.close }))
  })

  const zoomStart = $derived.by(() => {
    if (!range) return null
    if (range === 'MAX') return 'MAX' as const
    if (range === 'YTD') {
      const now = new Date()
      const ytd = new Date(now.getFullYear(), 0, 1)
      return ytd.toISOString().slice(0, 10)
    }
    const days = RANGES.find((r) => r.key === range)?.days ?? 365
    const cutoff = Date.now() - days * 24 * 60 * 60 * 1000
    return new Date(cutoff).toISOString().slice(0, 10)
  })

  function selectRange(r: RangeKey): void {
    programmaticallyZooming = true
    range = r
    setTimeout(() => (programmaticallyZooming = false), 300)
  }

  function handleChartZoom(): void {
    if (!programmaticallyZooming) {
      range = null
    }
  }

  // Data-completeness of the price history (#193): `null` for non-Yahoo
  // assets (payload carries no `data_status`) → the chart renders as before.
  const status = $derived(ctx.asset?.data_status?.state ?? null)
  // Progress of the running job (its processed/total), when it reports one:
  // the per-asset history backfill counts 2 work items (history + splits),
  // the global asset sync reports none → indeterminate bar.
  const syncTotal = $derived(ctx.asset?.data_status?.total ?? 0)
  const syncPercent = $derived(
    syncTotal > 0
      ? Math.min(100, Math.round(((ctx.asset?.data_status?.processed ?? 0) / syncTotal) * 100))
      : 0,
  )

  // Read-only identity summary of the loaded asset: the values the edit form
  // (Data tab) carries, as label/value pairs. ISINs and codes use the mono
  // font (D5); a missing value falls back to an em dash.
  const facts = $derived.by(() => {
    const a = ctx.asset
    if (!a) return []
    return [
      { label: t('asset.factIsin'), value: a.isin || '—', mono: true },
      { label: t('asset.factType'), value: assetTypeLabel(a.type), mono: false },
      {
        label: t('asset.factClass'),
        value: a.asset_class ? assetClassLabel(a.asset_class) : '—',
        mono: false,
      },
      { label: t('asset.factCurrency'), value: a.currency, mono: false },
      { label: t('asset.factExchange'), value: a.exchange || '—', mono: false },
      {
        label: t('asset.factPriceSource'),
        value: priceSourceLabel(a.price_source || 'yahoo'),
        mono: false,
      },
    ]
  })
</script>

{#snippet syncIndicator()}
  <!-- #193: while a backfill/sync job updates this asset. A determinate bar
       when the job reports processed/total, an indeterminate one otherwise. -->
  <div class="flex w-full max-w-64 flex-col items-center gap-2 text-sm text-muted-foreground">
    <span>{t('asset.dataSyncing')}</span>
    <div
      class="h-1.5 w-full overflow-hidden rounded-full bg-muted"
      role="progressbar"
      aria-label={t('asset.dataSyncing')}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={syncTotal > 0 ? syncPercent : undefined}
    >
      <div
        class="h-full rounded-full bg-accent {syncTotal > 0 ? '' : 'w-full animate-pulse'}"
        style={syncTotal > 0 ? `width: ${syncPercent}%` : undefined}
      ></div>
    </div>
  </div>
{/snippet}

{#if ctx.asset}
  <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h2 class="font-semibold">{t('asset.priceHistory')}</h2>
      <div class="flex gap-1">
        {#each RANGES as r (r.key)}
          <button
            onclick={() => selectRange(r.key)}
            class="rounded-control px-3 py-1.5 text-sm {range === r.key
              ? 'bg-accent text-accent-foreground'
              : 'text-muted-foreground hover:bg-muted'}"
          >
            {r.key}
          </button>
        {/each}
      </div>
    </div>
    {#if (ctx.backfillingHistory || ctx.dataSyncing) && ctx.prices.length === 0}
      <!-- Syncing with nothing to plot yet (#193): a progress bar instead of a
           silently empty chart. -->
      <div class="flex h-[340px] items-center justify-center">
        {@render syncIndicator()}
      </div>
    {:else if ctx.prices.length === 0 && status === 'missing'}
      {#snippet backfillAction()}
        <Button variant="secondary" disabled={ctx.backfillingHistory} onclick={ctx.backfillHistory}>
          {t('asset.backfillHistory')}
        </Button>
      {/snippet}
      <EmptyState
        dashed
        title={t('asset.dataMissing')}
        action={backfillAction}
      />
    {:else}
      <PriceChart
        series={chartSeries}
        currency={ctx.currency}
        {zoomStart}
        splits={ctx.splits}
        onDataZoom={handleChartZoom}
      />
      {#if ctx.dataSyncing}
        <!-- Still updating with data already plotted: progress bar, no action
             (a backfill is already running). -->
        <div class="mt-3 flex justify-center">
          {@render syncIndicator()}
        </div>
      {:else if status === 'partial'}
        <div class="mt-2 flex flex-wrap items-center justify-center gap-2 text-sm text-muted-foreground">
          {t('asset.dataPartial')}
          <Button variant="secondary" size="sm" disabled={ctx.backfillingHistory} onclick={ctx.backfillHistory}>
            {t('asset.backfillHistory')}
          </Button>
        </div>
      {/if}
    {/if}
  </div>

  <!-- "Where held" (spec §4.2 decision 5, NEW in K.4b): which of the user's
       portfolios currently hold this asset, at what qty/cost/value and with
       what open P/L — derived client-side from the per-portfolio `assets` of
       `GET /dashboard` by the shell's isolated `loadHoldings()` (amounts in
       the asset's own currency, richest holding first), so no backend
       addition was needed. While that fetch has not resolved the block
       renders nothing (it never blocks the page); on failure it degrades to
       the muted "unavailable" note, and an empty result to the "not held"
       line. -->
  {#if ctx.heldFailed}
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="mb-2 font-semibold">{t('asset.whereHeld')}</h2>
      <p class="text-sm text-muted-foreground">{t('asset.whereHeldUnavailable')}</p>
    </div>
  {:else if ctx.held !== null}
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="mb-4 font-semibold">{t('asset.whereHeld')}</h2>
      {#if ctx.held.length === 0}
        <p class="text-sm text-muted-foreground">{t('asset.whereHeldEmpty')}</p>
      {:else}
        <div class="overflow-x-auto">
          <Table aria-label={t('asset.whereHeld')}>
            <THead>
              <Tr>
                <Th>{t('asset.heldPortfolio')}</Th>
                <Th align="right">{t('asset.heldQty')}</Th>
                <Th align="right">{t('asset.heldCost')}</Th>
                <Th align="right">{t('asset.heldValue')}</Th>
                <Th align="right">{t('asset.heldGl')}</Th>
              </Tr>
            </THead>
            <TBody>
              {#each ctx.held as row (row.portfolioId)}
                <Tr>
                  <Td>
                    <a
                      href={resolve(`/portfolios/${row.portfolioId}`)}
                      class="font-medium text-accent-text hover:underline"
                    >{row.portfolioName}</a>
                  </Td>
                  <Td align="right">{Number(row.qty)}</Td>
                  <Td align="right">{formatCurrency(row.invested, row.currency)}</Td>
                  <Td align="right">{formatCurrency(row.value, row.currency)}</Td>
                  <Td align="right">
                    <PnlValue value={row.gain_loss} kind="currency" currency={row.currency} size="sm" />
                    <span class="ml-2 inline-block align-top">
                      <PnlValue value={row.roi} kind="percent" size="sm" />
                    </span>
                  </Td>
                </Tr>
              {/each}
            </TBody>
          </Table>
        </div>
      {/if}
    </div>
  {/if}

  <!-- Quick facts: the asset identity at a glance (spec §6.3 Overview zone).
       Editing lives in the Data tab's form. -->
  <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
    <h2 class="mb-4 font-semibold">{t('asset.quickFacts')}</h2>
    <dl class="grid grid-cols-2 gap-x-4 gap-y-3 md:grid-cols-3">
      {#each facts as fact (fact.label)}
        <div>
          <dt class="text-xs font-medium text-muted-foreground">{fact.label}</dt>
          <dd class="mt-0.5 text-sm {fact.mono ? 'font-mono' : ''}">{fact.value}</dd>
        </div>
      {/each}
    </dl>
  </div>
{/if}
