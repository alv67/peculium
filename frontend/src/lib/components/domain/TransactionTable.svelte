<script lang="ts">
  import { Pencil } from 'lucide-svelte'
  import { t } from '#lib/i18n/index.svelte.js'
  import { formatCurrency, quoteDivisor } from '#lib/format.js'
  import type { Transaction } from '#lib/services/api.js'
  import Table from '#lib/components/ui/Table.svelte'
  import THead from '#lib/components/ui/THead.svelte'
  import TBody from '#lib/components/ui/TBody.svelte'
  import Tr from '#lib/components/ui/Tr.svelte'
  import Th from '#lib/components/ui/Th.svelte'
  import Td from '#lib/components/ui/Td.svelte'
  import Badge from '#lib/components/ui/Badge.svelte'
  import Button from '#lib/components/ui/Button.svelte'

  /**
   * Transactions table (EPIC E.2): extracted from the portfolio detail page.
   * Monetary columns use the shared right-aligned + `tabular-nums` recipe;
   * `onedit` hands the row back to the caller (usually an edit modal).
   */
  let {
    transactions = [],
    currency = 'USD',
    onedit,
  }: {
    transactions?: Transaction[]
    currency?: string
    onedit: (tx: Transaction) => void
  } = $props()

  function typeVariant(type: Transaction['type']): 'positive' | 'negative' | 'accent' | 'neutral' {
    if (type === 'buy') return 'positive'
    if (type === 'sell') return 'negative'
    if (type === 'dividend') return 'accent'
    return 'neutral'
  }

  /** Cash of one row: buy/sell are `quantity × price / quote_divisor`
   *  (bonds quote per 100 nominal), dividends and fees are absolute. */
  function rowTotal(tx: Transaction): number {
    const gross = Number(tx.quantity) * Number(tx.price)
    return tx.type === 'buy' || tx.type === 'sell' ? gross / quoteDivisor(tx.quote_divisor) : gross
  }

  /** Badge label: the known types reuse the `activity.type*` chip labels;
   *  anything unexpected from the API falls back to the raw value. */
  function typeLabel(type: Transaction['type']): string {
    switch (type) {
      case 'buy':
        return t('activity.typeBuy')
      case 'sell':
        return t('activity.typeSell')
      case 'dividend':
        return t('activity.typeDividend')
      case 'split':
        return t('activity.typeSplit')
      case 'fee':
        return t('activity.typeFee')
      default:
        return type
    }
  }
</script>

<div class="overflow-x-auto">
  <Table aria-label={t('activity.transactions')}>
    <THead>
      <Tr>
        <Th>{t('common.colDate')}</Th>
        <Th>{t('common.colAsset')}</Th>
        <Th>{t('common.colType')}</Th>
        <Th align="right">{t('common.colQty')}</Th>
        <Th align="right">{t('common.colPrice')}</Th>
        <Th align="right">{t('common.colTotal')}</Th>
        <Th align="right">{t('common.colActions')}</Th>
      </Tr>
    </THead>
    <TBody>
      {#each transactions as tx (tx.id)}
        <Tr>
          <Td>{new Date(tx.date).toLocaleDateString()}</Td>
          <Td>
            <span class="font-medium">{tx.asset_ticker}</span>
            <span class="ml-1 text-xs text-muted-foreground">{tx.asset_name}</span>
          </Td>
          <Td>
            <Badge variant={typeVariant(tx.type)}>{typeLabel(tx.type)}</Badge>
          </Td>
          <Td align="right">{tx.quantity}</Td>
          <Td align="right">{formatCurrency(tx.price, currency)}</Td>
          <Td align="right">
            {formatCurrency(rowTotal(tx), currency)}
          </Td>
          <Td align="right">
            <Button
              variant="ghost"
              size="icon"
              aria-label={t('activity.editTransaction')}
              onclick={() => onedit(tx)}
            >
              <Pencil class="h-4 w-4" />
            </Button>
          </Td>
        </Tr>
      {/each}
    </TBody>
  </Table>
</div>
