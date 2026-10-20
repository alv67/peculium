import { createContext } from 'svelte'
import type {
  DashboardPerformance,
  Portfolio,
  PortfolioClassAllocation,
  PortfolioGeographyAllocation,
  PortfolioHistory,
  PortfolioSectorAllocation,
  PortfolioSummary,
  Transaction,
} from '#lib/services/api.js'
import type { TransactionFilters } from './tx-filters'

/**
 * Typed context shared by the portfolio-detail shell (`+layout.svelte`,
 * EPIC K.4a, spec §4.2/§6.2) and its four tab pages
 * (Overview / Positions / Activity / Allocation).
 *
 * The layout OWNS all data loading: the portfolio payload, the summary,
 * the transactions page (EPIC I.9 pagination), the monthly/annual TWR
 * buckets (EPIC I.8), the value history, the three allocation endpoints
 * (EPIC I.7), the summary/performance refetch that follows the shell's
 * session price refresh, and every post-mutation refetch (E.9). The tabs
 * read the same reactive state through the getters below — nothing is
 * ever fetched twice — and hand user intents (pagination, add/edit
 * transaction) back through the methods. View-level derivations
 * (position rows, exposure-bar rows, coverage notes) are re-computed
 * inside the tab that renders them.
 *
 * The state members are getters on purpose: they proxy the layout's
 * `$state`, and only reads performed *in the tab* register as that tab's
 * dependencies. Destructuring the context would snapshot the values and
 * silently break reactivity.
 */
export interface PortfolioPageContext {
  /** Route param (`page.params.id`, defensively optional like before). */
  readonly id: string | undefined
  readonly portfolio: Portfolio | null
  readonly summary: PortfolioSummary | null
  /** Portfolio currency, `'USD'` fallback while the payload loads. */
  readonly currency: string

  // --- Overview: performance card (EPIC I.8) + value history (I.8 secondary)
  readonly perf: DashboardPerformance | null
  readonly perfLoading: boolean
  readonly granularity: 'month' | 'year'
  /** String accessors for the `SegmentedControl` binding (union kept here). */
  getGranularity(): string
  setGranularity(value: string): void
  readonly history: PortfolioHistory | null

  // --- Activity: paginated transactions page (EPIC I.9)
  readonly transactions: Transaction[] | null
  readonly txPage: number
  readonly txLimit: number
  readonly txOffset: number
  readonly txTotal: number
  readonly txLoading: boolean
  readonly txRangeLabel: string
  /** Jump to a 1-based page (clamped) — refetches only the window. */
  gotoTxPage(target: number): void

  // --- Activity filters (EPIC K.4c, spec §6.2/§8.2): the URL query is
  // their single source of truth; `txFilters` is parsed from it (unset
  // dimensions are `undefined`), `setTxFilters` persists a new set back
  // into the URL (replaceState) — the layout's own watcher then resets the
  // window to the first page of the filtered result and refetches it, so
  // this also stays consistent across back/forward navigations.
  readonly txFilters: TransactionFilters
  setTxFilters(next: TransactionFilters): void

  // --- Transaction modal (mounted by the layout, opened from any tab)
  openAddTransaction(): void
  editTransaction(tx: Transaction): void

  // --- Allocation tab + Overview digest (EPIC I.7)
  readonly classAlloc: PortfolioClassAllocation | null
  readonly geoAlloc: PortfolioGeographyAllocation | null
  readonly sectorAlloc: PortfolioSectorAllocation | null
  readonly classAllocError: boolean
  readonly geoAllocError: boolean
  readonly sectorAllocError: boolean
}

/** `[get, set]` pair — `get` throws if no portfolio-detail layout provides it. */
export const [getPortfolioPage, setPortfolioPage] = createContext<PortfolioPageContext>()
