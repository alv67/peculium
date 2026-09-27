<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use } from 'echarts/core'
  import { LineChart } from 'echarts/charts'
  import {
    DataZoomComponent,
    GridComponent,
    LegendComponent,
    MarkLineComponent,
    TooltipComponent,
  } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { formatCurrency } from '$lib/format'
  import type { PositionPoint, SplitInfo } from '$lib/services/api'
  import { chartSemanticColors } from '$lib/chartPalette'
  import { PECULIUM_CHART_THEMES } from '$lib/chartTheme'
  import { resolved } from '$lib/stores/theme.svelte'
  import { t, type MessageKey } from '$lib/i18n/index.svelte'

  use([
    LineChart,
    DataZoomComponent,
    GridComponent,
    LegendComponent,
    MarkLineComponent,
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
    series = [] as PositionPoint[],
    currency = 'USD',
    splits = [] as SplitInfo[],
  } = $props()

  // Series identities resolve through the dictionary (legend/tooltip): cost
  // basis and market value have dedicated keys, realized reuses the hero
  // line. The names resolve inside the reactive `options` derived, so a
  // locale flip re-labels the chart.
  const LINES = [
    { key: 'cost_basis', nameKey: 'chartView.seriesCostBasis', colorKey: 'costBasis', step: 'end' },
    { key: 'market_value', nameKey: 'chartView.seriesMarketValue', colorKey: 'marketValue', step: undefined },
    { key: 'realized', nameKey: 'hero.realized', colorKey: 'realized', step: 'end' },
  ] as const satisfies ReadonlyArray<{ nameKey: MessageKey; [prop: string]: unknown }>

  // Series/line colors come from the semantic chart tokens, re-evaluated on
  // theme flips (the {#key} block below also re-inits the chart with the new
  // ECharts theme).
  const semantic = $derived(chartSemanticColors(resolved()))

  const options = $derived.by((): EChartsOption => ({
    color: LINES.map((l) => semantic[l.colorKey]),
    tooltip: {
      trigger: 'axis',
      formatter: (params: unknown) => {
        const raw = Array.isArray(params) ? params : [params]
        const rows = raw as TooltipRow[]
        const date = rows[0]?.axisValue
        const lines = rows
          .filter((p) => p.value != null)
          .map((p) => {
            const v = Array.isArray(p.value) ? p.value[1] : p.value
            return `${p.marker}${p.seriesName}: <b>${formatCurrency(Number(v), currency)}</b>`
          })
        const dateLabel = date ? `<div>${new Date(date).toLocaleDateString()}</div>` : ''
        return `${dateLabel}${lines.join('<br/>')}`
      },
    },
    legend: {
      data: LINES.map((l) => t(l.nameKey)),
      top: 0,
    },
    grid: { left: 48, right: 16, top: 40, bottom: 52 },
    dataZoom: [
      { type: 'inside', xAxisIndex: 0 },
      { type: 'slider', xAxisIndex: 0, bottom: 0 },
    ],
    xAxis: {
      type: 'time',
      axisLabel: {
        fontSize: 11,
        formatter: (value: string | number) => new Date(value).toLocaleDateString(),
      },
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 11 },
    },
    series: LINES.map((l) => ({
      name: t(l.nameKey),
      type: 'line',
      data: series.map((p) => [p.date, Number(p[l.key])]),
      sampling: 'lttb',
      smooth: l.step === 'end' ? false : true,
      step: l.step,
      showSymbol: true,
      symbolSize: 4,
      lineStyle: { width: 2 },
      ...(l.key === 'market_value' && splits.length > 0
        ? {
            markLine: {
              symbol: 'none',
              silent: true,
              lineStyle: { type: 'dashed', color: semantic.splitMarkLine, width: 1 },
              label: {
                show: true,
                position: 'insideEndTop',
                formatter: '{b}',
                color: semantic.splitMarkLine,
                fontSize: 10,
              },
              data: splits.map((s) => ({
                xAxis: new Date(s.date).getTime(),
                name: t('chartView.splitRatio', { ratio: s.ratio }),
              })),
            },
          }
        : {}),
    })),
  }))
</script>

{#if series.length === 0}
  <div class="flex h-[340px] w-full items-center justify-center text-sm text-muted-foreground">
    {t('chartView.noData')}
  </div>
{:else}
  <div class="h-[340px] w-full">
    {#key resolved()}
      <Chart {init} {options} theme={PECULIUM_CHART_THEMES[resolved()]} />
    {/key}
  </div>
{/if}
