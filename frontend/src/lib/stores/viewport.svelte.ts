import { browser } from '$app/env';

/**
 * Reactive viewport helper (EPIC K.2): a thin `matchMedia` wrapper exposing
 * the three device classes of the spec (§5.1), split on Tailwind's own
 * `sm`/`lg` boundaries (640 / 1024px — the `.98` upper bounds mirror
 * Tailwind's max-* variants so the JS state and the CSS classes never
 * disagree by a fraction of a pixel):
 *
 * - phone   < 640         → bottom nav + Fab, no sidebar chrome;
 * - tablet  640 … 1023    → icon rail (the shell forces `collapsed` there);
 * - desktop ≥ 1024        → expandable sidebar, persisted preference.
 *
 * The app is SPA-only (`ssr = false`), so the initializer normally runs in
 * the browser; it is still guarded with `browser` + a feature check and
 * falls back to desktop chrome, in which case no listener is registered.
 * `matchMedia` change events also cover resizes and orientation/zoom
 * changes, so no `resize` listener is needed.
 */
export const viewport = $state({
  isPhone: false,
  isTablet: false,
  isDesktop: true,
})

const QUERIES = {
  isPhone: '(max-width: 639.98px)',
  isTablet: '(min-width: 640px) and (max-width: 1023.98px)',
  isDesktop: '(min-width: 1024px)',
} as const

type ViewportKey = keyof typeof QUERIES

function canMatch(): boolean {
  return browser && typeof window.matchMedia === 'function'
}

function read(): Record<ViewportKey, boolean> {
  return {
    isPhone: window.matchMedia(QUERIES.isPhone).matches,
    isTablet: window.matchMedia(QUERIES.isTablet).matches,
    isDesktop: window.matchMedia(QUERIES.isDesktop).matches,
  }
}

function sync(): void {
  if (!canMatch()) return
  const next = read()
  viewport.isPhone = next.isPhone
  viewport.isTablet = next.isTablet
  viewport.isDesktop = next.isDesktop
}

// Evaluate once at import time so the first shell render is already correct
// (the module is never executed on the server: `ssr = false`).
sync()

if (canMatch()) {
  for (const query of Object.values(QUERIES)) {
    window.matchMedia(query).addEventListener('change', sync)
  }
}
