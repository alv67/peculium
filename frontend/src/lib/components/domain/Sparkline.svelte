<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use } from 'echarts/core'
  import { LineChart } from 'echarts/charts'
  import { GridComponent } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { chartSemanticColors } from '$lib/chartPalette'
  import { PECULIUM_CHART_THEMES } from '$lib/chartTheme'
  import { t } from '$lib/i18n/index.svelte'
  import { resolved } from '$lib/stores/theme.svelte'
  import { cx } from '$lib/components/ui/utils'

  use([LineChart, GridComponent, CanvasRenderer])

  /** Dated sparkline point: `date` is an ISO string (API convention),
   * `value` already numeric — callers convert the decimal strings. */
  export interface SparklinePoint {
    date: string
    value: number
  }

  let {
    /** Flat values or dated points (don't mix the two forms). Fewer than
     * two points render nothing: there is no shape to draw. */
    points = [] as number[],
    /** Line/fill color; defaults to the semantic `marketValue` green. */
    color = undefined as string | undefined,
    /** Subtle area fill under the line. */
    area = true,
    /** Canvas height, as a Tailwind class on the wrapper. */
    heightClass = 'h-10',
    class: className = '',
    /** Accessible name from the caller (e.g. "{name} value trend"); the
     * component falls back to a generic label. */
    ariaLabel = '' as string,
  }: {
    points?: number[] | SparklinePoint[]
    color?: string
    area?: boolean
    heightClass?: string
    class?: string
    ariaLabel?: string
  } = $props()

  const src = $derived(points ?? [])
  const hasLine = $derived(src.length > 1)
  // Dated points get a (hidden) time axis so calendar gaps stay truthful;
  // flat numbers get an index axis. Either way both axes are invisible:
  // a sparkline is the shape only, the card carries the numbers.
  const timed = $derived(hasLine && typeof src[0] !== 'number')
  const data = $derived.by((): number[] | [string, number][] =>
    timed
      ? (src as SparklinePoint[]).map((p) => [p.date, Number(p.value)] as [string, number])
      : (src as number[]).map(Number),
  )
  const xAxis = $derived(
    timed
      ? ({ type: 'time', show: false } as const)
      : ({ type: 'category', show: false, boundaryGap: false } as const),
  )

  // Color re-evaluated on theme flips (the {#key} block below also re-inits
  // the chart with the new ECharts theme), same convention as CapitalChart.
  const lineColor = $derived(color ?? chartSemanticColors(resolved()).marketValue)
  const label = $derived(ariaLabel || t('sparkline.trend'))

  const options = $derived.by((): EChartsOption => ({
    // Zeroed grid: the canvas is the plot, no reserved axis gutters.
    grid: { left: 0, right: 0, top: 2, bottom: 2 },
    xAxis,
    // `scale`: the line uses the full canvas height instead of hugging a
    // zero baseline — that is what makes a trend readable at this size.
    yAxis: { type: 'value', show: false, scale: true },
    series: [
      {
        type: 'line',
        data,
        // Decorative series: no hover/tooltip/symbols (no TooltipComponent
        // is even registered); long series are decimated visually with the
        // `sampling: 'lttb'` pattern the spec §9.2 prescribes.
        silent: true,
        sampling: 'lttb',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 1.5, color: lineColor },
        ...(area ? { areaStyle: { color: lineColor, opacity: 0.1 } } : {}),
      },
    ],
  }))
</script>

{#if hasLine}
  <!-- role="img" + aria-label: the shape is supplementary (the card already
       exposes the value and P/L), but screen readers get a concise summary. -->
  <div class={cx('w-full', heightClass, className)} role="img" aria-label={label}>
    <!-- {#key} re-inits the chart when the theme flips so the ECharts theme
         object passed below is picked up (svelte-echarts only reads `theme`
         at init time). -->
    {#key resolved()}
      <Chart {init} {options} theme={PECULIUM_CHART_THEMES[resolved()]} />
    {/key}
  </div>
{/if}
