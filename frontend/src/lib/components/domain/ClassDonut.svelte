<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart, type ECMouseEvent } from 'svelte-echarts'
  import { init, use, type EChartsType } from 'echarts/core'
  import { PieChart } from 'echarts/charts'
  import { LegendComponent, TooltipComponent } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { assetClassLabel, formatCurrency, formatPercent } from '$lib/format'
  import type { AssetClassSlice } from '$lib/services/api'
  import { chartSemanticColors, resolvePalette } from '$lib/chartPalette'
  import { PECULIUM_CHART_THEMES } from '$lib/chartTheme'
  import { dismissTooltipOutside } from '$lib/chartTooltip'
  import { resolved } from '$lib/stores/theme.svelte'
  import { t } from '$lib/i18n/index.svelte'
  import { cx } from '$lib/components/ui/utils'
  import ChartTableToggle from '$lib/components/ui/ChartTableToggle.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'

  use([PieChart, LegendComponent, TooltipComponent, CanvasRenderer])

  interface TooltipItem {
    marker: string
    name: string
    value: unknown
  }

  let {
    data = [] as AssetClassSlice[],
    currency = 'USD',
    label = undefined as string | undefined,
    showTableToggle = true,
    onDrill = undefined,
  }: {
    data?: AssetClassSlice[]
    currency?: string
    /** Optional heading rendered above the donut. */
    label?: string
    /** Hide the shared Chart ⇄ Table toggle (EPIC K.5b, spec §9.1);
     * shown by default like in every other chart wrapper. */
    showTableToggle?: boolean
    /** Drill-down callback (EPIC K.5, spec §6.5): when set, clicking a slice
     * opens the caller's drill panel with the raw class key (the friendly
     * label is only for display); slices also get a pointer cursor. Unset =
     * the donut stays a plain visual with the default cursor. */
    onDrill?: (classKey: string) => void
  } = $props()

  // Backend class keys are mapped to localized labels through the shared
  // `assetClassLabel` helper; zero/negative weights are dropped.
  const rows = $derived(data.filter((r) => Number(r.weight) > 0))
  function isOther(cls: string): boolean {
    return cls.toLowerCase() === 'other'
  }

  // The live ECharts instance (bound via `bind:chart`, refreshed on every
  // `{#key}` theme re-init). Needed to dismiss the tooltip imperatively:
  // on touch there is no hover-out, so a tap would otherwise leave the
  // tooltip pinned above the drill sheet (issue #123).
  let chartInstance = $state<EChartsType | undefined>(undefined)

  // Drill-down click (EPIC K.5, spec §6.5): the svelte-echarts wrapper
  // forwards the ECharts instance `click` event through its `onclick` prop,
  // so the binding survives the `{#key}` theme re-init (the handlers are
  // re-registered on every fresh instance). The `componentType` guard drops
  // legend clicks, and the pie's data array is exactly `rows`, so the
  // `dataIndex` maps back 1:1 to the raw class key the drill endpoint wants.
  function handleSliceClick(event: ECMouseEvent): void {
    if (!onDrill || event.componentType !== 'series') return
    const row = rows[event.dataIndex]
    if (row) {
      // Dismiss the tooltip the same tap popped up before opening the
      // drill panel, so it never lingers over the sheet (issue #123).
      chartInstance?.dispatchAction({ type: 'hideTip' })
      onDrill(row.class)
    }
  }

  // ── "View as table" (EPIC K.5b, spec §9.1) ──────────────────────────────
  // The table lists the same shaped `rows` the slices are drawn from; the
  // switch unmounts the canvas (out of the a11y tree) and empty data keeps
  // showing the empty state instead of an empty table.
  let view = $state<'chart' | 'table'>('chart')
  const showTable = $derived(showTableToggle && view === 'table')
  const caption = $derived(
    label ? t('chartView.caption', { name: label }) : t('chartView.captionGeneric'),
  )

  // Palette and "Other" grey come from the chart tokens and are re-evaluated
  // on theme flips; the {#key} block also re-inits the chart with the new
  // ECharts theme.
  const palette = $derived(resolvePalette(resolved()))
  const otherColor = $derived(chartSemanticColors(resolved()).other)
  // Pie labels do not inherit the ECharts theme textStyle: without an explicit
  // color they keep the default dark fill + white text border, which is
  // unreadable on a dark card ("outlined in white").
  const labelColor = $derived(PECULIUM_CHART_THEMES[resolved()].textStyle.color)

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
        const row = rows.find((r) => assetClassLabel(r.class) === p.name)
        if (!row) return ''
        return `${p.marker}${p.name}<br/>${t('chartView.colValue')}: <b>${formatCurrency(row.value, currency)}</b><br/>${t('chartView.colWeight')}: <b>${formatPercent(row.weight)}</b>`
      },
    },
    legend: rows.length > 0 && rows.length <= 6
      ? { bottom: 0, type: 'scroll', textStyle: { fontSize: 11 } }
      : undefined,
    series: [
      {
        name: t('chartView.seriesClassAllocation'),
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['50%', '50%'],
        avoidLabelOverlap: true,
        // Drilled donuts invite the click with a pointer cursor (K.5);
        // plain ones keep the default arrow.
        cursor: onDrill ? 'pointer' : 'default',
        label: {
          formatter: '{b}: {d}%',
          fontSize: 11,
          color: labelColor,
          textBorderColor: 'transparent',
          textBorderWidth: 0,
        },
        labelLine: { length: 10, length2: 10 },
        // Slice angles come from the amounts, so they stay truthful even when
        // the caller sends weights computed over a different base.
        data: rows.map((r) => ({
          name: assetClassLabel(r.class),
          value: Number(r.value),
          // The aggregated "other" bucket is muted in grey like in the
          // sibling donut charts.
          itemStyle: isOther(r.class) ? { color: otherColor } : undefined,
        })),
      },
    ],
  }))
</script>

{#if label || (showTableToggle && rows.length > 0)}
  <div
    class={cx(
      'mb-3 flex flex-wrap items-center gap-2',
      label ? 'justify-between' : 'justify-end',
    )}
  >
    {#if label}
      <h3 class="font-semibold">{label}</h3>
    {/if}
    {#if showTableToggle && rows.length > 0}
      <ChartTableToggle name={label} bind:view />
    {/if}
  </div>
{/if}
{#if rows.length === 0}
  <div class="flex h-[280px] w-full items-center justify-center text-sm text-muted-foreground">
    {t('chartView.noClassAllocation')}
  </div>
{:else if showTable}
  <!-- No wrapper scroll container (EPIC K bug-fix): the page is the only
       scroll container. `w-full` + wrapping class names keep the table inside
       the card; below `sm` rows collapse to a stacked key–value grid. -->
  <Table class="max-sm:block table-fixed">
    <caption class="sr-only">{caption}</caption>
    <THead class="max-sm:block">
      <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
        <Th class="max-sm:col-span-2 max-sm:py-0.5 break-words">{t('chartView.colName')}</Th>
        <Th align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">{t('chartView.colValue')}</Th>
        <Th align="right" class="max-sm:py-0.5 whitespace-nowrap">{t('chartView.colWeight')}</Th>
      </Tr>
    </THead>
    <TBody class="max-sm:block">
      {#each rows as r (r.class)}
        <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
          <!-- Localized class label, same mapping the slice names use. -->
          <Td class="max-sm:col-span-2 max-sm:py-0.5 font-medium break-words">
            {assetClassLabel(r.class)}
          </Td>
          <Td align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">
            {formatCurrency(r.value, currency)}
          </Td>
          <Td align="right" class="max-sm:py-0.5 whitespace-nowrap">{formatPercent(r.weight)}</Td>
        </Tr>
      {/each}
    </TBody>
  </Table>
{:else}
  <div class="h-[280px] w-full" use:dismissTooltipOutside={chartInstance}>
    {#key resolved()}
      <!-- `onclick` is the svelte-echarts wrapper's ECharts event prop (it
           registers `chart.on('click')` at init), so the drill-down binding
           is re-created together with each re-initialised instance;
           `bind:chart` keeps `chartInstance` pointed at the live instance
           so the tooltip can be dismissed imperatively (issue #123). -->
      <Chart
        {init}
        {options}
        theme={PECULIUM_CHART_THEMES[resolved()]}
        onclick={handleSliceClick}
        bind:chart={chartInstance}
      />
    {/key}
  </div>
{/if}
