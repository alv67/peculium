import type { Dashboard } from '$lib/services/api'

/**
 * Vault-level data-quality counters feeding the global quality strip in
 * `AppShell` (the FX-missing chip of `DataQualityStrip`). Module-scoped
 * `$state` like `priceRefresh`: the shell populates it on mount via
 * `portfolioApi.dashboard()` so the strip is correct on *any* landing page
 * (deep links never mount the dashboard), and the dashboard payload keeps it
 * fresh on every visit/refetch through `applyDashboardStatus`.
 */
export const dashboardStatus = $state({
  /** The counters below reflect at least one dashboard payload. */
  loaded: false,
  /** Base currency the excluded FX value is expressed in. */
  currency: 'EUR',
  fxMissingCount: 0,
  fxMissingValue: '0',
})

/** Mirror the vault-wide data-quality fields of a dashboard payload. */
export function applyDashboardStatus(dash: Dashboard): void {
  dashboardStatus.currency = dash.base_currency
  dashboardStatus.fxMissingCount = dash.summary?.fx_missing_count ?? 0
  dashboardStatus.fxMissingValue = dash.summary?.fx_missing_value ?? '0'
  dashboardStatus.loaded = true
}
