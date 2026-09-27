import { browser } from '$app/environment'

/**
 * Optional colour-vision-deficient (CVD) gain/loss palette (EPIC K.5c,
 * decision D6): an opt-in swap of the green/red P/L pair for a blue/orange
 * one (the axis that stays discriminable under protanopia, deuteranopia and
 * tritanopia). Mirrors `stores/theme.svelte.ts`: reactive `$state` +
 * localStorage persistence + a class on `<html>` (`cvd`) that flips the
 * `--positive`/`--negative` tokens in `app.css` (text) — charts read the
 * same reactive state through `lib/chartPalette.ts`. Sign + ▲▼ glyphs are
 * always rendered (K.1c), so this only ever changes hues.
 */

/**
 * Storage key and class name written on `<html>`. The pre-paint bootstrap
 * script in `src/app.html` mirrors both — keep them in sync.
 */
export const CVD_STORAGE_KEY = 'peculium-cvd'

/** Legacy storage key: when the new key is absent, the value stored here is
 *  adopted and written forward once, so no saved preference is lost. */
export const LEGACY_CVD_STORAGE_KEY = 'vaultlab-cvd'

/** Default: the classic green/red palette (finance convention, spec §7.2). */
export const DEFAULT_CVD = false

function readStoredCvd(): boolean {
  if (!browser) return DEFAULT_CVD
  try {
    let stored = window.localStorage.getItem(CVD_STORAGE_KEY)
    if (stored === null) {
      // One-time migration: adopt the legacy value and write it forward.
      const legacy = window.localStorage.getItem(LEGACY_CVD_STORAGE_KEY)
      if (legacy !== null) {
        window.localStorage.setItem(CVD_STORAGE_KEY, legacy)
        stored = legacy
      }
    }
    return stored === 'true'
  } catch {
    // Storage unavailable (e.g. private mode): fall back to the default.
    return DEFAULT_CVD
  }
}

/** Reactive palette state: `cvd` is the user's opt-in to the blue/orange P/L pair. */
export const palette = $state({
  cvd: readStoredCvd(),
})

function syncCvdClass(): void {
  if (!browser) return
  document.documentElement.classList.toggle('cvd', palette.cvd)
}

/** Set the CVD palette, persist it and keep the `<html class="cvd">` in sync. */
export function setCvd(enabled: boolean): void {
  palette.cvd = enabled
  if (browser) {
    try {
      window.localStorage.setItem(CVD_STORAGE_KEY, String(enabled))
    } catch {
      // Storage unavailable: the in-memory palette still applies for this visit.
    }
    syncCvdClass()
  }
}

if (browser) {
  // Cross-tab sync: another tab toggling the preference updates this one too
  // (text follows the `<html>` class via CSS; chart components re-init on
  // the reactive `palette.cvd` read inside `chartSemanticColors()`).
  window.addEventListener('storage', (event) => {
    if (event.key === CVD_STORAGE_KEY) {
      palette.cvd = event.newValue === 'true'
      syncCvdClass()
    }
  })

  // Reassert the class on boot in case the pre-paint script and the store
  // ever disagree (e.g. storage changed between the two).
  syncCvdClass()
}
