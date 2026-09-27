<script lang="ts">
  import type { EChartsOption } from 'echarts'
  import { Chart } from 'svelte-echarts'
  import { init, use, type EChartsType } from 'echarts/core'
  import { PieChart } from 'echarts/charts'
  import { TooltipComponent } from 'echarts/components'
  import { CanvasRenderer } from 'echarts/renderers'
  import { formatCurrency, formatPercent } from '$lib/format'
  import { resolvePalette } from '$lib/chartPalette'
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

  use([PieChart, TooltipComponent, CanvasRenderer])

  interface TooltipItem {
    marker: string
    name: string
    value: unknown
  }

  interface AllocationDatum {
    name: string
    value: number
  }

  interface AllocationRow extends AllocationDatum {
    weight: number
  }

  let {
    data = [] as AllocationDatum[],
    currency = 'USD',
    title = t('chartView.nameAllocation'),
    showLegend = false,
    showValue = true,
    showTableToggle = true,
  }: {
    data?: AllocationDatum[]
    currency?: string
    title?: string
    showLegend?: boolean
    /** Hide the formatted amount in the tooltip (e.g. when slices mix currencies). */
    showValue?: boolean
    /** Hide the shared Chart ⇄ Table toggle (EPIC K.5b, spec §9.1);
     * shown by default like in every other chart wrapper. */
    showTableToggle?: boolean
  } = $props()

  // Non-positive rows are dropped and the weights are recomputed over the
  // remaining total, so the visible slices always sum to 100%.
  const rows = $derived.by((): AllocationRow[] => {
    const positive = data.filter((d) => Number(d.value) > 0)
    const total = positive.reduce((acc, d) => acc + Number(d.value), 0)
    return positive.map((d) => ({ name: d.name, value: Number(d.value), weight: (Number(d.value) / total) * 100 }))
  })

  // ── "View as table" (EPIC K.5b, spec §9.1) ──────────────────────────────
  // The table lists the same recomputed `rows` (name / amount / share of the
  // visible total); the amount column follows `showValue`, mirroring the
  // tooltip's mixed-currency muting. Switching unmounts the canvas (out of
  // the a11y tree) and empty data keeps showing the empty state.
  let view = $state<'chart' | 'table'>('chart')
  const showTable = $derived(showTableToggle && view === 'table')

  // The live ECharts instance (bound via `bind:chart`, refreshed on every
  // `{#key}` theme re-init). The tap-outside action dismisses the tooltip
  // through it: on touch there is no hover-out, so a tap would otherwise
  // leave the tooltip pinned on screen (issue #123).
  let chartInstance = $state<EChartsType | undefined>(undefined)

  const palette = $derived(resolvePalette(resolved()))
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
        const row = rows.find((r) => r.name === p.name)
        if (!row) return ''
        const valueLine = showValue
          ? `${t('chartView.colValue')}: <b>${formatCurrency(row.value, currency)}</b><br/>`
          : ''
        return `${p.marker}${p.name}<br/>${valueLine}${t('chartView.colWeight')}: <b>${formatPercent(row.weight)}</b>`
      },
    },
    series: [
      {
        name: title,
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['50%', '50%'],
        avoidLabelOverlap: true,
        label: {
          formatter: '{b}: {d}%',
          fontSize: 11,
          color: labelColor,
          textBorderColor: 'transparent',
          textBorderWidth: 0,
        },
        labelLine: { length: 10, length2: 10 },
        data: rows.map((r) => ({ name: r.name, value: r.value })),
      },
    ],
  }))
</script>

{#if rows.length === 0}
  <div class="flex h-[240px] w-full items-center justify-center text-sm text-muted-foreground">
    {t('chartView.noAllocation')}
  </div>
{:else}
  {#if showTableToggle}
    <div class="mb-2 flex justify-end">
      <ChartTableToggle name={title} bind:view />
    </div>
  {/if}
  {#if showTable}
    <!-- No wrapper scroll container (EPIC K bug-fix): the page is the only
         scroll container. `w-full` + wrapping names keep the table inside the
         card; below `sm` rows collapse to a stacked key–value grid. -->
    <Table class="max-sm:block table-fixed">
      <caption class="sr-only">{t('chartView.caption', { name: title })}</caption>
      <THead class="max-sm:block">
        <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
          <Th class={cx('max-sm:py-0.5 break-words', showValue && 'max-sm:col-span-2')}>
            {t('chartView.colName')}
          </Th>
          {#if showValue}
            <Th align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">{t('chartView.colValue')}</Th>
          {/if}
          <Th align="right" class="max-sm:py-0.5 whitespace-nowrap">{t('chartView.colWeight')}</Th>
        </Tr>
      </THead>
      <TBody class="max-sm:block">
        {#each rows as r (r.name)}
          <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
            <Td class={cx('max-sm:py-0.5 font-medium break-words', showValue && 'max-sm:col-span-2')}>
              {r.name}
            </Td>
            {#if showValue}
              <Td align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">
                {formatCurrency(r.value, currency)}
              </Td>
            {/if}
            <Td align="right" class="max-sm:py-0.5 whitespace-nowrap">{formatPercent(r.weight)}</Td>
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
            <span class="ml-auto text-muted-foreground">{formatPercent(r.weight)}</span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
{/if}
