import { t, type MessageKey } from '#lib/i18n/index.svelte.js'

/**
 * Shared status/type vocabulary of the admin "Data & Sync" view (#226): ONE
 * status→badge map and the job type/target label maps, used by the jobs
 * table, the events table and `JobDetailPanel.svelte` so every consumer
 * renders the same badge and the same localized labels.
 */

/** Badge tones the status map may return (`ui/Badge.svelte` variants). */
type BadgeVariant = 'neutral' | 'accent' | 'positive' | 'negative' | 'warning'

// The badge is the one backend value we localise: known statuses map to a
// Badge variant + dictionary label (so `running` reads accent, not red), the
// 429 heuristic covers rate-limited codes, anything unexpected falls back to
// the raw value in the negative tone. The label text is always present.
const STATUS_BADGES: Record<string, { variant: BadgeVariant; key: MessageKey }> = {
  done: { variant: 'positive', key: 'jobs.statusDone' },
  success: { variant: 'positive', key: 'health.statusSuccess' },
  running: { variant: 'accent', key: 'jobs.statusRunning' },
  queued: { variant: 'neutral', key: 'jobs.statusQueued' },
  partial: { variant: 'warning', key: 'jobs.statusPartial' },
  failed: { variant: 'negative', key: 'jobs.statusFailed' },
  failure: { variant: 'negative', key: 'health.statusFailure' },
  rate_limited: { variant: 'warning', key: 'health.statusRateLimited' },
}

export function statusBadge(status: string): { variant: BadgeVariant; label: string } {
  if (status.includes('429')) return { variant: 'warning', label: t('health.statusRateLimited') }
  const known = STATUS_BADGES[status]
  if (known) return { variant: known.variant, label: t(known.key) }
  return { variant: 'negative', label: status }
}

export const JOB_TYPE_KEYS: Record<string, MessageKey> = {
  price_refresh: 'jobs.typePriceRefresh',
  history_backfill: 'jobs.typeHistoryBackfill',
  asset_sync: 'jobs.typeAssetSync',
  meta_backfill: 'jobs.typeMetaBackfill',
  splits_fetch: 'jobs.typeSplitsFetch',
}

export const JOB_TARGET_KEYS: Record<string, MessageKey> = {
  asset: 'jobs.targetAsset',
  portfolio: 'jobs.targetPortfolio',
  global: 'jobs.targetGlobal',
}

/** Localize a raw backend enum value; unknown values show the raw string. */
export function labeled(keys: Record<string, MessageKey>, value: string | undefined) {
  if (!value) return ''
  const key = keys[value]
  return key ? t(key) : value
}
