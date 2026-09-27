import { browser } from '$app/environment'

export type ThemeMode = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'peculium-theme'

/** Legacy storage key: when the new key is absent, the value stored here is
 *  adopted and written forward once, so no saved preference is lost. */
export const LEGACY_THEME_STORAGE_KEY = 'vaultlab-theme'

/**
 * Theme used when nothing valid is persisted in localStorage.
 *
 * Follow-the-system is the product default since K.1a (decision D9): light
 * and dark are equal citizens and dark is only the effective default on
 * systems configured dark. The pre-paint bootstrap script in `src/app.html`
 * mirrors this fallback — keep the two in sync.
 */
export const DEFAULT_MODE: ThemeMode = 'system'

function isThemeMode(value: unknown): value is ThemeMode {
  return value === 'light' || value === 'dark' || value === 'system'
}

function readStoredMode(): ThemeMode {
  if (!browser) return DEFAULT_MODE
  try {
    let stored = window.localStorage.getItem(THEME_STORAGE_KEY)
    if (stored === null) {
      // One-time migration: adopt the legacy value and write it forward.
      const legacy = window.localStorage.getItem(LEGACY_THEME_STORAGE_KEY)
      if (legacy !== null) {
        window.localStorage.setItem(THEME_STORAGE_KEY, legacy)
        stored = legacy
      }
    }
    return isThemeMode(stored) ? stored : DEFAULT_MODE
  } catch {
    // Storage unavailable (e.g. private mode): fall back to the default.
    return DEFAULT_MODE
  }
}

function systemPrefersDark(): boolean {
  return browser && window.matchMedia('(prefers-color-scheme: dark)').matches
}

/** Reactive theme state: `mode` is the user's intent, `systemPrefersDark` tracks the OS. */
export const theme = $state({
  mode: readStoredMode(),
  systemPrefersDark: systemPrefersDark(),
})

/**
 * The theme actually painted (what the `dark` class on `<html>` encodes):
 * `mode` unless it is `system`, in which case the OS preference wins.
 *
 * Exposed as a function (Svelte 5 does not allow exporting `$derived` from a
 * module); calling it inside a component template, `$derived` or `$effect`
 * tracks the underlying `theme` state, so callers re-run on theme changes.
 */
export function resolved(): ResolvedTheme {
  return theme.mode === 'system' ? (theme.systemPrefersDark ? 'dark' : 'light') : theme.mode
}

function syncThemeClass(): void {
  if (!browser) return
  document.documentElement.classList.toggle('dark', resolved() === 'dark')
}

/** Set the mode, persist it and keep the `<html class="dark">` in sync. */
export function setThemeMode(mode: ThemeMode): void {
  theme.mode = mode
  if (browser) {
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, mode)
    } catch {
      // Storage unavailable: the in-memory mode still applies for this visit.
    }
    syncThemeClass()
  }
}

if (browser) {
  // Follow OS-level light/dark changes while in `system` mode.
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (event) => {
    theme.systemPrefersDark = event.matches
    syncThemeClass()
  })

  // Cross-tab sync: another tab changing the setting updates this one too.
  window.addEventListener('storage', (event) => {
    if (event.key === THEME_STORAGE_KEY) {
      theme.mode = readStoredMode()
      syncThemeClass()
    }
  })

  // Reassert the class on boot in case the pre-paint script and the store
  // ever disagree (e.g. storage changed between the two).
  syncThemeClass()
}
