<script lang="ts">
  import { RefreshCw } from 'lucide-svelte'
  import { t } from '$lib/i18n/index.svelte'
  import { priceRefresh, refreshPrices } from '$lib/stores/priceRefresh.svelte'
  import Button from '../ui/Button.svelte'
  import { cx } from '../ui/utils'

  /**
   * Global price-freshness control in the app header: always visible on
   * every page, it answers "how fresh are these numbers?" at the app chrome
   * level and doubles as the manual refresh trigger (`refreshPrices` — the
   * same shared path the automatic session refresh, the Fab and the palette
   * use, so every consumer re-syncs through `revision`).
   *
   * Desktop (`lg`+) shows the "Prices as of HH:MM" label next to the glyph;
   * below `lg` it is icon-only, so the `aria-label` carries the same text
   * plus the partial/failure context. A completed-but-degraded outcome
   * (a `partial` job or a failed refresh) tints the control with the
   * `--info` token — never colour alone, the strip below spells out the
   * chips — and the glyph spins while a refresh is in flight.
   */
  const time = $derived.by(() => {
    if (!priceRefresh.finishedAt) return ''
    const d = new Date(priceRefresh.finishedAt)
    return Number.isNaN(d.getTime())
      ? ''
      : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  })

  /** Same partial semantics the stamp had; `failed` only widens the tint. */
  const partial = $derived(priceRefresh.partial)
  const degraded = $derived(partial || priceRefresh.failed)

  const label = $derived(
    priceRefresh.refreshing
      ? t('freshness.refreshing')
      : time
        ? t('freshness.asOf', { time })
        : // No completed refresh yet (e.g. the session one failed): the
          // control is only a trigger, so name the action instead.
          t('quickActions.refreshPrices'),
  )

  // `partialHint` already spells out the timestamp, so pairing it with the
  // label would repeat "Prices as of HH:MM"; a bare failure (no partial
  // outcome) keeps the plain label — the quality strip spells the failure out.
  const ariaLabel = $derived(partial && time ? t('freshness.partialHint', { time }) : label)
</script>

<Button
  variant="ghost"
  size="icon"
  class={cx('lg:w-auto lg:gap-2 lg:px-3', degraded && !priceRefresh.refreshing && 'text-info')}
  aria-label={ariaLabel}
  disabled={priceRefresh.refreshing}
  onclick={() => void refreshPrices({ announceSuccess: true })}
>
  <RefreshCw
    class={cx('h-5 w-5 shrink-0', priceRefresh.refreshing && 'animate-spin')}
    aria-hidden="true"
  />
  <span class="hidden text-sm lg:inline">{label}</span>
</Button>
