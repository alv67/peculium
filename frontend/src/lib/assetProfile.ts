import type { MessageKey } from '#lib/i18n/index.svelte.js'

/** Field keys the create modal can render. */
export type CreateFieldKey =
  | 'ticker' | 'isin' | 'name' | 'type' | 'currency' | 'exchange' | 'price_source'

/** How the create modal renders a field. */
export type FieldControl = 'ticker' | 'text' | 'asset-type' | 'currency' | 'price-source'

export interface FieldSpec {
  key: CreateFieldKey
  label: MessageKey
  control: FieldControl
  hint?: MessageKey
  /** Placeholder message; only `name` and `exchange` show one today. */
  placeholder?: MessageKey
}

export type AssetTab = 'overview' | 'exposure' | 'data'
export type QuoteChip = 'change_1d' | 'change_1w' | 'change_1m' | 'change_1y' | 'change_ytd'

export interface AssetProfile {
  /** Create-modal fields, in render order. */
  createFields: FieldSpec[]
  /** Fields required before Save is enabled. */
  required: CreateFieldKey[]
  tabs: AssetTab[]
  quoteChips: QuoteChip[]
}

const ALL_TABS: AssetTab[] = ['overview', 'exposure', 'data']
const ALL_CHIPS: QuoteChip[] = ['change_1d', 'change_1w', 'change_1m', 'change_1y', 'change_ytd']

/** Stock behaviour: the baseline every other type falls back to for now. */
const STOCK: AssetProfile = {
  createFields: [
    { key: 'ticker', label: 'positions.colTicker', control: 'ticker' },
    { key: 'name', label: 'chartView.colName', control: 'text', placeholder: 'chartView.colName' },
    { key: 'isin', label: 'asset.factIsin', control: 'text', hint: 'common.optional' },
    { key: 'type', label: 'asset.factType', control: 'asset-type' },
    { key: 'currency', label: 'asset.factCurrency', control: 'currency' },
    { key: 'exchange', label: 'asset.factExchange', control: 'text', placeholder: 'asset.factExchange' },
    { key: 'price_source', label: 'asset.factPriceSource', control: 'price-source' },
  ],
  required: ['ticker', 'name'],
  tabs: ALL_TABS,
  quoteChips: ALL_CHIPS,
}

const PROFILES: Partial<Record<string, AssetProfile>> = { stock: STOCK }

/** Profile of an asset type; unknown/loading types keep the stock baseline. */
export function assetProfile(type: string | null | undefined): AssetProfile {
  return (type && PROFILES[type]) || STOCK
}
