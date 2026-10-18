<script lang="ts">
  import { resolve } from '$app/paths'
  import { TriangleAlert } from 'lucide-svelte'
  import { formatCurrency } from '$lib/format'
  import { t } from '$lib/i18n/index.svelte'

  /**
   * Vault-level data-quality strip (EPIC K.3a, redesign spec §6.1/§8.5): a
   * thin row of warning chips rendered only when at least one item is
   * actionable; every chip is a link to the surface that fixes it (currency
   * whitelist for missing FX, Data & Sync for the price feed).
   *
   * Scope note: the spec also sketches "missing sectors/countries" and
   * "stale prices" vault counters, but none of them exists on the `Dashboard`
   * payload today — rendering guesses would be dishonest (the trust principle
   * behind §8.5). They need a backend field first (K.3 fast-follow ask);
   * until then this strip consumes only `summary.fx_missing_count` /
   * `fx_missing_value` plus the outcome of the session price refresh.
   */
  let {
    /** Base currency the excluded FX value is expressed in. */
    currency = 'USD',
    fxMissingCount = 0,
    fxMissingValue = '0',
    /** The session refresh job completed `partial` (some quotes failed). */
    partial = false,
    /** The session refresh request itself failed (prices may be stale). */
    refreshFailed = false,
  }: {
    currency?: string
    fxMissingCount?: number
    fxMissingValue?: string | number
    partial?: boolean
    refreshFailed?: boolean
  } = $props()

  interface Chip {
    id: string
    label: string
    href: string
  }

  const chips = $derived.by((): Chip[] => {
    const out: Chip[] = []
    if (fxMissingCount > 0) {
      out.push({
        id: 'fx',
        label: t('quality.fxMissing', {
          amount: formatCurrency(fxMissingValue, currency),
          count: fxMissingCount,
        }),
        href: resolve('/settings/currencies'),
      })
    }
    if (partial) {
      out.push({ id: 'partial', label: t('quality.refreshPartial'), href: resolve('/admin/health') })
    }
    if (refreshFailed) {
      out.push({ id: 'stale', label: t('quality.refreshFailed'), href: resolve('/admin/health') })
    }
    return out
  })
</script>

{#if chips.length > 0}
  <div role="note" class="flex flex-wrap items-center gap-2">
    {#each chips as chip (chip.id)}
      <!-- The rule can't trace `resolve()` through the derived chips array
           (it reports on the attribute line, so a next-line disable would
           miss); every href above is built with it, as per Button.svelte. -->
      <!-- eslint-disable svelte/no-navigation-without-resolve -->
      <a
        href={chip.href}
        class="focus-ring inline-flex items-center gap-1.5 rounded-full bg-warning/10 px-2.5 py-1 text-xs font-medium text-warning transition-colors duration-fast ease-standard hover:bg-warning/20"
      >
        <TriangleAlert class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        {chip.label}
      </a>
      <!-- eslint-enable svelte/no-navigation-without-resolve -->
    {/each}
  </div>
{/if}
