const BASE_URL = '/api/v1'

// In-memory GET cache with TTL (~60s). Module-level so it survives component
// mounts/unmounts and page navigations within the SPA, avoiding refetching
// heavy endpoints (dashboard, summaries, history) on every mount.
const CACHE_TTL_MS = 60_000

interface CacheEntry {
  data: unknown
  expiresAt: number
}

const getCache = new Map<string, CacheEntry>()

/** Drop the in-memory GET cache. Needed after a background job (e.g. the
 * asset sync) changes server data without a request that would clear it. */
export function clearGetCache(): void {
  getCache.clear()
}

// Deep-copy cached data so callers never share mutable references with the
// cache entry (a caller mutating a response would otherwise poison it).
// Falls back to JSON round-trip, then to the raw reference: it must never
// break the request.
function cloneCached<T>(data: unknown): T {
  try {
    if (typeof structuredClone === 'function') {
      return structuredClone(data) as T
    }
  } catch {
    // structuredClone unavailable/failed — fall through to JSON copy
  }
  try {
    return JSON.parse(JSON.stringify(data)) as T
  } catch {
    // Last resort: return the reference rather than failing the request.
    return data as T
  }
}

export interface User {
  id: string
  email: string
  name: string
  /** `owner` | `admin` | `editor` | `viewer`; `owner`/`admin` are admin-equivalent. */
  role: string
  /** `active` | `pending` | `disabled`; non-active accounts cannot sign in. */
  status: string
  /** User's base currency for consolidated views (EPIC I.1, default "EUR"). */
  base_currency: string
  created_at: string
}

/** One row of `GET /admin/users`: the full account record as stored,
 * including the admin-managed `role`/`status` and both timestamps. */
export interface AdminUser {
  id: string
  email: string
  name: string
  role: string
  status: string
  base_currency: string
  created_at: string
  updated_at: string
}

/** Singleton server settings row (`GET/PATCH /admin/settings`). */
export interface ServerSettings {
  auto_approve_registrations: boolean
  updated_at: string
}

export interface Portfolio {
  id: string
  user_id: string
  name: string
  description: string
  currency: string
  created_at: string
  updated_at: string
}

export interface Asset {
  id: string
  ticker: string
  isin: string
  name: string
  type: string
  asset_class: string
  country: string
  currency: string
  exchange?: string
  sector?: string
  industry?: string
  price_source?: string
}

export interface AssetQuote {
  currency: string
  last_close: string
  last_date: string
  change_1d: string
  change_1w: string
  change_1m: string
  change_1y: string
  change_ytd: string
  has_data: boolean
}

// Body accettato da PATCH /assets/{id}. Campi omessi = invariati; stringa
// vuota = svuota (isin, country, exchange, sector, industry).
export interface AssetPatch {
  ticker?: string
  isin?: string
  name?: string
  type?: string
  asset_class?: string
  country?: string
  currency?: string
  exchange?: string
  sector?: string
  industry?: string
  price_source?: string
}

export interface Price {
  id: string
  asset_id: string
  date: string
  open: string
  high: string
  low: string
  close: string
  volume: number
  source: string
  created_at: string
}

export interface ExposureRow {
  name: string
  weight: string
}

/** Persisted provenance of one exposure dimension: which source currently
 * owns the stored weights and when they were last written. */
export interface ExposureProvenance {
  source: string
  updated_at: string
}

export interface AssetExposure {
  countries: ExposureRow[]
  regions: ExposureRow[]
  sectors: ExposureRow[]
  isin?: string
  /** Per-dimension persisted provenance, keyed by dimension name
   * ('countries' | 'regions' | 'sectors'). Only persisted dimensions are
   * present (each key omitted when empty), and the whole field is absent
   * when nothing was ever saved. Fetch/prefill endpoints never include it:
   * their preview is not persisted yet. */
  provenance?: Record<string, ExposureProvenance>
}

// Body accettato da PUT /assets/{id}/exposure. Le dimensioni sono
// indipendenti: omettendo una chiave la relativa distribuzione non viene
// modificata. Ogni dimensione inviata può portare la propria fonte di
// provenienza (`*_source`); inviarla senza fonte fa usare 'manual' al
// backend.
export interface AssetExposurePatch {
  countries?: ExposureRow[]
  regions?: ExposureRow[]
  sectors?: ExposureRow[]
  countries_source?: string
  regions_source?: string
  sectors_source?: string
}

export interface Currency {
  code: string
  name: string
  symbol?: string
  enabled?: boolean
  sort?: number
  created_at?: string
}

export interface CurrencyList {
  currencies: Currency[]
}

export interface Transaction {
  id: string
  portfolio_id: string
  asset_id: string
  asset_ticker: string
  asset_name: string
  type: 'buy' | 'sell' | 'dividend' | 'split' | 'fee'
  quantity: string
  price: string
  fees: string
  date: string
  notes: string
}

/** Envelope of the paginated `GET /portfolios/{id}/transactions` (EPIC I.9,
 * #88): one page of transactions (newest first), the total count of the
 * (optionally filtered, K.4c) set and the `limit`/`offset` the backend
 * actually applied (default limit 20, clamped to a max of 100). */
export interface TransactionPage {
  transactions: Transaction[]
  total: number
  limit: number
  offset: number
}

export interface AssetHolding {
  asset_id: string
  ticker: string
  name: string
  currency: string
  qty: string
  cost: string
  cost_ccy: string
  value: string
  value_pf: string
  /** Latest closing price in the asset's own currency (decimal string). */
  last_close: string
  realized: string
  realized_ccy: string
  unrealized: string
  roi: string
  fx_missing: boolean
  closed: boolean
}

export interface PortfolioSummary {
  portfolio_id: string
  portfolio_name: string
  /** Active/closed roll-ups in the PORTFOLIO currency (EPIC I.6, #85): the
   * same nested shape the dashboard summary exposes, so both pages can share
   * the `InvestmentsTable` component. */
  active: ActiveBreakdown
  closed: ClosedBreakdown
  total_value: string
  total_cost: string
  gain_loss: string
  gain_loss_pct: string
  realized_gl: string
  unrealized_gl: string
  asset_count: number
  holdings: AssetHolding[]
}

export interface AssetAllocation {
  asset_id: string
  ticker: string
  name: string
  value: string
  alloc_pct: string
}

export interface AssetClassSlice {
  class: string
  value: string
  weight: string
}

export interface PortfolioClassAllocation {
  currency: string
  classes: AssetClassSlice[]
}

export interface RegionAllocation {
  region: string
  value: string
  weight: string
}

export interface SectorAllocation {
  sector: string
  value: string
  weight: string
}

/** One country bucket of the vault-wide country exposure (EPIC I.4).
 * `country` is an ISO alpha-2 code. */
export interface CountryAllocation {
  country: string
  value: string
  weight: string
}

export interface PortfolioGeographyAllocation {
  currency: string
  regions: RegionAllocation[]
  /** Equity-only per-country exposure of this portfolio (EPIC I.7, #86),
   * with the same semantics as the dashboard's `countries`: ISO alpha-2
   * codes, non-zero buckets only, sorted by descending value. Values in the
   * portfolio currency. */
  countries: CountryAllocation[]
  covered_value?: string
  excluded_value?: string
}

export interface PortfolioSectorAllocation {
  currency: string
  sectors: SectorAllocation[]
  covered_value?: string
  excluded_value?: string
}

/** Vault-wide allocation across all portfolios, converted to the user's base
 * currency (EPIC I.4): asset classes over every holding, plus the equity-only
 * breakdown by macro-region, country (ISO alpha-2, non-zero rows only) and
 * GICS sector. `classes`/`regions`/`countries`/`sectors` come back sorted by
 * descending value; `covered_value`/`excluded_value` split the total between
 * the equity universe and the non-equity holdings excluded from it. */
export interface DashboardAllocation {
  currency: string
  classes: AssetClassSlice[]
  regions: RegionAllocation[]
  countries: CountryAllocation[]
  sectors: SectorAllocation[]
  covered_value?: string
  excluded_value?: string
}

/** Dimension of an allocation bucket the backend can decompose into its
 * contributing assets (EPIC K.5 drill-down). */
export type AllocationDrillDim = 'class' | 'country' | 'region' | 'sector'

/** One asset's participation in a drill-down bucket (EPIC K.5): `weight`
 * is the asset's exposure weight *within that bucket* (%), `contribution`
 * = value × weight/100 is the amount it places into the bucket. Rows come
 * back sorted by descending contribution, `value` in the drill currency. */
export interface AllocationDrillAsset {
  asset_id: string
  ticker: string
  name: string
  value: string
  weight: string
  contribution: string
}

/** Response of `GET .../allocation/drill` (EPIC K.5): the contributing
 * assets of one allocation bucket, in the currency of the allocation
 * endpoint that produced it; `total` is the bucket total (Σ contributions)
 * and `key` echoes the (possibly space-carrying) bucket identifier. */
export interface AllocationDrill {
  currency: string
  dim: string
  key: string
  total: string
  assets: AllocationDrillAsset[]
}

export interface PortfolioPerformance {
  date: string
  value: string
}

export interface AssetROI {
  asset_id: string
  ticker: string
  name: string
  roi: string
  total_invested: string
  current_value: string
}

export interface CurrencyPerformance {
  currency: string
  invested: string
  value: string
  gain_loss: string
  gain_loss_pct: string
  realized: string
}

/** Roll-up of the open (still held) lot portions (EPIC I.2); `dividends`
 * are those received on still-open positions. */
export interface ActiveBreakdown {
  invested: string
  value: string
  gain_loss: string
  gain_loss_pct: string
  dividends: string
}

/** Roll-up of the closed (already sold) lot portions (EPIC I.2): `invested`
 * is the cost of the sold lots, `proceeds` the net sale proceeds plus the
 * dividends of fully-closed positions, `realized` = proceeds − invested
 * (so dividends are already folded into the capital figures). */
export interface ClosedBreakdown {
  invested: string
  proceeds: string
  realized: string
  realized_pct: string
}

export interface PortfolioPerformanceSummary {
  portfolio_id: string
  portfolio_name: string
  currency: string
  active: ActiveBreakdown
  closed: ClosedBreakdown
  asset_count: number
  fx_missing: number
}

export interface AssetPerformance {
  asset_id: string
  ticker: string
  name: string
  currency: string
  qty: string
  invested: string
  value: string
  gain_loss: string
  roi: string
  fx_missing: boolean
  value_pf: string
  realized: string
  realized_pf: string
}

export interface PortfolioAssets {
  portfolio_id: string
  portfolio_name: string
  currency: string
  assets: AssetPerformance[]
}

/** One month or year bucket of the performance charts (EPIC I.3):
 * `period` is "YYYY-MM" (monthly) or "YYYY" (annual), `return` the
 * time-weighted return % generated inside the bucket (bars), `twr` the cumulative
 * time-weighted return % up to the bucket's last date (line), `invested` the
 * net invested capital and `value` the market value at the bucket's end, both
 * in the payload's currency. Buckets come back ascending; empty ones are
 * omitted. */
export interface PerformanceBucket {
  period: string
  return: string
  twr: string
  invested: string
  value: string
}

/** Return/capital chart bucketed by month or year: vault-wide in the user's
 * base currency (`/dashboard/performance`, EPIC I.3) or — since EPIC I.8
 * (#87) — of a single portfolio in the portfolio's own currency
 * (`/portfolios/{id}/performance/buckets`). Same shape for both endpoints. */
export interface DashboardPerformance {
  currency: string
  granularity: 'month' | 'year'
  buckets: PerformanceBucket[]
}

export interface PositionPoint {
  date: string
  qty: string
  cost_basis: string
  market_value: string
  realized: string
}

export interface SplitInfo {
  date: string // ISO
  ratio: string // es. "7:1", "4:1"
}

export interface AssetPositionSeries {
  asset_id: string
  ticker: string
  name: string
  currency: string
  series: PositionPoint[]
  splits: SplitInfo[]
}

export interface PortfolioHistory {
  portfolio_id: string
  portfolio_name: string
  currency: string
  series: PositionPoint[]
  splits: SplitInfo[]
  assets: AssetPositionSeries[]
}

export interface PortfolioExportDocument {
  version: number
  exported_at: string
  portfolio: {
    name: string
    description: string
    currency: string
  }
  assets: {
    ticker: string
    name: string
    type: string
    currency: string
    isin?: string
  }[]
  transactions: {
    date: string
    type: string
    asset_ticker: string
    quantity: string
    price: string
    fees?: string
    notes?: string
  }[]
}

/** Aggregated dashboard totals converted into the user's base currency
 * (EPIC I.1), split into the nested `active`/`closed` breakdowns of
 * EPIC I.2. Decimal fields are JSON strings, like the rest of the API. */
export interface DashboardSummary {
  currency: string
  active: ActiveBreakdown
  closed: ClosedBreakdown
  /** Number of holdings whose FX rate was missing in the conversion. */
  fx_missing_count: number
  /** Value of those holdings (decimal string), i.e. what the count refers to. */
  fx_missing_value: string
}

/** One open asset aggregated across all the user's portfolios, expressed in
 * the base currency (EPIC I.5): `invested` is the cost of the open quantity,
 * `value` the market value — carried at cost when `has_price` is false, so
 * such rows show a zero P/L. Sorted by descending value. */
export interface InvestedAsset {
  asset_id: string
  ticker: string
  name: string
  currency: string
  invested: string
  value: string
  gain_loss: string
  gain_loss_pct: string
  has_price: boolean
}

export interface Dashboard {
  by_currency: CurrencyPerformance[]
  portfolios: PortfolioPerformanceSummary[]
  assets: PortfolioAssets[]
  /** Consolidated invested-assets table (EPIC I.5): open positions merged
   * across portfolios in `base_currency`, sorted by value descending. */
  invested_assets: InvestedAsset[]
  /** User's base currency: the consolidated `summary` (and the separate
   * `/dashboard/performance` endpoint) are expressed in it. The old `history`
   * series was removed in EPIC I.3, superseded by `dashboardPerformance`. */
  base_currency: string
  /** Consolidated totals in `base_currency`; absent on older backends. */
  summary?: DashboardSummary
}

/** Body of a job-enqueue endpoint (HTTP 202): poll `job_id` via `jobsApi`. */
export interface JobEnqueued {
  job_id: string
  status: Job['status']
}

/** Queue job tracked through `GET /jobs/{id}`: long external-site work
 * (price refresh, history/meta backfills) runs out of band and reports
 * progress here. `status` is `queued`/`running` while open, `done`/`failed`/
 * `partial` once terminal. */
export interface Job {
  id: string
  type: string
  /** `asset` | `portfolio` | `global` (plus the optional uuid `target_id`). */
  target_type?: string
  target_id?: string
  status: 'queued' | 'running' | 'done' | 'failed' | 'partial'
  total: number
  processed: number
  error?: string
  requested_by?: string
  created_at: string
  started_at?: string
  finished_at?: string
  /** Wall-clock run time of a terminal job (derived server-side). */
  duration_ms?: number
  /** Item-level health rollup of a terminal job (derived server-side). */
  summary?: { ok: number; failed: number }
}

export interface AuthResponse {
  user: User
  access_token: string
  refresh_token: string
}

export interface AssetLookupResult {
  ticker: string
  name: string
  type: string
  currency: string
  exchange: string
}

export interface AssetMeta {
  ticker: string
  name: string
  type: string
  currency: string
  exchange: string
  country: string
  asset_class: string
}

interface RequestOptions {
  method?: string
  body?: unknown
  params?: Record<string, string>
  /** Multipart body: sent as-is (no JSON encoding, no Content-Type so fetch
   * derives the boundary). Overrides `body`. */
  form?: FormData
  /** Return the raw `Response` instead of parsing JSON (for file downloads). */
  raw?: boolean
  /** Skip the GET cache on reads of mutable resources (job status polling
   * would otherwise replay the same cached answer for the whole TTL). */
  noCache?: boolean
}

function buildUrl(path: string, params?: Record<string, string>): string {
  const url = new URL(BASE_URL + path, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([key, value]) => url.searchParams.set(key, value))
  }
  return url.toString()
}

/**
 * Shared 401 → refresh → retry step: on success stores the rotated token
 * pair, updates the `Authorization` header in place and returns `true` (the
 * caller re-runs its request). A refused or failed refresh clears the
 * session and hard-redirects to `/login` — exactly what `request` used to do
 * inline — and returns `false`, leaving the original response to be handled
 * by the caller.
 */
async function tryRefreshToken(headers: Record<string, string>): Promise<boolean> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) return false
  try {
    const refreshRes = await fetch(`${BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (refreshRes.ok) {
      const data = await refreshRes.json()
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      headers.Authorization = `Bearer ${data.access_token}`
      return true
    }
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    window.location.replace('/login')
  } catch {
    window.location.replace('/login')
  }
  return false
}

/** The HTTP status carried by an error built by `errorFrom`, if any (network
 * errors and other plain `Error`s have none). Shared by every page that maps
 * specific statuses to localized messages. */
export function errorStatus(err: unknown): number | undefined {
  return err instanceof Error && 'status' in err
    ? (err as Error & { status: number }).status
    : undefined
}

/** Trigger a browser download for a blob, falling back to `fallbackName`
 * when the server did not suggest a filename. Shared by the backup/export
 * flows. */
export function saveBlob(blob: Blob, fallbackName: string, filename: string | null): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename || fallbackName
  a.click()
  URL.revokeObjectURL(url)
}

/** Build the `Error` for a failed response: the backend `{"error": …}` body
 * when present (never the raw body of a non-JSON failure), plus the HTTP
 * status on the error object so pages can map specific codes. */
async function errorFrom(res: Response): Promise<Error> {
  let message = 'Something went wrong'
  try {
    const data = await res.json()
    if (data?.error) message = data.error
  } catch {
    // keep default message
  }
  return Object.assign(new Error(message), { status: res.status })
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const method = options.method ?? 'GET'
  const url = buildUrl(path, options.params)
  const cacheKey = `${method} ${url}`

  // Any non-GET invalidates the whole cache. This is intentionally coarse:
  // dashboard/statistics endpoints aggregate across all portfolios/assets, so
  // a single mutation (transaction, portfolio edit, or /prices/refresh) can
  // change any cached GET. Clearing everything is simpler and safer than
  // tracking fine-grained dependencies, at the cost of a refetch after each
  // mutation.
  if (method !== 'GET') {
    getCache.clear()
  } else if (options.noCache) {
    // Read-through: never served from, nor written into, the cache.
  } else {
    const cached = getCache.get(cacheKey)
    if (cached) {
      if (cached.expiresAt > Date.now()) {
        return cloneCached<T>(cached.data)
      }
      getCache.delete(cacheKey)
    }
  }

  const headers: Record<string, string> = {}
  if (!options.form) headers['Content-Type'] = 'application/json'
  const token = localStorage.getItem('access_token')
  if (token) headers.Authorization = `Bearer ${token}`

  const init = (): RequestInit => ({
    method,
    headers,
    body: options.form ?? (options.body !== undefined ? JSON.stringify(options.body) : undefined),
  })

  let res = await fetch(url, init())

  if (res.status === 401 && !path.includes('/auth/')) {
    if (await tryRefreshToken(headers)) {
      res = await fetch(url, init())
    }
  }

  if (!res.ok) {
    // Do not cache failures (4xx/5xx): the normal throw/retry flow applies.
    throw await errorFrom(res)
  }

  if (options.raw) return res as T

  if (res.status === 204) return undefined as T

  const data: T = await res.json()
  if (method === 'GET' && !options.noCache) {
    getCache.set(cacheKey, { data, expiresAt: Date.now() + CACHE_TTL_MS })
  }
  return cloneCached<T>(data)
}

export const api = {
  get: (url: string, init: RequestInit = {}) => request(url, { ...init, method: 'GET' }),
  post: (url: string, body: unknown, init: RequestInit = {}) => request(url, { ...init, method: 'POST', body }),
  put: (url: string, body: unknown, init: RequestInit = {}) => request(url, { ...init, method: 'PUT', body }),
  patch: (url: string, body: unknown, init: RequestInit = {}) => request(url, { ...init, method: 'PATCH', body }),
  delete: (url: string, init: RequestInit = {}) => request(url, { ...init, method: 'DELETE' }),
};

export const authApi = {
  login: (email: string, password: string) =>
    request<AuthResponse>('/auth/login', { method: 'POST', body: { email, password } }),
  register: (email: string, name: string, password: string) =>
    request<User>('/auth/register', { method: 'POST', body: { email, name, password } }),
  me: () => request<User>('/users/me'),
  // `base_currency` is sent only when provided (JSON.stringify drops the
  // undefined key): the backend keeps the existing value when omitted.
  updateProfile: (data: { name: string; email: string; base_currency?: string }) =>
    request<User>('/users/me', { method: 'PATCH', body: data }),
  changePassword: (data: { current_password: string; new_password: string }) =>
    request<void>('/users/me/password', { method: 'POST', body: data }),
}

export const portfolioApi = {
  list: () => request<Portfolio[]>('/portfolios'),
  create: (data: Partial<Portfolio>) =>
    request<Portfolio>('/portfolios', { method: 'POST', body: data }),
  get: (id: string) => request<Portfolio>(`/portfolios/${id}`),
  update: (id: string, data: Partial<Portfolio>) =>
    request<Portfolio>(`/portfolios/${id}`, { method: 'PATCH', body: data }),
  delete: (id: string) => request<void>(`/portfolios/${id}`, { method: 'DELETE' }),
  summary: (id: string) => request<PortfolioSummary>(`/portfolios/${id}/summary`),
  allocation: (id: string) => request<AssetAllocation[]>(`/portfolios/${id}/allocation`),
  classAllocation: (id: string) =>
    request<PortfolioClassAllocation>(`/portfolios/${id}/allocation/class`),
  geographyAllocation: (id: string) =>
    request<PortfolioGeographyAllocation>(`/portfolios/${id}/allocation/geography`),
  sectorAllocation: (id: string) =>
    request<PortfolioSectorAllocation>(`/portfolios/${id}/allocation/sector`),
  dashboardAllocation: () => request<DashboardAllocation>('/dashboard/allocation'),
  // EPIC K.5 drill-down: the contributing assets behind one allocation bucket.
  // `key` is the raw bucket identifier exactly as the chart carries it (ISO
  // country code, region/sector name, class key — it may contain spaces);
  // `request`'s `params` takes care of the URL encoding.
  allocationDrill: (id: string, dim: AllocationDrillDim, key: string) =>
    request<AllocationDrill>(`/portfolios/${id}/allocation/drill`, { params: { dim, key } }),
  dashboardAllocationDrill: (dim: AllocationDrillDim, key: string) =>
    request<AllocationDrill>('/dashboard/allocation/drill', { params: { dim, key } }),
  // EPIC I.3: vault-wide P/L buckets in the user's base currency, monthly or
  // yearly (`period` = "YYYY-MM" / "YYYY", ascending, empty buckets omitted).
  dashboardPerformance: (granularity: 'month' | 'year') =>
    request<DashboardPerformance>('/dashboard/performance', { params: { granularity } }),
  // EPIC I.8 (#87): the same monthly/yearly TWR buckets for a SINGLE portfolio,
  // in the portfolio's own currency (same `DashboardPerformance` shape).
  performanceBuckets: (id: string, granularity: 'month' | 'year') =>
    request<DashboardPerformance>(`/portfolios/${id}/performance/buckets`, { params: { granularity } }),
  performance: (id: string) => request<PortfolioPerformance[]>(`/portfolios/${id}/performance`),
  roi: (id: string) => request<AssetROI[]>(`/portfolios/${id}/roi`),
  history: (id: string) => request<PortfolioHistory>(`/portfolios/${id}/history`),
  dashboard: () => request<Dashboard>('/dashboard'),
  exportDoc: (id: string) => request<PortfolioExportDocument>(`/portfolios/${id}/export`),
  importDoc: (data: {
    document: PortfolioExportDocument
    mode: 'new' | 'overwrite'
    name?: string
    target_portfolio_id?: string
  }) => request<Portfolio>('/portfolios/import', { method: 'POST', body: data }),
}

export const assetApi = {
  list: () => request<Asset[]>('/assets'),
  search: (q: string) => request<Asset[]>(`/assets/search?q=${q}`),
  lookup: (q: string) => request<AssetLookupResult[]>(`/assets/lookup?q=${q}`),
  meta: (ticker: string) => request<AssetMeta>(`/assets/meta?ticker=${ticker}`),
  get: (id: string) => request<Asset>(`/assets/${id}`),
  create: (data: Partial<Asset>) => request<Asset>('/assets', { method: 'POST', body: data }),
  update: (id: string, patch: AssetPatch) =>
    request<Asset>(`/assets/${id}`, { method: 'PATCH', body: patch }),
  quote: (id: string) => request<AssetQuote>(`/assets/${id}/quote`),
  splits: (id: string) => request<SplitInfo[]>(`/assets/${id}/splits`),
  fetchProfile: (id: string) =>
    request<Asset>(`/assets/${id}/fetch-profile`, { method: 'POST' }),
  exposure: (id: string) => request<AssetExposure>(`/assets/${id}/exposure`),
  saveExposure: (id: string, exposure: AssetExposurePatch) =>
    request<AssetExposure>(`/assets/${id}/exposure`, { method: 'PUT', body: exposure }),
  fetchExposure: (id: string) =>
    request<AssetExposure>(`/assets/${id}/fetch-exposure`, { method: 'POST' }),
  fetchETFExposure: (id: string) =>
    request<AssetExposure>(`/assets/${id}/fetch-etf-exposure`, { method: 'POST' }),
  fetchMorningstarExposure: (id: string) =>
    request<AssetExposure>(`/assets/${id}/fetch-morningstar-exposure`, { method: 'POST' }),
  // Preview (not persisted) of how a country weight distribution maps onto the
  // canonical macro-regions. Returns the full canonical region list in order.
  deriveRegions: (id: string, countries: ExposureRow[]) =>
    request<{ regions: ExposureRow[] }>(`/assets/${id}/exposure/derive`, {
      method: 'POST',
      body: { countries },
    }),
  backfillHistory: (id: string) =>
    request<JobEnqueued>(`/assets/${id}/backfill-history`, { method: 'POST' }),
  remove: (id: string) => request<void>(`/assets/${id}`, { method: 'DELETE' }),
  // Queues a global asset sync (splits + history for every Yahoo asset) as a
  // worker job; poll the returned job id, then invalidate caches.
  sync: () => request<JobEnqueued>('/assets/sync', { method: 'POST' }),
}

export const transactionApi = {
  // EPIC I.9 (#88): paginated list returning the `TransactionPage` envelope.
  // `limit`/`offset` are only appended when provided; without them the
  // backend serves its default first page (limit 20, order date desc).
  // EPIC K.4c adds the optional list filters of
  // `GET /portfolios/{id}/transactions` (combinable, each omitted when
  // unset/empty): `type`, `asset_id` (uuid), `from`/`to` (inclusive
  // `YYYY-MM-DD` calendar-date bounds). `total` reflects the FILTERED count.
  list: (
    portfolioId: string,
    params?: {
      limit?: number
      offset?: number
      type?: Transaction['type']
      asset_id?: string
      from?: string
      to?: string
    },
  ): Promise<TransactionPage> => {
    const query: Record<string, string> = {}
    if (params?.limit !== undefined) query.limit = String(params.limit)
    if (params?.offset !== undefined) query.offset = String(params.offset)
    if (params?.type) query.type = params.type
    if (params?.asset_id) query.asset_id = params.asset_id
    if (params?.from) query.from = params.from
    if (params?.to) query.to = params.to
    return request<TransactionPage>(`/portfolios/${portfolioId}/transactions`, { params: query })
  },
  create: (portfolioId: string, data: Partial<Transaction>) =>
    request<Transaction>(`/portfolios/${portfolioId}/transactions`, { method: 'POST', body: data }),
  update: (id: string, data: Partial<Transaction>) =>
    request<Transaction>(`/transactions/${id}`, { method: 'PATCH', body: data }),
  remove: (id: string) => request<void>(`/transactions/${id}`, { method: 'DELETE' }),
}

export const pricesApi = {
  refresh: (portfolioId?: string) =>
    request<JobEnqueued>('/prices/refresh', {
      method: 'POST',
      params: portfolioId ? { portfolio_id: portfolioId } : {},
    }),
  byAsset: (assetId: string) => request<Price[]>(`/prices/${assetId}?full=1`),
}

const JOB_TERMINAL_STATUSES: Job['status'][] = ['done', 'failed', 'partial']

export const jobsApi = {
  get: (id: string) => request<Job>(`/jobs/${id}`, { noCache: true }),
  list: (params?: { limit?: number; offset?: number }) => {
    const query: Record<string, string> = {}
    if (params?.limit !== undefined) query.limit = String(params.limit)
    if (params?.offset !== undefined) query.offset = String(params.offset)
    return request<Job[]>('/jobs', { params: query, noCache: true })
  },
  /** Poll `get` until the job reaches a terminal status (done/failed/partial)
   * or the timeout lapses. On timeout it RESOLVES with the last observed
   * (still-open) job instead of throwing: every caller already maps any
   * non-terminal/non-done status to its localized failure toast, so nothing
   * would translate a thrown string, and a slow job keeps its progress data
   * readable by the caller either way. */
  wait: async (id: string, opts?: { intervalMs?: number; timeoutMs?: number }): Promise<Job> => {
    const intervalMs = opts?.intervalMs ?? 2000
    const deadline = Date.now() + (opts?.timeoutMs ?? 5 * 60_000)
    for (;;) {
      const job = await jobsApi.get(id)
      if (JOB_TERMINAL_STATUSES.includes(job.status) || Date.now() >= deadline) return job
      await new Promise((resolve) => setTimeout(resolve, intervalMs))
    }
  },
}

export const settingsApi = {
  listCurrencies: () => request<CurrencyList>('/settings/currencies'),
  addCurrency: (code: string, name?: string) =>
    request<Currency>('/settings/currencies', { method: 'POST', body: { code, name } }),
  deleteCurrency: (code: string) =>
    request<void>(`/settings/currencies/${encodeURIComponent(code)}`, { method: 'DELETE' }),
}

/** Restore strategy (`#56`): `add` imports portfolios as new (non-destructive,
 * the backend default); `replace` deletes the user's portfolios first. */
export type BackupRestoreMode = 'add' | 'replace'

/** Summary returned by `POST /backup/restore` (extra fields are tolerated by
 * the structural type). */
export interface BackupRestoreSummary {
  portfolios_created: number
  transactions_created: number
  assets_created: number
  assets_reused: number
}

/** Raw bytes of an attachment endpoint plus the server-suggested file name
 * parsed from `Content-Disposition` (`null` when the header is absent). */
export interface BackupDownload {
  blob: Blob
  filename: string | null
}

/** Summary returned by `POST /admin/db/restore` (#57 Phase D). */
export interface DBRestoreSummary {
  dump_bytes: number
}

/**
 * Authenticated file download shared by `backupApi.download` and
 * `adminApi.dbBackup`: a raw GET through `request` (auth header, 401 →
 * refresh → retry and error shape included) whose body is kept as bytes. The
 * suggested name comes from `Content-Disposition` (`attachment;
 * filename="…"`), with the same tolerant quote/charset handling the backends
 * that emit it use.
 */
async function downloadAttachment(path: string): Promise<BackupDownload> {
  const res = await request<Response>(path, { raw: true })
  const disposition = res.headers.get('Content-Disposition') ?? ''
  const match = /filename="?([^";]+)"?/.exec(disposition)
  return { blob: await res.blob(), filename: match?.[1]?.trim() || null }
}

/**
 * Per-user backup & restore (`/api/v1/backup`). `download` delegates to the
 * shared `downloadAttachment` helper; `restore` POSTs the bundle object read
 * from the chosen file: the server owns format/version validation (400 for
 * anything it does not understand) and answers with the counts summary.
 */
export const backupApi = {
  download: (): Promise<BackupDownload> => downloadAttachment('/backup'),
  restore: (bundle: unknown, mode: BackupRestoreMode) =>
    request<BackupRestoreSummary>('/backup/restore', { method: 'POST', params: { mode }, body: bundle }),
}

/** Admin-area endpoints (`/api/v1/admin/*`), gated server-side to the
 * `owner`/`admin` roles (403 "admin access required" for everyone else). */
export const adminApi = {
  listUsers: () => request<AdminUser[]>('/admin/users'),
  // PATCH body: at least one of role/status (role ∈ owner|admin|editor|viewer,
  // status ∈ active|pending|disabled). 409 when the change would leave the
  // server without an active admin.
  updateUser: (id: string, patch: { role?: string; status?: string }) =>
    request<AdminUser>(`/admin/users/${id}`, { method: 'PATCH', body: patch }),
  resetPassword: (id: string, password: string) =>
    request<void>(`/admin/users/${id}/reset-password`, { method: 'POST', body: { password } }),
  getSettings: () => request<ServerSettings>('/admin/settings'),
  updateSettings: (autoApproveRegistrations: boolean) =>
    request<ServerSettings>('/admin/settings', {
      method: 'PATCH',
      body: { auto_approve_registrations: autoApproveRegistrations },
    }),
  // Server database dump (`GET /admin/db/backup`): streamed
  // `application/octet-stream` attachment (`peculium-db-<date>.dump`), so it
  // shares the raw `downloadAttachment` path like the per-user backup.
  dbBackup: (): Promise<BackupDownload> => downloadAttachment('/admin/db/backup'),
  /**
   * Server database restore (#57 Phase D): DESTRUCTIVE — replaces the whole
   * database (users included) from a custom-format dump. Multipart POST with
   * the archive in the `dump` file field and the literal `confirm=replace`
   * form field (the handler also accepts it as a query param; the field is
   * the contract this client keeps). A 400 answers a missing
   * confirmation/file; a 500 carries the sanitized `pg_restore` error on
   * purpose — the admin must read what failed, so pages surface
   * `error.message` for that status.
   */
  dbRestore: (file: File): Promise<DBRestoreSummary> => {
    const form = new FormData()
    form.append('confirm', 'replace')
    form.append('dump', file)
    return request<DBRestoreSummary>('/admin/db/restore', { method: 'POST', form })
  },
}
