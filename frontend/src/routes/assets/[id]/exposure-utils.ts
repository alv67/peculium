import type { ExposureRow } from '$lib/services/api'

/**
 * Pure exposure-list helpers shared between the asset shell
 * (`+layout.svelte`: load hydration, modal reopen, prefill/derive/save
 * assignments) and the Exposure tab's display derivations
 * (`exposure/+page.svelte`). Extracted verbatim from the pre-K.4b single
 * page — behaviour is identical, only the file moved so both consumers can
 * import the same normalisation rules.
 */

/** The hidden fallback row injected server-side at persist time for the
 * regions AND sectors dimensions; the UI never displays or edits it. */
export const OTHER_REGION = 'Other / Not Classified'

/**
 * Round a persisted weight to 2 decimals (the precision the inputs allow and
 * the UI shows): providers return greedy floats (21.26815...) whose raw sum
 * can trip the > 100 guard by fractions of a percent even when the display
 * reads 100.00%.
 */
export function roundWeight(w: string | number): string {
  const n = Number(w)
  return Number.isFinite(n) ? String(Math.round(n * 100) / 100) : '0'
}

/**
 * Normalise a provider import whose 2-decimal weights slightly exceed 100:
 * some providers (e.g. JustETF on LYSX.DE) publish pre-rounded weights that
 * sum to 100.01; the backend accepts up to 100.5 (`weightSumMax100`) but the
 * UI save guard blocks anything > 100, so the import would be unsavable.
 * Totals in (100, 100.5] are shaved down to exactly 100 by subtracting the
 * excess from the single heaviest row (first row wins ties, weight kept as a
 * 2-decimal string via `roundWeight`). This is an IMPORT-time fix only: it
 * runs at the bottom of `positiveCountries`/`withoutOther`, which every
 * provider assignment and canonical reload passes through; manual edits
 * bypass these helpers and stay blocked by the guard when they exceed 100.
 * Totals ≤ 100 are returned unchanged (no-op for normal load/save/display),
 * and totals > 100.5 are a genuine provider anomaly the guard must keep
 * surfacing, so they are also left untouched. Negative weights are never
 * introduced: if the excess exceeds the heaviest row (extreme case) the
 * rows are returned unchanged.
 */
export function capAtHundred(rows: ExposureRow[]): ExposureRow[] {
  const total =
    Math.round(rows.reduce((acc, r) => acc + (Number(r.weight) || 0), 0) * 100) / 100
  if (total <= 100 || total > 100.5) return rows
  const excess = Math.round((total - 100) * 100) / 100
  let maxIdx = 0
  for (let i = 1; i < rows.length; i++) {
    if ((Number(rows[i].weight) || 0) > (Number(rows[maxIdx].weight) || 0)) maxIdx = i
  }
  const maxWeight = Number(rows[maxIdx]?.weight) || 0
  if (excess > maxWeight) return rows
  return rows.map((r, i) =>
    i === maxIdx ? { ...r, weight: roundWeight(maxWeight - excess) } : r,
  )
}

/**
 * Copy backend/provider country rows keeping only positive weights, rounded
 * to 2 decimals and normalised to ≤ 100 via `capAtHundred` (slightly-over
 * provider totals are shaved at import, manual over-100 edits are not). The
 * exposure endpoints answer with the full canonical zero-filled list, while
 * the edit list must stay minimal (the backend drops non-positive rows on
 * save, so sending only the > 0 rows is lossless).
 */
export function positiveCountries(rows: ExposureRow[]): ExposureRow[] {
  return capAtHundred(
    rows
      .filter((r) => Number(r.weight) > 0)
      .map((r) => ({ ...r, weight: roundWeight(r.weight) })),
  )
}

/**
 * Copy backend/provider region or sector rows in canonical order, dropping
 * the «Other / Not Classified» residual, rounding weights and normalising
 * slightly-over-100 provider totals via `capAtHundred`: sums, donut and
 * table then work on visible rows only (the page re-adds the open-donut
 * gap via complete={false}).
 */
export function withoutOther(rows: ExposureRow[]): ExposureRow[] {
  return capAtHundred(
    rows
      .filter((r) => r.name !== OTHER_REGION)
      .map((r) => ({ ...r, weight: roundWeight(r.weight) })),
  )
}
