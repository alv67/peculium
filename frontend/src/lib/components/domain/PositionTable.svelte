<script lang="ts">
  import { resolve } from '$app/paths'
  import { t } from '#lib/i18n/index.svelte.js'
  import { formatCurrency, formatPercent } from '#lib/format.js'
  import { pnlColorClass } from '#lib/ui-colors.js'
  import Table from '#lib/components/ui/Table.svelte'
  import THead from '#lib/components/ui/THead.svelte'
  import TBody from '#lib/components/ui/TBody.svelte'
  import Tr from '#lib/components/ui/Tr.svelte'
  import Th from '#lib/components/ui/Th.svelte'
  import Td from '#lib/components/ui/Td.svelte'
  import Badge from '#lib/components/ui/Badge.svelte'

  // Shape-agnostic row consumed by both the dashboard (E.1) and the portfolio
  // detail page (E.2); callers map their API payload onto it. Monetary fields
  // are already expressed in `currency`.
  export interface PositionRow {
    assetId?: string
    ticker: string
    name?: string
    qty?: number
    cost?: number
    value?: number
    realized?: number
    unrealized?: number
    roi?: number
    closed?: boolean
    /** Latest closing price, expressed in `priceCurrency` (the asset's ccy). */
    price?: number
    priceCurrency?: string
  }

  let {
    rows = [] as PositionRow[],
    currency = 'USD',
    linkAssets = false,
    showCost = false,
    showRealized = false,
    showUnrealized = false,
    showPrice = false,
  }: {
    rows?: PositionRow[]
    currency?: string
    linkAssets?: boolean
    showCost?: boolean
    showRealized?: boolean
    showUnrealized?: boolean
    showPrice?: boolean
  } = $props()

  // Closed positions dashes out every cell except realized P/L, which stays
  // visible (portfolio-page convention).
  function money(value: number | undefined, closed = false): string {
    if (closed || value === undefined) return '-'
    return formatCurrency(value, currency)
  }

  // Last price is a per-asset metadata figure (asset currency), so — unlike
  // the portfolio-ccy columns — it also shows for closed positions.
  function price(value: number | undefined, rowCurrency?: string): string {
    if (value === undefined || value === 0) return '-'
    return formatCurrency(value, rowCurrency ?? currency)
  }
</script>

<div class="overflow-x-auto">
  <Table aria-label={t('positions.title')}>
    <THead>
      <Tr>
        <Th>{t('positions.colTicker')}</Th>
        {#if showPrice}
          <Th align="right">{t('common.colPrice')}</Th>
        {/if}
        <Th align="right">{t('common.colQty')}</Th>
        <Th align="right">{t('common.colValue')}</Th>
        {#if showCost}
          <Th align="right">{t('positions.colCost')}</Th>
        {/if}
        {#if showRealized}
          <Th align="right">{t('positions.colRealized')}</Th>
        {/if}
        {#if showUnrealized}
          <Th align="right">{t('positions.colUnrealized')}</Th>
        {/if}
        <Th align="right">{t('positions.colRoi')}</Th>
        <Th>{t('positions.colStatus')}</Th>
      </Tr>
    </THead>
    <TBody>
      {#each rows as row (row.assetId ?? row.ticker)}
        <Tr>
          <Td>
            {#if linkAssets && row.assetId}
              <a
                href={resolve(`assets/${row.assetId}`)}
                class="font-medium text-accent-text hover:underline"
              >
                {row.ticker}
              </a>
            {:else}
              <span class="font-medium">{row.ticker}</span>
            {/if}
            {#if row.name}
              <span class="block text-xs text-muted-foreground">{row.name}</span>
            {/if}
          </Td>
          {#if showPrice}
            <Td align="right">{price(row.price, row.priceCurrency)}</Td>
          {/if}
          <Td align="right">{row.closed || row.qty === undefined ? '-' : row.qty}</Td>
          <Td align="right">{money(row.value, row.closed)}</Td>
          {#if showCost}
            <Td align="right">{money(row.cost, row.closed)}</Td>
          {/if}
          {#if showRealized}
            <Td align="right" class="font-medium {pnlColorClass(row.realized)}">
              {money(row.realized)}
            </Td>
          {/if}
          {#if showUnrealized}
            <Td align="right" class="font-medium {pnlColorClass(row.unrealized)}">
              {money(row.unrealized, row.closed)}
            </Td>
          {/if}
          <Td align="right" class="font-medium {pnlColorClass(row.roi)}">
            {row.closed || row.roi === undefined ? '-' : formatPercent(row.roi)}
          </Td>
          <Td>
            {#if row.closed}
              <Badge variant="neutral">{t('positions.closed')}</Badge>
            {/if}
          </Td>
        </Tr>
      {/each}
    </TBody>
  </Table>
</div>
