import type { ResolvedTheme } from '#lib/stores/theme.svelte.js'
import { palette as paletteState } from '#lib/stores/palette.svelte.js'

/**
 * Light-mode series palette. Mirrors the `--chart-1..12` custom properties
 * defined on `:root` in `src/app.css` — keep the two in sync. Kept as the
 * module-level default so non-theme-aware callers (and SSR/prerender passes)
 * always have the historical values.
 */
export const CHART_PALETTE = [
  '#2563eb', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6',
  '#ec4899', '#14b8a6', '#f97316', '#6366f1', '#84cc16',
  '#06b6d4', '#a855f7',
]

/** Dark-mode series palette. Mirrors the `.dark` overrides in `src/app.css`. */
export const CHART_PALETTE_DARK = [
  '#60a5fa', '#34d399', '#fbbf24', '#f87171', '#a78bfa',
  '#f472b6', '#2dd4bf', '#fb923c', '#818cf8', '#a3e635',
  '#22d3ee', '#c084fc',
]

/** Number of series color tokens (`--chart-1` … `--chart-N`). */
export const CHART_TOKEN_COUNT = 12

/** Grey for de-emphasized slices ("Other") and unknown rows; mirrors `--chart-muted`. */
export const CHART_MUTED: Record<ResolvedTheme, string> = {
  light: '#9ca3af',
  dark: '#64748b',
}

/**
 * Chart-specific semantic colors (per resolved theme). Light values equal the
 * colors previously hard-coded inside the chart components; chart components
 * switch to these in the D.1b migration commit.
 */
export interface ChartSemanticColors {
  /** Flat cost-basis reference line (PositionChart). */
  costBasis: string
  /** Market-value line (PositionChart). */
  marketValue: string
  /** Realized P/L step line (PositionChart). */
  realized: string
  /** Stock-split markLine (PriceChart / PositionChart). */
  splitMarkLine: string
  /** Grey of the aggregated "Other" slice in the donut charts. */
  other: string
  /** Positive P/L bars (PerformanceChart); mirrors `--positive`. */
  positive: string
  /** Negative P/L bars (PerformanceChart); mirrors `--negative`. */
  negative: string
}

export const CHART_SEMANTIC_COLORS: Record<ResolvedTheme, ChartSemanticColors> = {
  light: {
    costBasis: '#64748b',
    marketValue: '#16a34a',
    realized: '#f59e0b',
    splitMarkLine: '#7c3aed',
    other: CHART_MUTED.light,
    positive: '#16a34a',
    negative: '#dc2626',
  },
  dark: {
    costBasis: '#94a3b8',
    marketValue: '#4ade80',
    realized: '#fbbf24',
    splitMarkLine: '#a78bfa',
    other: CHART_MUTED.dark,
    positive: '#4ade80',
    negative: '#f87171',
  },
}

/**
 * CVD variants (EPIC K.5c, decision D6): the opt-in blue/orange gain/loss
 * pair, mirroring the `html.cvd` overrides in `src/app.css` — keep the two
 * in sync. Only `positive`/`negative` differ; every other semantic color is
 * colour-vision-neutral and stays.
 */
export const CHART_SEMANTIC_COLORS_CVD: Record<ResolvedTheme, ChartSemanticColors> = {
  light: { ...CHART_SEMANTIC_COLORS.light, positive: '#0072b2', negative: '#c2410c' },
  dark: { ...CHART_SEMANTIC_COLORS.dark, positive: '#56b4e9', negative: '#fb923c' },
}

/** Read the 12 `--chart-N` tokens from the document root. Null when unavailable. */
function readChartTokensFromCss(): string[] | null {
  if (typeof document === 'undefined' || !document.documentElement) return null
  const styles = getComputedStyle(document.documentElement)
  const colors: string[] = []
  for (let i = 1; i <= CHART_TOKEN_COUNT; i++) {
    const value = styles.getPropertyValue(`--chart-${i}`).trim()
    if (!value) return null
    colors.push(value)
  }
  return colors
}

/** Which theme is currently painted, read from the source of truth (`<html>`). */
function paintedTheme(): ResolvedTheme {
  if (typeof document === 'undefined' || !document.documentElement) return 'light'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const paletteCache = new Map<ResolvedTheme, string[]>()

/**
 * The series palette for the given (default: currently painted) theme, read
 * from the `--chart-1..12` CSS custom properties and cached per resolved
 * theme so repeated chart renders don't hit `getComputedStyle`.
 * Falls back to the static mirror on the server / before app.css is applied.
 */
export function resolvePalette(mode: ResolvedTheme = paintedTheme()): string[] {
  const cached = paletteCache.get(mode)
  if (cached) return cached
  const fallback = mode === 'dark' ? CHART_PALETTE_DARK : CHART_PALETTE
  const palette = mode === paintedTheme() ? (readChartTokensFromCss() ?? fallback) : fallback
  paletteCache.set(mode, palette)
  return palette
}

/** The `--chart-muted` token for the given (default: painted) theme. */
export function resolveChartMuted(mode: ResolvedTheme = paintedTheme()): string {
  if (mode !== paintedTheme()) return CHART_MUTED[mode]
  if (typeof document === 'undefined' || !document.documentElement) return CHART_MUTED[mode]
  const value = getComputedStyle(document.documentElement).getPropertyValue('--chart-muted').trim()
  return value || CHART_MUTED[mode]
}

/**
 * Semantic chart colors for the given (default: painted) theme. When the
 * CVD palette is enabled (`palette.cvd` in `stores/palette.svelte.ts`, the
 * source of truth kept in lockstep with the `<html class="cvd">` class that
 * drives the CSS tokens) the blue/orange variants are returned. The
 * `paletteState.cvd` read is what makes a caller's `$derived`/`{#key}`
 * reactive to palette flips — same tracking trick as `resolved()` for the
 * theme — and the two static tables give correct results per (theme, cvd)
 * without any cache invalidation or `getComputedStyle` pass.
 */
export function chartSemanticColors(mode: ResolvedTheme = paintedTheme()): ChartSemanticColors {
  const table = paletteState.cvd ? CHART_SEMANTIC_COLORS_CVD : CHART_SEMANTIC_COLORS
  return table[mode]
}

export interface ChartRow {
  name: string
  weight: string | number
}

// Restituisce il colore che ECharts assegna a una riga in un pie chart:
// i colori sono dati per indice sulle sole righe con weight > 0 (ordine della
// palette), quindi il quadratino in tabella combacia sempre con la fetta.
// `palette` è il tema resolved corrente (resolvePalette() dei chiamanti
// theme-aware, D.1b); le righe non visibili in grafico (peso 0) usano il
// token muted del tema attualmente dipinto.
export function colorForRow(row: ChartRow, rows: ChartRow[], palette: string[]): string {
  const visible = rows.filter((r) => Number(r.weight) > 0)
  const index = visible.findIndex((r) => r.name === row.name)
  if (index < 0) {
    return CHART_MUTED[paintedTheme()]
  }
  return palette[index % palette.length]
}
