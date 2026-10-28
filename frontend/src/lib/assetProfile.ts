import type { MessageKey } from '#lib/i18n/index.svelte.js'

/** Field keys the create modal can render. */
export type CreateFieldKey =
  | 'ticker' | 'isin' | 'name' | 'type' | 'currency' | 'exchange' | 'price_source'
  | 'issuer' | 'issuer_country' | 'maturity_date'

/** How the create modal renders a field. */
export type FieldControl =
  | 'ticker' | 'text' | 'asset-type' | 'currency' | 'price-source' | 'country' | 'date'

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

export interface AttributeField {
  key: string
  label: MessageKey
  kind: 'text' | 'number'
}

export interface AssetProfile {
  /** Create-modal fields, in render order. */
  createFields: FieldSpec[]
  /** Type-specific extra fields the create modal renders after them. */
  createAttributes?: AttributeField[]
  /** Fields required before Save is enabled. */
  required: CreateFieldKey[]
  tabs: AssetTab[]
  quoteChips: QuoteChip[]
  /** Create-modal defaults applied when the type changes. */
  defaultPriceSource: 'yahoo' | 'manual' | 'none'
  defaultQuoteDivisor: string
  /** Prefill the code from the ISIN when the code is still empty. */
  codeFromIsin?: boolean
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
  defaultPriceSource: 'yahoo',
  defaultQuoteDivisor: '1',
}

/** Bond (obbligazione): fixed income, manual prices per 100 of nominal, no
 * ticker lookup — the code is usually the ISIN itself. */
const BOND: AssetProfile = {
  createFields: [
    { key: 'isin', label: 'asset.factIsin', control: 'text', hint: 'common.optional' },
    { key: 'name', label: 'chartView.colName', control: 'text', placeholder: 'chartView.colName' },
    { key: 'ticker', label: 'asset.factCode', control: 'text', hint: 'asset.codeHint' },
    { key: 'type', label: 'asset.factType', control: 'asset-type' },
    { key: 'currency', label: 'asset.factCurrency', control: 'currency' },
    { key: 'issuer', label: 'asset.factIssuer', control: 'text' },
    { key: 'issuer_country', label: 'asset.factIssuerCountry', control: 'country' },
    { key: 'maturity_date', label: 'asset.factMaturity', control: 'date' },
    { key: 'price_source', label: 'asset.factPriceSource', control: 'price-source' },
  ],
  createAttributes: [
    { key: 'coupon_rate', label: 'asset.attrCouponRate', kind: 'number' },
    { key: 'rating', label: 'asset.attrRating', kind: 'text' },
  ],
  required: ['ticker', 'name'],
  tabs: ALL_TABS,
  quoteChips: [],
  defaultPriceSource: 'manual',
  defaultQuoteDivisor: '100',
  codeFromIsin: true,
}

const CREATE_HINTS: Record<string, MessageKey> = {
  stock: 'asset.lookupHint',
  etf: 'asset.createHintEtf',
  mutual_fund: 'asset.createHintFund',
  bond: 'asset.createManualHint',
  crypto: 'asset.createHintCrypto',
  commodity: 'asset.createHintCommodity',
  cash: 'asset.createHintCash',
  certificate: 'asset.createHintCertificate',
}

/** One-line help shown in the create modal once a type is chosen. */
export function createHint(type: string | null | undefined): MessageKey {
  return (type && CREATE_HINTS[type]) || 'asset.lookupHint'
}

const PROFILES: Partial<Record<string, AssetProfile>> = { stock: STOCK, bond: BOND }

/** Profile of an asset type; unknown/loading types keep the stock baseline. */
export function assetProfile(type: string | null | undefined): AssetProfile {
  return (type && PROFILES[type]) || STOCK
}
