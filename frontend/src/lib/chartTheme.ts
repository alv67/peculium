/**
 * ECharts theme registration for the Peculium design system (EPIC D / D.1a).
 *
 * The canvas renderer cannot resolve CSS custom properties, so the token
 * triples below are a canvas-safe mirror of `:root` / `.dark` in `src/app.css`
 * — keep the two files in sync. The series palettes come from the static
 * mirrors in `lib/chartPalette.ts` (same single source: app.css).
 *
 * No chart component consumes the registered themes yet: wiring happens in
 * the D.1b migration commit. Because svelte-echarts types its `theme` prop as
 * `'light' | 'dark' | object`, components can either pass the matching object
 * from `PECULIUM_CHART_THEMES[resolved()]` or the registered name through a
 * plain `init` (ECharts resolves registered names at runtime).
 */
import { registerTheme } from 'echarts/core'
import type { ResolvedTheme } from '$lib/stores/theme.svelte'
import { CHART_PALETTE, CHART_PALETTE_DARK } from '$lib/chartPalette'

/** Keep in sync with `fontFamily.sans` in tailwind.config.js (the canvas
 *  cannot resolve the CSS font stack). */
export const CHART_FONT_FAMILY =
  'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif'

/** Space-separated HSL triples mirroring the semantic tokens of app.css
 *  (raw hex for `grid`, which mirrors the literal `--chart-grid` token). */
interface ChartTokens {
  foreground: string
  muted: string
  mutedForeground: string
  border: string
  input: string
  ring: string
  surface: string
  surfaceRaised: string
  grid: string
}

/** Gridline ink (`--chart-grid`) is painted as a whisper over the surface. */
const GRID_LINE_OPACITY = 0.08

const TOKENS: Record<ResolvedTheme, ChartTokens> = {
  light: {
    foreground: '222.2 47.4% 11.2%',
    muted: '214.3 31.8% 91.4%',
    mutedForeground: '215.3 19.3% 34.5%',
    border: '214.3 31.8% 91.4%',
    input: '212.7 26.8% 83.9%',
    ring: '221.2 83.2% 53.3%',
    surface: '0 0% 100%',
    surfaceRaised: '0 0% 100%',
    grid: '#0f172a',
  },
  dark: {
    foreground: '210 40% 98%',
    muted: '217.2 32.6% 17.5%',
    mutedForeground: '215.4 16.3% 56.9%',
    border: '217.2 32.6% 17.5%',
    input: '215.3 25% 26.7%',
    ring: '213.1 93.9% 67.8%',
    surface: '222.2 47.4% 11.2%',
    surfaceRaised: '217.2 32.6% 17.5%',
    grid: '#cbd5e1',
  },
}

/** Turn an `h s l` triple into a canvas-parsable hsl()/hsla() string. */
function hsl(triple: string, alpha = 1): string {
  const [h, s, l] = triple.split(' ')
  return alpha < 1 ? `hsla(${h}, ${s}, ${l}, ${alpha})` : `hsl(${h}, ${s}, ${l})`
}

function axisTheme(tokens: ChartTokens, splitLineShown: boolean) {
  return {
    axisLine: { show: true, lineStyle: { color: hsl(tokens.border) } },
    axisTick: { show: false, lineStyle: { color: hsl(tokens.border) } },
    axisLabel: { show: true, color: hsl(tokens.mutedForeground) },
    splitLine: {
      show: splitLineShown,
      lineStyle: { color: tokens.grid, opacity: GRID_LINE_OPACITY },
    },
    splitArea: { show: false },
  }
}

function buildTheme(mode: ResolvedTheme) {
  const t = TOKENS[mode]
  const palette = mode === 'dark' ? CHART_PALETTE_DARK : CHART_PALETTE
  return {
    color: [...palette],
    backgroundColor: 'transparent',
    textStyle: { fontFamily: CHART_FONT_FAMILY, color: hsl(t.foreground) },
    title: {
      textStyle: { color: hsl(t.foreground) },
      subtextStyle: { color: hsl(t.mutedForeground) },
    },
    legend: { textStyle: { color: hsl(t.mutedForeground) } },
    tooltip: {
      backgroundColor: hsl(t.surfaceRaised),
      borderColor: hsl(t.border),
      borderWidth: 1,
      textStyle: { color: hsl(t.foreground) },
      axisPointer: {
        lineStyle: { color: hsl(t.input) },
        crossStyle: { color: hsl(t.input) },
      },
    },
    grid: { borderColor: hsl(t.border) },
    // The app uses category, value and time axes (line/bar/candlestick charts
    // plus the price-history dataZoom slider).
    categoryAxis: axisTheme(t, false),
    valueAxis: axisTheme(t, true),
    timeAxis: axisTheme(t, true),
    logAxis: axisTheme(t, true),
    dataZoom: {
      borderColor: 'transparent',
      backgroundColor: hsl(t.muted, 0.2),
      fillerColor: hsl(t.ring, 0.15),
      handleStyle: { color: hsl(t.surface), borderColor: hsl(t.input) },
      moveHandleStyle: { color: hsl(t.input) },
      textStyle: { color: hsl(t.mutedForeground) },
      dataBackground: {
        lineStyle: { color: hsl(t.input) },
        areaStyle: { color: hsl(t.muted, 0.6) },
      },
      selectedDataBackground: {
        lineStyle: { color: hsl(t.ring) },
        areaStyle: { color: hsl(t.ring, 0.2) },
      },
    },
  }
}

/** Registered theme names, keyed by resolved mode. */
export const CHART_THEME_NAMES: Record<ResolvedTheme, string> = {
  light: 'peculium-light',
  dark: 'peculium-dark',
}

/** Theme option objects (handy for wrappers that only accept an object). */
export const PECULIUM_CHART_THEMES: Record<ResolvedTheme, ReturnType<typeof buildTheme>> = {
  light: buildTheme('light'),
  dark: buildTheme('dark'),
}

// Side effect on import: make the themes resolvable by `init(el, themeName)`.
// registerTheme is idempotent per name, so multiple imports are safe.
registerTheme(CHART_THEME_NAMES.light, PECULIUM_CHART_THEMES.light)
registerTheme(CHART_THEME_NAMES.dark, PECULIUM_CHART_THEMES.dark)

/** Pick the registered ECharts theme name for the current resolved mode. */
export function chartThemeName(mode: ResolvedTheme): string {
  return CHART_THEME_NAMES[mode]
}
