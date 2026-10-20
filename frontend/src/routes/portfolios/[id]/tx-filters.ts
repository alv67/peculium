import type { Transaction } from '#lib/services/api.js'

/**
 * Activity-tab filter model + URL query codec (EPIC K.4c, spec §6.2/§6.4/§8.2:
 * "Activity filters are URL-persisted chips").
 *
 * The URL query of `/portfolios/[id]/activity` is the single source of truth
 * for the filters — `?type=sell&asset=<asset-id>&from=YYYY-MM-DD&to=YYYY-MM-DD`
 * — so a filtered view is shareable, survives reloads and deep links, and the
 * back button restores the exact previous view. The layout DERIVES the active
 * filters from `page.url.searchParams` and honours them in every transaction
 * fetch (the backend `total` is the filtered count, so the "1–20 of N" range
 * label stays truthful); the tab page writes them back through
 * `goto(..., { replaceState: true })`.
 *
 * Deliberately dependency-free (pure functions over URL/URLSearchParams) so
 * both the layout and the tab page can share it.
 */
export interface TransactionFilters {
  /** One of the five API types; absent/invalid values never parse in. */
  type?: Transaction['type']
  /** Asset id (uuid). URL key is `asset`; the API param is `asset_id`. */
  assetId?: string
  /** Inclusive lower bound, strict `YYYY-MM-DD` (same spelling the API takes). */
  from?: string
  /** Inclusive upper bound, strict `YYYY-MM-DD`. */
  to?: string
}

/** Query keys of the persisted filter contract (value → URL param name). */
const URL_KEYS: Record<keyof TransactionFilters, string> = {
  type: 'type',
  assetId: 'asset',
  from: 'from',
  to: 'to',
}

const TX_TYPES: readonly string[] = ['buy', 'sell', 'dividend', 'split', 'fee']
/** Only the strict ISO spelling is even considered; the round-trip check in
 * `parseDateParam` then rejects semantically impossible dates too — exactly
 * like the backend's `ParseTransactionDate` format round-trip. */
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

function parseDateParam(value: string): string | undefined {
  if (!DATE_RE.test(value)) return undefined
  const t = new Date(`${value}T00:00:00Z`)
  return !Number.isNaN(t.getTime()) && t.toISOString().slice(0, 10) === value ? value : undefined
}

/** Parse the filters out of a URL query. Unknown types, malformed dates and
 * empty values degrade to "unfiltered" instead of erroring, so a stale or
 * hand-edited link never wedges the tab. */
export function parseTransactionFilters(params: Pick<URLSearchParams, 'get'>): TransactionFilters {
  const type = params.get(URL_KEYS.type) ?? ''
  const assetId = params.get(URL_KEYS.assetId)?.trim() ?? ''
  return {
    type: TX_TYPES.includes(type) ? (type as Transaction['type']) : undefined,
    assetId: assetId || undefined,
    from: parseDateParam(params.get(URL_KEYS.from) ?? ''),
    to: parseDateParam(params.get(URL_KEYS.to) ?? ''),
  }
}

/** Write `filters` into `url`'s query in place (set what is active, delete
 * what is not, so "clear filter" leaves no `type=` residue). */
export function applyTransactionFilters(url: URL, filters: TransactionFilters): void {
  for (const key of Object.keys(URL_KEYS) as (keyof TransactionFilters)[]) {
    const value = filters[key]
    if (value) url.searchParams.set(URL_KEYS[key], value)
    else url.searchParams.delete(URL_KEYS[key])
  }
}

/** Stable identity of a filter set — the layout's refetch trigger compares
 * this so same-value URL churn never double-fetches. FIXED-POSITION join
 * (every dimension is always represented): `{from: X}` and `{to: X}` must
 * not collide, while the values (enum/uuid/date) never contain the
 * separator. */
export function transactionFiltersSignature(filters: TransactionFilters): string {
  return [filters.type ?? '', filters.assetId ?? '', filters.from ?? '', filters.to ?? ''].join('|')
}

/** Whether at least one filter is active (drives "Clear filters" and the
 * filtered-empty state). */
export function hasTransactionFilters(filters: TransactionFilters): boolean {
  return Boolean(filters.type || filters.assetId || filters.from || filters.to)
}
