<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use, type EChartsType } from 'echarts/core'
  import { PieChart } from 'echarts/charts'
  import { TooltipComponent } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { formatPercent } from '$lib/format'
  import type { ExposureRow } from '$lib/services/api'
  import { resolvePalette } from '$lib/chartPalette'
  import { PECULIUM_CHART_THEMES } from '$lib/chartTheme'
  import { dismissTooltipOutside } from '$lib/chartTooltip'
  import { resolved } from '$lib/stores/theme.svelte'
  import { t } from '$lib/i18n/index.svelte'
  import ChartTableToggle from '$lib/components/ui/ChartTableToggle.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'

  use([PieChart, TooltipComponent, CanvasRenderer])

  interface TooltipItem {
    marker: string
    name: string
    value: unknown
  }

  // Shape of a pie data item: real rows plus the optional invisible residual
  // slice used in "open" (complete = false) mode. The extra keys mirror
  // ECharts' PieDataItemOption so the array stays assignable to `data`.
  interface PieSlice {
    name: string
    value: number
    itemStyle?: { opacity: number }
    label?: { show: boolean }
    labelLine?: { show: boolean }
    emphasis?: { disabled: boolean }
    silent?: boolean
  }

  let {
    data = [] as ExposureRow[],
    title = t('chartView.nameDistribution'),
    showLegend = false,
    mute = false,
    // When false the donut is rendered "open": if the passed rows sum below 100
    // a transparent residual slice is appended so visible arcs are proportional
    // to the real percentages (default true = legacy behavior, no residual).
    complete = true,
    // EPIC K.5b: the shared Chart ⇄ Table disclosure (spec §9.1). Hidden by
    // the exposure modals (mute previews next to their own weight grids) and
    // by the asset Exposure tab (which renders the same rows as a legend).
    showTableToggle = true,
  }: {
    data?: ExposureRow[]
    title?: string
    showLegend?: boolean
    mute?: boolean
    complete?: boolean
    showTableToggle?: boolean
  } = $props()

  const rows = $derived(data.filter((r) => Number(r.weight) > 0))

  // Rows are normalized weights (> 0), so their sum is always > 0 here.
  const pieData = $derived.by((): PieSlice[] => {
    const slices: PieSlice[] = rows.map((r) => ({ name: r.name, value: Number(r.weight) }))
    if (!complete) {
      const sum = slices.reduce((acc, s) => acc + s.value, 0)
      if (sum < 100 - 0.5) {
        slices.push({
          name: '',
          value: 100 - sum,
          itemStyle: { opacity: 0 },
          label: { show: false },
          labelLine: { show: false },
          emphasis: { disabled: true },
          silent: true,
        })
      }
    }
    return slices
  })

  // Series palette consumed by both the donut and the HTML legend below;
  // depends on resolved() so colors (and the {#key} re-init) follow the theme.
  const palette = $derived(resolvePalette(resolved()))
  // Pie labels do not inherit the ECharts theme textStyle: without an explicit
  // color they keep the default dark fill + white text border, which is
  // unreadable on a dark card ("outlined in white").
  const labelColor = $derived(PECULIUM_CHART_THEMES[resolved()].textStyle.color)

  // ── "View as table" (EPIC K.5b, spec §9.1) ──────────────────────────────
  // The table lists the same shaped `rows` ({name, weight}) the arcs are
  // drawn from — never the synthetic residual slice; switching unmounts the
  // canvas (out of the a11y tree) and empty data keeps the empty state.
  let view = $state<'chart' | 'table'>('chart')
  const showTable = $derived(showTableToggle && view === 'table')
  const caption = $derived(t('chartView.caption', { name: title }))

  // The live ECharts instance (bound via `bind:chart`, refreshed on every
  // `{#key}` theme re-init). The tap-outside action dismisses the tooltip
  // through it: on touch there is no hover-out, so a tap would otherwise
  // leave the tooltip pinned on screen (issue #123).
  let chartInstance = $state<EChartsType | undefined>(undefined)

  const options = $derived.by((): EChartsOption => ({
    color: palette,
    tooltip: {
      trigger: 'item',
      // Explicit show/dismiss policy (issue #123): 'mousemove|click' is
      // ECharts' default, but pinning it keeps desktop hover behavior
      // identical while making the touch-tap toggle intentional; `hideDelay`
      // lets the tooltip fade shortly after a tap/tap-outside instead of
      // staying pinned on touch, where there is no hover-out event.
      triggerOn: 'mousemove|click',
      hideDelay: 150,
      formatter: (params: unknown) => {
        const p = params as TooltipItem
        // Guard: the synthetic residual slice has an empty name and must never
        // surface in a tooltip.
        if (!p.name) return ''
        return `${p.marker}${p.name}: <b>${formatPercent(Number(p.value))}</b>`
      },
    },
    series: [
      {
        name: title,
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['50%', '50%'],
        avoidLabelOverlap: true,
        label: mute
          ? { show: false }
          : {
              formatter: '{b}: {d}%',
              fontSize: 11,
              color: labelColor,
              textBorderColor: 'transparent',
              textBorderWidth: 0,
            },
        labelLine: mute ? { show: false } : { length: 10, length2: 10 },
        data: pieData,
      },
    ],
  }))
</script>

{#if rows.length === 0}
  <div class="flex h-[280px] w-full items-center justify-center text-sm text-muted-foreground">
    {t('chartView.noDistribution')}
  </div>
{:else}
  {#if showTableToggle}
    <div class="mb-2 flex justify-end">
      <ChartTableToggle name={title} bind:view />
    </div>
  {/if}
  {#if showTable}
    <!-- No wrapper scroll container (EPIC K bug-fix): the page is the only
         scroll container. Two short columns with wrapping names stay inside
         the card; below `sm` each row blockifies into a name/weight pair. -->
    <Table class="max-sm:block table-fixed">
      <caption class="sr-only">{caption}</caption>
      <THead class="max-sm:block">
        <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
          <Th class="max-sm:min-w-0 max-sm:py-0.5 break-words">{t('chartView.colName')}</Th>
          <Th align="right" class="max-sm:py-0.5 whitespace-nowrap">{t('chartView.colWeight')}</Th>
        </Tr>
      </THead>
      <TBody class="max-sm:block">
        {#each rows as r (r.name)}
          <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
            <Td class="max-sm:min-w-0 max-sm:py-0.5 font-medium break-words">{r.name}</Td>
            <Td align="right" class="max-sm:py-0.5 whitespace-nowrap">{formatPercent(Number(r.weight))}</Td>
          </Tr>
        {/each}
      </TBody>
    </Table>
  {:else}
    <div class="h-[240px] w-full" use:dismissTooltipOutside={chartInstance}>
      {#key resolved()}
        <Chart {init} {options} theme={PECULIUM_CHART_THEMES[resolved()]} bind:chart={chartInstance} />
      {/key}
    </div>
    {#if showLegend}
      <div class="mt-2 grid grid-cols-1 gap-x-3 gap-y-1 sm:grid-cols-2">
        {#each rows as r, i (r.name)}
          <div class="flex items-center gap-2 text-xs">
            <span
              class="inline-block h-2.5 w-2.5 shrink-0 rounded-sm"
              style="background-color: {palette[i % palette.length]};"
            ></span>
            <span class="truncate">{r.name}</span>
            <span class="ml-auto text-muted-foreground">{formatPercent(Number(r.weight))}</span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
{/if}