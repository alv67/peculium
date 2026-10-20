<script lang="ts">
  import { t } from '#lib/i18n/index.svelte.js'
  import PositionTable, { type PositionRow } from '#lib/components/domain/PositionTable.svelte'
  import { getPortfolioPage } from '../context'

  /**
   * Positions tab (EPIC K.4a): the full holdings table exactly as it lived
   * on the old single page — `summary.holdings` mapped onto `PositionRow`
   * (amounts already in the portfolio currency), tickers linking to the
   * asset detail, closed positions badged with the "-" dash-out, and the
   * same "No positions" line when nothing is held. Rows come straight from
   * the shell's shared summary state.
   */
  const ctx = getPortfolioPage()
  const currency = $derived(ctx.currency)

  const positionRows = $derived<PositionRow[]>(
    (ctx.summary?.holdings ?? []).map((h) => ({
      assetId: h.asset_id,
      ticker: h.ticker,
      name: h.name,
      qty: Number(h.qty),
      cost: Number(h.cost),
      value: Number(h.value_pf),
      realized: Number(h.realized),
      unrealized: Number(h.unrealized),
      roi: Number(h.roi),
      closed: h.closed,
      price: Number(h.last_close) > 0 ? Number(h.last_close) : undefined,
      priceCurrency: h.currency,
    })),
  )
</script>

{#if positionRows.length > 0}
  <div class="rounded-card border-border bg-surface p-4 shadow-card">
    <h2 class="mb-4 font-semibold">{t('positions.title')}</h2>
    <PositionTable
      rows={positionRows}
      {currency}
      linkAssets
      showCost
      showRealized
      showUnrealized
      showPrice
    />
  </div>
{:else}
  <p class="text-sm text-muted-foreground">{t('positions.noPositions')}</p>
{/if}
