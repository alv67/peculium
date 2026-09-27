<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use } from 'echarts/core'
  import { BarChart, LineChart } from 'echarts/charts'
  import {
    DataZoomComponent,
    GridComponent,
    LegendComponent,
    TooltipComponent,
  } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { formatSignedPercent } from '$lib/format'
  import type { PerformanceBucket } from '$lib/services/api'
  import { chartSemanticColors } from '$lib/chartPalette'
  import { PECULIUM_CHART_THEMES } from '$lib/chartTheme'
  import { palette } from '$lib/stores/palette.svelte'
  import { resolved } from '$lib/stores/theme.svelte'
  import { pnlColorClass } from '$lib/ui-colors'
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
    BarChart,
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
    granularity = 'month' as 'month' | 'year',
    // EPIC K.5b: the shared Chart ⇄ Table disclosure (spec §9.1). Callers
    // that pair this chart with their own bucket table can opt out.
    showTableToggle = true,
  }: {
    buckets?: PerformanceBucket[]
    granularity?: 'month' | 'year'
    showTableToggle?: boolean
  } = $props()

  // Bar/line colors come from the semantic chart tokens, re-evaluated on
  // theme flips and on CVD-palette flips (`chartSemanticColors` tracks
  // `palette.cvd`); the {#key} block below also re-inits the chart with the
  // new ECharts theme, same convention as PositionChart.
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
  // One row per plotted bucket, reusing the very period labels the x-axis
  // shows; the signed return keeps `pnlColorClass` so the table reads like
  // the green/red bars. Switching unmounts the canvas (out of the a11y
  // tree) and empty data still shows the empty state, never an empty
  // table. The chart carries no own heading, hence the dictionary name.
  const chartName = $derived(t('chartView.namePerformance'))
  let view = $state<'chart' | 'table'>('chart')
  const showTable = $derived(showTableToggle && view === 'table')

  // Series identities (legend/tooltip): the bar carries the per-bucket
  // gain/loss (same label as the dashboard table column), the line the
  // cumulative TWR.
  const gainLossName = $derived(t('dashboard.colGainLoss'))
  const cumulativeName = $derived(t('chartView.seriesCumulative'))

  const options = $derived.by((): EChartsOption => {
    const rows = buckets ?? []
    return {
      tooltip: {
        trigger: 'axis',
        formatter: (params: unknown) => {
          const raw = Array.isArray(params) ? params : [params]
          const series = raw as TooltipRow[]
          const period = series[0]?.axisValue ?? ''
          // Both series are percentages (return and cumulative TWR): no
          // currency formatting here, just a signed `+3.42%`.
          const lines = series
            .filter((p) => p.value != null)
            .map((p) => {
              const v = Array.isArray(p.value) ? p.value[1] : p.value
              return `${p.marker}${p.seriesName}: <b>${formatSignedPercent(Number(v))}</b>`
            })
          return `<div>${period}</div>${lines.join('<br/>')}`
        },
      },
      legend: {
        data: [gainLossName, cumulativeName],
        top: 0,
      },
      grid: { left: 56, right: 16, top: 40, bottom: 52 },
      // Long monthly ranges stay usable: wheel/drag zoom plus the slider.
      dataZoom: [
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
        // Percentage axis: the values are already % numbers (e.g. 3.42).
        axisLabel: { fontSize: 11, formatter: '{value}%' },
      },
      series: [
        {
          name: gainLossName,
          type: 'bar',
          // Time-weighted return generated inside each bucket: green bar
          // when positive, red when negative (per-bar itemStyle, mirrors
          // `pnlColorClass`).
          data: rows.map((b) => {
            const ret = Number(b.return)
            return { value: ret, itemStyle: { color: ret >= 0 ? semantic.positive : semantic.negative } }
          }),
          itemStyle: { color: semantic.positive },
          barMaxWidth: 28,
        },
        {
          name: cumulativeName,
          type: 'line',
          // Cumulative time-weighted return (TWR) up to the end of each
          // bucket; reuses the amber cumulative-line semantic token.
          data: rows.map((b) => Number(b.twr)),
          smooth: false,
          symbolSize: 5,
          lineStyle: { width: 2, color: semantic.realized },
          itemStyle: { color: semantic.realized },
        },
      ],
    }
  })
</script>

{#if (buckets ?? []).length === 0}
  <div class="flex h-[340px] w-full items-center justify-center text-sm text-muted-foreground">
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
          <Th align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">{t('chartView.colReturn')}</Th>
          <Th align="right" class="max-sm:min-w-0 max-sm:py-0.5 whitespace-nowrap">{t('chartView.colCumulative')}</Th>
        </Tr>
      </THead>
      <TBody class="max-sm:block">
        {#each buckets as b (b.period)}
          <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
            <Td class="max-sm:col-span-2 max-sm:py-0.5 font-medium break-words">
              {formatPeriod(b.period)}
            </Td>
            <Td align="right" class={cx('max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap', pnlColorClass(b.return))}>
              {formatSignedPercent(b.return)}
            </Td>
            <Td align="right" class="max-sm:min-w-0 max-sm:py-0.5 whitespace-nowrap">
              {formatSignedPercent(b.twr)}
            </Td>
          </Tr>
        {/each}
      </TBody>
    </Table>
  {:else}
    <div class="h-[340px] w-full">
      <!-- {#key} re-inits the chart when the theme flips so the ECharts theme
           object passed below is picked up (svelte-echarts only reads `theme`
           at init time). The palette variant is part of the key too: flipping
           the CVD toggle (D6, K.5c) repaints the gain/loss bars, which this
           chart is the only consumer of via `semantic.positive/negative`. -->
      {#key `${resolved()}:${palette.cvd ? 'cvd' : 'classic'}`}
        <Chart {init} {options} theme={PECULIUM_CHART_THEMES[resolved()]} />
      {/key}
    </div>
  {/if}
{/if}
