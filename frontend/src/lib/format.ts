import { t, type MessageKey } from '$lib/i18n/index.svelte'

const currencySymbols: Record<string, string> = {
  USD: '$',
  EUR: '€',
  GBP: '£',
  CHF: 'CHF',
  JPY: '¥',
  CAD: 'C$',
  AUD: 'A$',
  CNY: '¥',
  SEK: 'kr',
  NOK: 'kr',
  DKK: 'kr',
  PLN: 'zł',
  KRW: '₩',
  INR: '₹',
  BRL: 'R$',
  MXN: 'MX$',
  SGD: 'S$',
  NZD: 'NZ$',
  HKD: 'HK$',
  TRY: '₺',
  RUB: '₽',
  ZAR: 'R',
}

export function currencySymbol(code: string): string {
  return currencySymbols[code] || code
}

export function formatCurrency(amount: number | string, currency: string = 'USD'): string {
  const sym = currencySymbol(currency)
  const val = typeof amount === 'string' ? Number(amount) : amount
  return `${sym}${val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

export function formatPercent(value: number | string): string {
  const val = typeof value === 'string' ? Number(value) : value
  return `${val.toFixed(2)}%`
}

/** Like `formatPercent` but always prefixes an explicit `+` on positive
 * values (e.g. `+3.42%`), used by the dashboard return chart tooltips where
 * the sign carries the meaning. */
export function formatSignedPercent(value: number | string): string {
  const val = typeof value === 'string' ? Number(value) : value
  return `${val > 0 ? '+' : ''}${val.toFixed(2)}%`
}

/** Human-scale a millisecond duration: `842 ms` → `1.4 s` → `2m 5s`;
 * null/0 (unfinished or instant) renders the shared em-dash placeholder. */
export function formatDuration(ms?: number | null): string {
  if (!ms || ms < 1) return '—'
  if (ms < 1000) return `${ms} ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)} s`
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`
}

// ── Localized asset vocabulary (types, classes, price sources) ──────────────
// Raw backend values render through the active locale via `t()`, so chips,
// quick facts, selects and donut slices all follow the interface language.

/** Type values offered by the Type selects (Data tab + create-asset modal).
 * The stored enum is wider — the backend also accepts `cash` — and any value
 * outside the selects still gets a localized label from the tables below. */
export const ASSET_TYPES = ['stock', 'etf', 'bond', 'mutual_fund', 'crypto', 'commodity'] as const

/** Class values offered by the Data-tab Class select. */
export const ASSET_CLASSES = [
  'equity',
  'bond',
  'commodity',
  'currency',
  'crypto',
  'real_estate',
  'mixed',
  'other',
] as const

const ASSET_TYPE_KEYS: Record<string, MessageKey> = {
  stock: 'asset.typeStock',
  etf: 'asset.typeEtf',
  bond: 'asset.typeBond',
  mutual_fund: 'asset.typeMutualFund',
  crypto: 'asset.typeCrypto',
  commodity: 'asset.typeCommodity',
  cash: 'asset.typeCash',
}

const ASSET_CLASS_KEYS: Record<string, MessageKey> = {
  equity: 'asset.classEquity',
  bond: 'asset.classBond',
  commodity: 'asset.classCommodity',
  currency: 'asset.classCurrency',
  crypto: 'asset.classCrypto',
  real_estate: 'asset.classRealEstate',
  mixed: 'asset.classMixed',
  other: 'asset.classOther',
}

const PRICE_SOURCE_KEYS: Record<string, MessageKey> = {
  yahoo: 'asset.priceSourceYahoo',
  manual: 'asset.priceSourceManual',
  none: 'asset.priceSourceNone',
}

/** Resolve a raw backend value to its localized label; values absent from
 * the table fall back to the raw string so nothing ever disappears. The
 * `t()` call reads the reactive locale, so callers re-render on language
 * changes. */
function localizedValue(keys: Record<string, MessageKey>, value: string): string {
  const key = keys[value]
  return key === undefined ? value : t(key)
}

export function assetTypeLabel(type: string): string {
  return localizedValue(ASSET_TYPE_KEYS, type)
}

export function assetClassLabel(cls: string): string {
  return localizedValue(ASSET_CLASS_KEYS, cls)
}

export function priceSourceLabel(source: string): string {
  return localizedValue(PRICE_SOURCE_KEYS, source)
}
