<script lang="ts">
  import TransactionTable from '#lib/components/domain/TransactionTable.svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import EmptyState from '#lib/components/ui/EmptyState.svelte'
  import Input from '#lib/components/ui/Input.svelte'
  import PeriodChips from '#lib/components/ui/PeriodChips.svelte'
  import Select from '#lib/components/ui/Select.svelte'
  import { t } from '#lib/i18n/index.svelte.js'
  import type { Transaction } from '#lib/services/api.js'
  import { getPortfolioPage } from '../context'
  import { hasTransactionFilters, type TransactionFilters } from '../tx-filters'

  /**
   * Activity tab (EPIC K.4a table, K.4c filters): the paginated transactions
   * table (EPIC I.9, #88 — same columns, same edit-pencil flow into the
   * shell-owned form, same Previous/Next footer) with the spec §6.2 filter
   * row bolted on top: `ui/PeriodChips` for the type, a `ui/Select` over the
   * portfolio's registered assets and native from/to date inputs for the
   * range. The whole row is URL state (spec §6.4/§8.2): `ctx.txFilters` is
   * the layout's read of `?type=…&asset=…&from=…&to=…` and `ctx.setTxFilters`
   * writes it back, so filtered views are shareable and back-button-safe;
   * the layout resets the window to the first page and refetches it on every
   * change (pagination total = filtered count). Deleting a row from the edit
   * form is undo-based now (D11) — see `AddTransactionModal`.
   */
  const ctx = getPortfolioPage()
  const currency = $derived(ctx.currency)

  const filters = $derived(ctx.txFilters)
  const hasFilters = $derived(hasTransactionFilters(filters))

  // Filtered-empty view once the refetch settled: the row exists but the
  // active filters match nothing (an unfiltered empty portfolio keeps the
  // classic empty-table look from K.4a).
  const filteredEmpty = $derived(
    hasFilters && ctx.transactions !== null && !ctx.txLoading && ctx.txTotal === 0,
  )

  // Type group: `ui/PeriodChips` doubles as the radio-style filter chip row
  // (§8.2 puts these URL-persisted chips exactly here). `'all'` is the
  // unfiltered sentinel — it never reaches the URL.
  const typeChips = $derived<{ value: string; label: string }[]>([
    { value: 'all', label: t('activity.typeAll') },
    { value: 'buy', label: t('activity.typeBuy') },
    { value: 'sell', label: t('activity.typeSell') },
    { value: 'dividend', label: t('activity.typeDividend') },
    { value: 'split', label: t('activity.typeSplit') },
    { value: 'fee', label: t('activity.typeFee') },
  ])

  // The portfolio's registered assets (holdings, closed ones included —
  // they still have transaction history), deduped and ticker-sorted. A
  // deep-linked asset outside the list gets a placeholder option so the
  // select still tells the truth about the URL.
  const assetOptions = $derived.by(() => {
    const options: { value: string; label: string }[] = []
    for (const h of ctx.summary?.holdings ?? []) {
      if (options.some((o) => o.value === h.asset_id)) continue
      options.push({ value: h.asset_id, label: h.name ? `${h.ticker} — ${h.name}` : h.ticker })
    }
    options.sort((a, b) => a.label.localeCompare(b.label))
    if (filters.assetId && !options.some((o) => o.value === filters.assetId)) {
      options.unshift({ value: filters.assetId, label: t('activity.assetUnknown') })
    }
    return options
  })

  /** One funnel for every control: patch the current filter set and hand it
   * to the layout, which persists it to the URL (replaceState) and — through
   * its URL watcher — resets the window to page 1 and refetches it. */
  function patchFilters(patch: Partial<TransactionFilters>): void {
    ctx.setTxFilters({ ...filters, ...patch })
  }
</script>

{#snippet clearButton()}
  <Button variant="link" size="sm" onclick={() => ctx.setTxFilters({})}>
    {t('activity.clearFilters')}
  </Button>
{/snippet}

<div class="rounded-card border-border bg-surface p-4 shadow-card">
  <h2 class="mb-4 font-semibold">{t('activity.transactions')}</h2>

  <!-- Filter row (spec §6.2/§8.2): chips + native select + native date
       inputs, wrapping on phones; the URL is the single source of truth, so
       these controls render `filters` one-way and only ever write through
       `setTxFilters`. -->
  <div class="mb-4 flex flex-wrap items-center gap-x-4 gap-y-3">
    <PeriodChips
      periods={typeChips}
      value={filters.type ?? 'all'}
      onchange={(v) => patchFilters({ type: v === 'all' ? undefined : (v as Transaction['type']) })}
      ariaLabel={t('activity.typeGroup')}
    />
    <label class="flex w-full min-w-0 items-center gap-2 text-sm text-muted-foreground sm:w-auto">
      {t('activity.asset')}
      <Select
        class="min-w-0 sm:w-48"
        value={filters.assetId ?? ''}
        onchange={(e) => patchFilters({ assetId: e.currentTarget.value || undefined })}
      >
        <option value="">{t('activity.assetAll')}</option>
        {#each assetOptions as opt (opt.value)}
          <option value={opt.value}>{opt.label}</option>
        {/each}
      </Select>
    </label>
    <label class="flex items-center gap-2 text-sm text-muted-foreground">
      {t('activity.from')}
      <Input
        type="date"
        class="w-40"
        value={filters.from ?? ''}
        max={filters.to ?? undefined}
        onchange={(e) => patchFilters({ from: e.currentTarget.value || undefined })}
      />
    </label>
    <label class="flex items-center gap-2 text-sm text-muted-foreground">
      {t('activity.to')}
      <Input
        type="date"
        class="w-40"
        value={filters.to ?? ''}
        min={filters.from ?? undefined}
        onchange={(e) => patchFilters({ to: e.currentTarget.value || undefined })}
      />
    </label>
    {#if hasFilters}
      {@render clearButton()}
    {/if}
  </div>

  {#if filteredEmpty}
    <!-- Server-confirmed zero matches: swap table AND footer for the
         explanation + way out (nothing to paginate). -->
    <EmptyState
      dashed
      title={t('activity.emptyFiltered')}
      description={t('activity.emptyFilteredHint')}
      action={clearButton}
    />
  {:else}
    <TransactionTable
      transactions={ctx.transactions ?? []}
      {currency}
      onedit={ctx.editTransaction}
    />
    <!-- EPIC I.9 (#88): pagination footer with the same "1–20 of 137" range
         label and Previous/Next layout used by the admin health page — the
         total here is the FILTERED count (K.4c). -->
    <div class="mt-3 flex flex-wrap items-center justify-between gap-3 border-t border-border pt-3">
      <span class="text-sm tabular-nums text-muted-foreground">{ctx.txRangeLabel}</span>
      <div class="flex items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          disabled={ctx.txOffset === 0 || ctx.txLoading}
          onclick={() => ctx.gotoTxPage(ctx.txPage - 1)}
        >
          {t('common.previous')}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          disabled={ctx.txOffset + ctx.txLimit >= ctx.txTotal || ctx.txLoading}
          onclick={() => ctx.gotoTxPage(ctx.txPage + 1)}
        >
          {t('common.next')}
        </Button>
      </div>
    </div>
  {/if}
</div>
