<script lang="ts">
  import { resolve } from '$app/paths'
  import { t } from '#lib/i18n/index.svelte.js'
  import { ArrowRight } from 'lucide-svelte'
  import ClassDonut from '#lib/components/domain/ClassDonut.svelte'
  import InvestmentsTable from '#lib/components/domain/InvestmentsTable.svelte'
  import PerformanceChart from '#lib/components/domain/PerformanceChart.svelte'
  import PositionChart from '#lib/components/PositionChart.svelte'
  import Card from '#lib/components/ui/Card.svelte'
  import SegmentedControl from '#lib/components/ui/SegmentedControl.svelte'
  import Spinner from '#lib/components/ui/Spinner.svelte'
  import { getPortfolioPage } from './context'

  /**
   * Overview tab (EPIC K.4a, spec §6.2): the Investments breakdown card,
   * the portfolio's own TWR performance card (EPIC I.8) with its
   * Monthly/Annual toggle, the value-history secondary view (the former
   * "Performance history" card) and an allocation digest (class donut +
   * link to the Allocation tab) — the same widgets the single long page
   * used to stack, minus the Positions table (→ Positions tab) and the
   * transactions page (→ Activity tab). Every value is read from the shell
   * context; nothing is fetched here.
   */
  const ctx = getPortfolioPage();

  const currency = $derived(ctx.currency);

  // Card-level control state only: which history series is plotted.
  let selectedAsset = $state('')

  const perfItems = $derived([
    { value: 'month', label: t('performance.monthly') },
    { value: 'year', label: t('performance.annual') },
  ])
</script>

{#if ctx.summary}
  <!-- Same "Investments" card the dashboard shows (EPIC I.6, #85), fed by
       the portfolio-currency active/closed roll-ups of
       `GET /portfolios/{id}/summary`; the old flat KPI row (Value /
       Realized / Open G/L / Assets) is gone, only the asset count stays as
       a muted secondary line under the card. -->
  <div class="mb-6">
    <InvestmentsTable
      active={ctx.summary.active}
      closed={ctx.summary.closed}
      currency={currency}
    />

    <p class="mt-2 text-xs text-muted-foreground">{t('common.assetCount', { count: ctx.summary.asset_count })}</p>
  </div>
{/if}

<!-- EPIC I.8 (#87): the portfolio's own percentage performance, same shared
     `PerformanceChart` as the dashboard card (bars = per-bucket `return` %,
     line = cumulative `twr`) with its own Monthly/Annual toggle; amounts in
     the Performance bucket payloads are already in the portfolio currency,
     and both series are pure percentages so no `currency` prop is needed.
     The buckets live in the shell so the session price refresh and the
     post-mutation refetches (E.9) keep updating the card from any tab. -->
<Card class="mb-6 p-4">
  <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
    <h2 class="font-semibold">{t('performance.title')}</h2>
    <SegmentedControl
      items={perfItems}
      bind:value={ctx.getGranularity, ctx.setGranularity}
      ariaLabel={t('performance.granularity')}
    />
  </div>
  {#if ctx.perfLoading}
    <div class="flex h-[340px] items-center justify-center text-muted-foreground">
      <Spinner />
    </div>
  {:else}
    <PerformanceChart buckets={ctx.perf?.buckets ?? []} granularity={ctx.granularity} />
  {/if}
</Card>

<!-- Secondary view (kept below the percentage chart, EPIC I.8 #87): the
     raw invested/value/realized capital lines with the per-asset selector. -->
<div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
  <h2 class="mb-4 font-semibold">{t('portfolio.performanceHistory')}</h2>
  {#if ctx.history && ctx.history.series.length > 0}
    <div class="mb-4">
      <select bind:value={selectedAsset} class="rounded-control border border-input px-3 py-2 text-sm">
        <option value="">{t('portfolio.seriesPortfolio')}</option>
        {#each ctx.history.assets as a (a.asset_id)}
          <option value={a.asset_id}>{a.ticker} - {a.name}</option>
        {/each}
      </select>
    </div>
    <PositionChart
      series={selectedAsset
        ? ctx.history.assets.find((a) => a.asset_id === selectedAsset)?.series ?? []
        : ctx.history.series}
      splits={selectedAsset
        ? ctx.history.assets.find((a) => a.asset_id === selectedAsset)?.splits ?? []
        : ctx.history.splits}
      currency={currency}
    />
  {:else}
    <p class="text-sm text-muted-foreground">{t('chartView.noData')}</p>
  {/if}
</div>

<!-- Allocation digest (spec §6.2 Overview zone): the class donut up front
     (EPIC I.7 payload, portfolio currency, same isolated "non disponibile"
     fallback as the Allocation tab) with a link to the full region/sector/
     country breakdown. -->
<Card class="p-4">
  <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
    <h2 class="font-semibold">{t('portfolio.tabAllocation')}</h2>
    {#if ctx.id}
      <a
        href={resolve(`portfolios/${ctx.id}/allocation`)}
        class="focus-ring inline-flex items-center gap-1 rounded-control text-sm font-medium text-accent-text hover:underline"
      >
        {t('portfolio.viewAllocation')}
        <ArrowRight class="h-4 w-4 shrink-0" aria-hidden="true" />
      </a>
    {/if}
  </div>
  {#if ctx.classAllocError}
    <p class="text-sm text-muted-foreground">{t('allocation.classUnavailable')}</p>
  {:else}
    <ClassDonut
      data={ctx.classAlloc?.classes ?? []}
      currency={ctx.classAlloc?.currency || currency}
      label={t('allocation.assetClasses')}
    />
  {/if}
</Card>
