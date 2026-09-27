<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use } from 'echarts/core'
  import { LineChart } from 'echarts/charts'
  import {
    DataZoomComponent,
    GridComponent,
    MarkLineComponent,
    TooltipComponent,
  } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { formatCurrency } from '$lib/format'
  import type { SplitInfo } from '$lib/services/api'
  import { chartSemanticColors, resolvePalette } from '$lib/chartPalette'
  import { PECULIUM_CHART_THEMES } from '$lib/chartTheme'
  import { resolved } from '$lib/stores/theme.svelte'
  import { t } from '$lib/i18n/index.svelte'

  use([LineChart, DataZoomComponent, GridComponent, MarkLineComponent, TooltipComponent, CanvasRenderer])

  interface PricePoint {
    date: string
    close: string
  }

  interface TooltipRow {
    marker: string
    seriesName: string
    value: unknown
    axisValue?: string | number
  }

  let {
    series = [] as PricePoint[],
    currency = 'USD',
    zoomStart = null as string | 'MAX' | null,
    onDataZoom = null as (() => void) | null,
    splits = [] as SplitInfo[],
  } = $props()

  const zoomOption = $derived.by(() => {
    if (zoomStart === 'MAX') return { start: 0, end: 100 }
    if (zoomStart && series.length > 0) {
      const times = series.map((p) => new Date(p.date).getTime())
      const min = Math.min(...times)
      const max = Math.max(...times)
      const start = new Date(zoomStart).getTime()
      const startPct =
        max === min ? 0 : Math.max(0, Math.min(100, ((start - min) / (max - min)) * 100))
      return { start: startPct, end: 100 }
    }
    return {}
  })

  // Theme-aware colors: the single close line uses the `chart-1` series
  // token and split marks the semantic split color, both re-evaluated on
  // theme flips (the {#key} block below also re-inits the chart).
  const lineColor = $derived(resolvePalette(resolved())[0])
  const splitMarkLine = $derived(chartSemanticColors(resolved()).splitMarkLine)

  const options = $derived.by((): EChartsOption => ({
    color: [lineColor],
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
    grid: { left: 48, right: 16, top: 24, bottom: 52 },
    dataZoom: [
      {
        type: 'inside',
        xAxisIndex: 0,
        ...zoomOption,
      },
      {
        type: 'slider',
        xAxisIndex: 0,
        bottom: 0,
        ...zoomOption,
      },
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
    series: [
      {
        name: t('chartView.seriesClose'),
        type: 'line',
        data: series.map((p) => [p.date, parseFloat(p.close)]),
        connectNulls: true,
        symbol: 'none',
        sampling: 'lttb',
        lineStyle: { width: 2 },
        ...(splits.length > 0
          ? {
              markLine: {
                symbol: 'none',
                silent: true,
                lineStyle: { type: 'dashed', color: splitMarkLine, width: 1 },
                label: {
                  show: true,
                  position: 'insideEndTop',
                  formatter: '{b}',
                  color: splitMarkLine,
                  fontSize: 10,
                },
                data: splits.map((s) => ({
                  xAxis: new Date(s.date).getTime(),
                  name: t('chartView.splitRatio', { ratio: s.ratio }),
                })),
              },
            }
          : {}),
      },
    ],
  }))
</script>

{#if series.length === 0}
  <div class="flex h-[340px] w-full items-center justify-center text-sm text-muted-foreground">
    {t('chartView.noPriceData')}
  </div>
{:else}
  <div class="h-[340px] w-full">
    {#key resolved()}
      <Chart {init} {options} theme={PECULIUM_CHART_THEMES[resolved()]} notMerge={false} ondatazoom={() => onDataZoom?.()} />
    {/key}
  </div>
{/if}