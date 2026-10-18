import { createContext } from 'svelte'
import type {
  Asset,
  AssetExposure,
  AssetQuote,
  Price,
  SplitInfo,
} from '$lib/services/api'

/** Editable metadata form (the Data tab's inputs bind into it; provider
 * prefills sync `form.isin` from the layout). */
export interface AssetForm {
  ticker: string
  isin: string
  name: string
  type: string
  currency: string
  exchange: string
  asset_class: string
  price_source: string
}

/**
 * One "Where held" row: this asset as held by ONE of the user's
 * portfolios, derived client-side from `portfolioApi.dashboard()`
 * (`assets: PortfolioAssets[]` → per-portfolio `assets: AssetPerformance[]`,
 * filtered to this asset id with a positive qty — fully-closed holdings are
 * skipped). `invested`/`value`/`gain_loss`/`roi` are in the ASSET's own
 * currency (`currency`), matching the per-asset rows of
 * `GET /dashboard`; no new endpoint is involved.
 */
export interface HeldRow {
  portfolioId: string
  portfolioName: string
  currency: string
  qty: string
  invested: string
  value: string
  gain_loss: string
  roi: string
}

/**
 * Typed context shared by the asset-detail shell (`+layout.svelte`,
 * EPIC K.4b, spec §4.2/§6.3) and its three tab pages (Overview /
 * Exposure / Data).
 *
 * The layout OWNS all data loading: the asset payload, quote, full price
 * history, splits and stored exposure fetched together on mount, the
 * fresh quote/prices refetch when the shell's session price refresh
 * completes, the isolated `portfolioApi.dashboard()` behind the "Where
 * held" rows, and every mutation the old single page performed — the
 * metadata PATCH (and the shared `form` the prefills write into), the
 * exposure saves/prefills/derives behind `openGeoModal`/`openSectorModal`
 * (the modals themselves are mounted by the layout, mirroring K.4a's
 * transaction modal), and the Yahoo-refresh / backfill / delete header
 * actions. The tabs read the same reactive state through the getters
 * below — nothing is ever fetched twice — and hand user intents back
 * through the methods. View-level derivations (chart series, display
 * exposure lists, "Where held" rows) are re-computed inside the tab that
 * renders them.
 *
 * The state members are getters on purpose: they proxy the layout's
 * `$state`, and only reads performed *in the tab* register as that tab's
 * dependencies. Destructuring the context would snapshot the values and
 * silently break reactivity.
 */
export interface AssetPageContext {
  /** Route param (`page.params.id`, defensively optional like before). */
  readonly id: string | undefined
  /** Initial load in flight: the shell shows its own "Loading..." line. */
  readonly loading: boolean
  readonly asset: Asset | null
  readonly quote: AssetQuote | null
  readonly prices: Price[]
  readonly splits: SplitInfo[]
  readonly exposure: AssetExposure | null
  /** Asset currency, `'USD'` fallback while the payload loads. */
  readonly currency: string

  // --- Overview tab: "Where held" (isolated dashboard-derived block)
  /** `null` while the (non-blocking) dashboard fetch hasn't resolved. */
  readonly held: HeldRow[] | null
  /** The dashboard call failed: the block degrades to an unavailable note. */
  readonly heldFailed: boolean

  // --- Header `⋯` actions, also rendered in the Data tab's danger zone
  readonly refreshingMeta: boolean
  refreshFromYahoo(): void
  readonly backfillingHistory: boolean
  backfillHistory(): void
  /** True while an open sync/backfill job is updating this asset's data. */
  readonly dataSyncing: boolean
  /** Opens the layout-mounted delete confirmation. */
  requestDelete(): void

  // --- Data tab: metadata form (PATCH) — the tab binds straight into `form`
  readonly form: AssetForm
  readonly hasChanges: boolean
  /** Metadata save in flight (the exposure saves have their own flags). */
  readonly saving: boolean
  saveAsset(): void

  // --- Exposure tab: rehydrate + open the layout-mounted edit modals
  openGeoModal(): void
  openSectorModal(): void
}

/** `[get, set]` pair — `get` throws if no asset-detail layout provides it. */
export const [getAssetPage, setAssetPage] = createContext<AssetPageContext>()
