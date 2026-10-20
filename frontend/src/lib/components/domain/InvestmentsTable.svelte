<script lang="ts">
  import type { ActiveBreakdown, ClosedBreakdown } from '#lib/services/api.js'
  import { formatCurrency, formatPercent } from '#lib/format.js'
  import { pnlColorClass } from '#lib/ui-colors.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import Card from '#lib/components/ui/Card.svelte'
  import Table from '#lib/components/ui/Table.svelte'
  import THead from '#lib/components/ui/THead.svelte'
  import TBody from '#lib/components/ui/TBody.svelte'
  import Tr from '#lib/components/ui/Tr.svelte'
  import Th from '#lib/components/ui/Th.svelte'
  import Td from '#lib/components/ui/Td.svelte'

  /**
   * "Investments" card (EPIC I.2 breakdown shape): one table with a shared
   * header and one row per group, reused by the dashboard (consolidated
   * totals in the user's base currency) and the portfolio detail (same
   * roll-up in the portfolio currency, EPIC I.6 #85).
   *
   * Active = lots still held: invested / value / gain-loss + % / dividends
   * of the open positions. Closed = sold lots: invested / proceeds /
   * realized + %; the dividends of fully-closed positions are already folded
   * into `proceeds` by the backend, so the Dividends cell is an em-dash.
   * Amounts go through `formatCurrency` in `currency`, percentages through
   * `formatPercent`, and every signed P/L cell is colored with
   * `pnlColorClass` over the right-aligned `tabular-nums` `Table` primitives.
   */
  let {
    active,
    closed,
    currency = 'USD',
    title,
  }: {
    active: ActiveBreakdown
    closed: ClosedBreakdown
    /** Currency the amounts are rendered in (base or portfolio currency). */
    currency?: string
    /** Card heading; also used as the table's aria-label. Falls back to the
     *  localized `investments.title` when omitted (kept reactive so a locale
     *  switch updates the heading in place). */
    title?: string
  } = $props()

  // Derived, not a destructuring default: a default would be evaluated once
  // and never re-run when the locale changes.
  const heading = $derived(title ?? t('investments.title'))
</script>

<Card class="p-4">
  <h2 class="mb-3 font-semibold">{heading}</h2>
  <div class="overflow-x-auto">
    <Table aria-label={heading}>
      <THead>
        <Tr>
          <Th class="sr-only">{t('investments.group')}</Th>
          <Th align="right">{t('investments.invested')}</Th>
          <Th align="right">{t('investments.valueProceeds')}</Th>
          <Th align="right">{t('investments.gainLoss')}</Th>
          <Th align="right">{t('investments.pct')}</Th>
          <Th align="right">{t('investments.dividends')}</Th>
        </Tr>
      </THead>
      <TBody>
        <Tr>
          <Td class="font-medium">{t('investments.active')}</Td>
          <Td align="right">{formatCurrency(active.invested, currency)}</Td>
          <Td align="right">{formatCurrency(active.value, currency)}</Td>
          <Td align="right" class="font-medium {pnlColorClass(active.gain_loss)}">
            {formatCurrency(active.gain_loss, currency)}
          </Td>
          <Td align="right" class={pnlColorClass(active.gain_loss_pct)}>
            {formatPercent(active.gain_loss_pct)}
          </Td>
          <Td align="right">{formatCurrency(active.dividends, currency)}</Td>
        </Tr>
        <Tr>
          <Td class="font-medium">{t('investments.closed')}</Td>
          <Td align="right">{formatCurrency(closed.invested, currency)}</Td>
          <Td align="right">{formatCurrency(closed.proceeds, currency)}</Td>
          <Td align="right" class="font-medium {pnlColorClass(closed.realized)}">
            {formatCurrency(closed.realized, currency)}
          </Td>
          <Td align="right" class={pnlColorClass(closed.realized_pct)}>
            {formatPercent(closed.realized_pct)}
          </Td>
          <Td align="right" class="text-muted-foreground">—</Td>
        </Tr>
      </TBody>
    </Table>
  </div>
</Card>
