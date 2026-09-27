<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use } from 'echarts/core'
  import { LineChart } from 'echarts/charts'
  import {
    DataZoomComponent,
    GridComponent,
    LegendComponent,
    TooltipComponent,
  } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { formatCurrency } from '$lib/format'
  import type { PerformanceBucket } from '$lib/services/api'
  import { chartSemanticColors } from '$lib/chartPalette'
  import { PECULIUM_CHART_THEMES } from '$lib/chartTheme'
  import { resolved } from '$lib/stores/theme.svelte'
  import { cx } from '$lib/components/ui/utils'
  import { t } from '$lib/i18n/index.svelte'
  import ChartTableToggle from '$lib/components/ui/ChartTableToggle.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'

  use([
    LineChart,
    DataZoomComponent,
    GridComponent,
    LegendComponent,
    TooltipComponent,
    CanvasRenderer,
  ])

  interface TooltipRow {
    marker: string
    seriesName: string
    value: unknown
    axisValue?: string | number
  }

  let {
    buckets = [] as PerformanceBucket[],
    currency = 'USD',
    granularity = 'month' as 'month' | 'year',
    /**
     * Hero variant (EPIC K.3a): a shorter canvas with no slider dataZoom for
     * the 2-up hero card; `false` keeps the exact standalone-card geometry.
     */
    compact = false,
    /**
     * EPIC K.5b: the shared Chart ⇄ Table disclosure (spec §9.1). Also
     * shown in the `compact` hero variant: the table is the phone-friendly
     * read of the same buckets.
     */
    showTableToggle = true,
  } = $props()

  // Line colors come from the semantic chart tokens (grey cost basis for the
  // invested capital, green market value), re-evaluated on theme flips (the
  // {#key} block below also re-inits the chart with the new ECharts theme),
  // same convention as PositionChart / PerformanceChart.
  const semantic = $derived(chartSemanticColors(resolved()))

  /** "2025-06" → "Jun 2025" (month name follows the browser locale);
   * "2025" stays "2025" on the annual granularity. */
  function formatPeriod(period: string): string {
    if (granularity !== 'month') return period
    const [year, month] = period.split('-').map(Number)
    if (!year || !month) return period
    return new Date(year, month - 1, 1).toLocaleDateString(undefined, {
      month: 'short',
      year: 'numeric',
    })
  }

  // ── "View as table" (EPIC K.5b, spec §9.1) ──────────────────────────────
  // One row per plotted bucket with the two series the lines carry (net
  // invested capital and market value), reusing the x-axis period labels and
  // `formatCurrency` — the same formatter the tooltip uses. Switching
  // unmounts the canvas (out of the a11y tree); empty data keeps showing the
  // empty state. The chart has no own heading (the hero card titles it), so
  // the accessible name comes from the dictionary.
  const chartName = $derived(t('chartView.nameCapital'))
  let view = $state<'chart' | 'table'>('chart')
  const showTable = $derived(showTableToggle && view === 'table')

  // Series identities (legend/tooltip) are translated labels: `Invested`
  // reuses the hero line key, `Value` the shared chart-table column header.
  const investedName = $derived(t('hero.invested'))
  const valueName = $derived(t('chartView.colValue'))

  const options = $derived.by((): EChartsOption => {
    const rows = buckets ?? []
    return {
      tooltip: {
        trigger: 'axis',
        formatter: (params: unknown) => {
          const raw = Array.isArray(params) ? params : [params]
          const series = raw as TooltipRow[]
          const period = series[0]?.axisValue ?? ''
          const lines = series
            .filter((p) => p.value != null)
            .map((p) => {
              const v = Array.isArray(p.value) ? p.value[1] : p.value
              return `${p.marker}${p.seriesName}: <b>${formatCurrency(Number(v), currency)}</b>`
            })
          return `<div>${period}</div>${lines.join('<br/>')}`
        },
      },
      legend: {
        data: [investedName, valueName],
        top: 0,
      },
      grid: { left: 48, right: 16, top: 40, bottom: compact ? 24 : 52 },
      // Long monthly ranges stay usable: wheel/drag zoom plus the slider; the
      // compact hero variant drops the slider (the period chips already scope
      // the range) but keeps wheel/drag zoom.
      dataZoom: compact
        ? [{ type: 'inside', xAxisIndex: 0 }]
        : [
            { type: 'inside', xAxisIndex: 0 },
            { type: 'slider', xAxisIndex: 0, bottom: 0 },
          ],
      xAxis: {
        type: 'category',
        data: rows.map((b) => formatPeriod(b.period)),
        axisLabel: { fontSize: 11, hideOverlap: true },
      },
      yAxis: {
        type: 'value',
        axisLabel: { fontSize: 11 },
      },
      series: [
        {
          name: investedName,
          type: 'line',
          // Net invested capital at the end of each bucket: it only moves on
          // cash flows, so a stepped line mirrors the PositionChart cost basis.
          data: rows.map((b) => Number(b.invested)),
          step: 'end',
          smooth: false,
          symbolSize: 5,
          lineStyle: { width: 2, color: semantic.costBasis },
          itemStyle: { color: semantic.costBasis },
        },
        {
          name: valueName,
          type: 'line',
          // Market value at the end of each bucket.
          data: rows.map((b) => Number(b.value)),
          smooth: true,
          symbolSize: 5,
          lineStyle: { width: 2, color: semantic.marketValue },
          itemStyle: { color: semantic.marketValue },
        },
      ],
    }
  })
</script>

{#if (buckets ?? []).length === 0}
  <div class={cx('flex w-full items-center justify-center text-sm text-muted-foreground', compact ? 'h-[240px]' : 'h-[340px]')}>
    {t('chartView.noData')}
  </div>
{:else}
  {#if showTableToggle}
    <div class="mb-2 flex justify-end">
      <ChartTableToggle name={chartName} bind:view />
    </div>
  {/if}
  {#if showTable}
    <!-- No wrapper scroll container (EPIC K bug-fix): the page is the only
         scroll container. `w-full` + wrapping period labels stay inside the
         card; below `sm` rows collapse to a stacked key–value grid. -->
    <Table class="max-sm:block table-fixed">
      <caption class="sr-only">{t('chartView.caption', { name: chartName })}</caption>
      <THead class="max-sm:block">
        <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
          <Th class="max-sm:col-span-2 max-sm:py-0.5 break-words">{t('chartView.colPeriod')}</Th>
          <Th align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">{t('chartView.colInvested')}</Th>
          <Th align="right" class="max-sm:min-w-0 max-sm:py-0.5 whitespace-nowrap">{t('chartView.colValue')}</Th>
        </Tr>
      </THead>
      <TBody class="max-sm:block">
        {#each buckets as b (b.period)}
          <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
            <Td class="max-sm:col-span-2 max-sm:py-0.5 font-medium break-words">
              {formatPeriod(b.period)}
            </Td>
            <Td align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">
              {formatCurrency(b.invested, currency)}
            </Td>
            <Td align="right" class="max-sm:min-w-0 max-sm:py-0.5 whitespace-nowrap">
              {formatCurrency(b.value, currency)}
            </Td>
          </Tr>
        {/each}
      </TBody>
    </Table>
  {:else}
    <div class={cx('w-full', compact ? 'h-[240px]' : 'h-[340px]')}>
      <!-- {#key} re-inits the chart when the theme flips so the ECharts theme
           object passed below is picked up (svelte-echarts only reads `theme`
           at init time). -->
      {#key resolved()}
        <Chart {init} {options} theme={PECULIUM_CHART_THEMES[resolved()]} />
      {/key}
    </div>
  {/if}
{/if}
