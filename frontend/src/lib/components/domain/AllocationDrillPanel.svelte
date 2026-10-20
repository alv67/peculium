<script lang="ts">
  import { untrack } from 'svelte'
  import { resolve } from '$app/paths'
  import { formatCurrency, formatPercent } from '#lib/format.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import { viewport } from '#lib/stores/viewport.svelte.js'
  import type { AllocationDrill, AllocationDrillDim } from '#lib/services/api.js'
  import AsyncCard from '#lib/components/ui/AsyncCard.svelte'
  import Drawer from '#lib/components/ui/Drawer.svelte'
  import EmptyState from '#lib/components/ui/EmptyState.svelte'
  import Sheet from '#lib/components/ui/Sheet.svelte'
  import Table from '#lib/components/ui/Table.svelte'
  import THead from '#lib/components/ui/THead.svelte'
  import TBody from '#lib/components/ui/TBody.svelte'
  import Tr from '#lib/components/ui/Tr.svelte'
  import Th from '#lib/components/ui/Th.svelte'
  import Td from '#lib/components/ui/Td.svelte'

  /**
   * Allocation drill-down panel (EPIC K.5, spec §6.5, decision D4): the
   * read-only list of the contributing assets behind one allocation bucket,
   * opened when a chart slice/bar is clicked. Renders as the right-side
   * `ui/Drawer` at ≥ `lg` and the bottom `ui/Sheet` below it (the same
   * viewport split as the rest of the app), controlled exactly like both
   * primitives: the caller owns `open`/`dim`/`key`/`title`, we only call
   * `onClose`.
   *
   * Data comes from `fetcher` — one drill endpoint call per bucket, so the
   * page never pre-loads every slice. It refetches when the panel opens and
   * whenever `dim`/`key` change while it is open (clicking another slice
   * swaps the content in place); a monotonic request id keeps late responses
   * from overwriting a newer bucket. The four UI states follow the
   * `ui/AsyncCard` conventions (skeleton / one-line error + Retry / empty /
   * data), and the table mirrors the chart table views: full grid on desktop,
   * stacked key–value rows below `sm` — no nested scroll area anywhere (the
   * Drawer/Sheet body is the only scroll container of the panel).
   */
  let {
    open,
    onClose,
    title,
    dim,
    key,
    fetcher,
    currencyHint = undefined

    /** Called on Esc, backdrop click and the ✕ (forwarded to Drawer/Sheet). */
    /** Friendly label of the bucket, as the calling chart displays it. */
    /** Raw bucket dimension and identifier as the backend knows them. */
    /** One bucket fetch: `portfolioApi.allocationDrill` / `dashboardAllocationDrill`. */
    /** Fallback currency while the payload is absent/older than the backend. */
  }: { 
    open: boolean;
    onClose: () => void;
    title: string;
    dim: AllocationDrillDim;
    key: string;
    fetcher: (dim: AllocationDrillDim, key: string) => Promise<AllocationDrill>;
    currencyHint?: string
  } = $props()

  let data = $state<AllocationDrill | null>(null)
  let loading = $state(false)
  let error = $state<string | null>(null)

  // Monotonic request id (same last-write-wins pattern as the page fetches):
  // only the newest load may write, so a slow response for a previous bucket
  // can never land on top of the current one.
  let req = 0
  async function load(): Promise<void> {
    const id = ++req
    loading = true
    error = null
    try {
      const res = await fetcher(dim, key)
      if (id === req) data = res
    } catch (e) {
      if (id === req) {
        data = null
        error = e instanceof Error ? e.message : String(e)
      }
    } finally {
      if (id === req) loading = false
    }
  }

  // Fetch on open, and refetch when the caller swaps the bucket while the
  // panel stays mounted (clicking another slice). `untrack` keeps the fetcher
  // call itself out of the effect's dependency set; only `open`/`dim`/`key`
  // re-trigger it.
  $effect(() => {
    void dim;
    void key;

    if (!open) return;

    untrack(() => void load());
  });

  // The drill payload carries its own currency (portfolio or base currency,
  // mirroring the allocation endpoint that produced the bucket); the hint is
  // only the fallback while loading or against an older backend.
  const currency = $derived(data?.currency || currencyHint || 'USD');

  const assets = $derived(data?.assets ?? []);
  const total = $derived(Number(data?.total ?? 0));

  /** Asset's share of the slice: contribution ÷ bucket total. Guarded so a
   * zero/NaN total (empty bucket, legacy payload) never renders `NaN%`. */
  function shareOfSlice(contribution: string): string {
    if (!(total > 0)) return '—';
    return formatPercent(Number(contribution) / total * 100);
  }
</script>

{#if viewport.isDesktop}
  <Drawer
    open={open}
    onClose={onClose}
    title={title}
    closeLabel={t('common.close')}
  >{@render body()}</Drawer>
{:else}
  <Sheet
    open={open}
    onClose={onClose}
    title={title}
    closeLabel={t('common.close')}
  >{@render body()}</Sheet>
{/if}

{#snippet body()}
  <AsyncCard
    loading={loading}
    error={error}
    empty={assets.length === 0}
    onRetry={() => void load()}
    retryLabel={t('drill.retry')}
  >
    {#snippet emptyContent()}
      <EmptyState dashed title={t('drill.empty')} />
    {/snippet}

    <!-- Rows arrive sorted by descending contribution from the backend;
         re-rendering in that order keeps the biggest participant on top. -->
    <p class="mb-3 text-sm text-muted-foreground">
      {t('drill.contributingAssets', { count: assets.length })}
    </p>
    <Table class="max-sm:block table-fixed">
      <caption class="sr-only">{t('drill.caption', { name: title })}</caption>
      <THead class="max-sm:block">
        <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
          <Th class="max-sm:col-span-2 max-sm:py-0.5 break-words">{t('drill.colAsset')}</Th>
          <Th align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">{t('chartView.colValue')}</Th>
          <Th align="right" class="max-sm:py-0.5 whitespace-nowrap">{t('chartView.colWeight')}</Th>
          <Th align="right" class="max-sm:py-0.5 whitespace-nowrap">{t('drill.colContribution')}</Th>
          <Th align="right" class="max-sm:py-0.5 whitespace-nowrap">{t('drill.colShare')}</Th>
        </Tr>
      </THead>
      <TBody class="max-sm:block">
        {#each assets as a (a.asset_id)}
          <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
            <Td class="max-sm:col-span-2 max-sm:py-0.5 font-medium min-w-0">
              <a
                href={resolve(`assets/${a.asset_id}`)}
                class="text-accent-text hover:underline"
              >{a.ticker}</a>
              <span class="block truncate text-xs font-normal text-muted-foreground">{a.name}</span>
            </Td>
            <Td align="right" class="max-sm:min-w-0 max-sm:py-0.5 max-sm:text-left whitespace-nowrap">
              {formatCurrency(a.value, currency)}
            </Td>
            <Td align="right" class="max-sm:py-0.5 whitespace-nowrap">{formatPercent(a.weight)}</Td>
            <Td align="right" class="max-sm:py-0.5 whitespace-nowrap">
              {formatCurrency(a.contribution, currency)}
            </Td>
            <Td align="right" class="max-sm:py-0.5 whitespace-nowrap">{shareOfSlice(a.contribution)}</Td>
          </Tr>
        {/each}
      </TBody>
    </Table>
  </AsyncCard>
{/snippet}
