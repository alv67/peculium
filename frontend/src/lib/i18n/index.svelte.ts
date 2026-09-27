import { browser } from '$app/environment'
import { en, type Dictionary } from './en'
import { it } from './it'

/**
 * Tiny rune-based i18n runtime (EPIC K.1b, decision D1): two dictionaries +
 * a reactive locale, no external dependency.
 *
 * - dictionaries (`en.ts` / `it.ts`) are two-level nested objects
 *   (`group.key`), flattened here into dot-joined lookup keys
 *   (`nav.dashboard`); `MessageKey` is the compile-time union of them all,
 *   so `t()` calls are typo-checked;
 * - the active locale lives in `localStorage['peculium-locale']` (read with
 *   a one-time fallback to the legacy storage key) and falls back to
 *   `DEFAULT_LOCALE` (Italian) when unset/invalid; English is the
 *   fallback language: a string missing in the active locale resolves from
 *   `en` before giving up, and a fully unknown key returns the key itself
 *   (never throws, dev warning only);
 * - `t()` reads the reactive `locale`, so any component using it re-renders
 *   on `setLocale()`.
 *
 * The app is SPA-only (`ssr = false` in `routes/+layout.ts`), so reading
 * localStorage at init never runs on the server. The boot block at the
 * bottom mirrors `stores/theme.svelte.ts`: apply `<html lang>` before the
 * first render and keep it in sync across tabs.
 */
export const SUPPORTED_LOCALES = ['it', 'en'] as const

export type Locale = (typeof SUPPORTED_LOCALES)[number]

/** Italian is the product default (decision D1); `app.html` ships matching. */
export const DEFAULT_LOCALE: Locale = 'it'

export const LOCALE_STORAGE_KEY = 'peculium-locale'

/** Legacy storage key: when the new key is absent, the value stored here is
 *  adopted and written forward once, so no saved preference is lost. */
export const LEGACY_LOCALE_STORAGE_KEY = 'vaultlab-locale'

/**
 * Dot-joined lookup key union (e.g. `'nav.dashboard'`), derived from the
 * canonical English dictionary. Only two nesting levels are supported —
 * matching the dictionary files.
 */
export type MessageKey = {
  [G in keyof Dictionary & string]: {
    [K in keyof Dictionary[G] & string]: `${G}.${K}`
  }[keyof Dictionary[G] & string]
}[keyof Dictionary & string]

/** Runtime type guard for values coming from storage or `<select>` inputs. */
export function isLocale(value: unknown): value is Locale {
  return (SUPPORTED_LOCALES as readonly unknown[]).includes(value)
}

function flatten(dict: Dictionary): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [group, entries] of Object.entries(dict)) {
    for (const [key, value] of Object.entries(entries)) {
      out[`${group}.${key}`] = value
    }
  }
  return out
}

// English is both the canonical shape (type level) and the fallback
// language (runtime level), so `en` is always consulted last.
const dictionaries: Record<Locale, Record<string, string>> = {
  it: flatten(it),
  en: flatten(en),
}

function readStoredLocale(): Locale {
  if (!browser) return DEFAULT_LOCALE
  try {
    let stored = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    if (stored === null) {
      // One-time migration: adopt the legacy value and write it forward.
      const legacy = window.localStorage.getItem(LEGACY_LOCALE_STORAGE_KEY)
      if (legacy !== null) {
        window.localStorage.setItem(LOCALE_STORAGE_KEY, legacy)
        stored = legacy
      }
    }
    return isLocale(stored) ? stored : DEFAULT_LOCALE
  } catch {
    // Storage unavailable (e.g. private mode): fall back to the default.
    return DEFAULT_LOCALE
  }
}

/** Reactive locale state: `locale.current` is the active language. */
export const locale = $state({ current: readStoredLocale() })

/**
 * Translate a dot-joined key, interpolating `{name}` placeholders from
 * `params` (a placeholder without a matching param is left verbatim).
 * Falls back to English, then to the key itself — never throws.
 */
export function t(key: MessageKey, params?: Record<string, string | number>): string {
  const template = dictionaries[locale.current][key] ?? dictionaries.en[key]
  if (template === undefined) {
    if (import.meta.env.DEV) {
      console.warn(`[i18n] missing key "${key}" for locale "${locale.current}"`)
    }
    return key
  }
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  )
}

function applyDocumentLang(): void {
  if (browser) document.documentElement.lang = locale.current
}

/** Set the locale, persist it and keep `<html lang>` in sync. */
export function setLocale(next: Locale): void {
  locale.current = next
  if (browser) {
    try {
      window.localStorage.setItem(LOCALE_STORAGE_KEY, next)
    } catch {
      // Storage unavailable: the in-memory locale still applies for this visit.
    }
    applyDocumentLang()
  }
}

if (browser) {
  // Boot: the static `app.html` ships DEFAULT_LOCALE; reflect the persisted
  // one before the first page render.
  applyDocumentLang()

  // Cross-tab sync: another tab changing the language updates this one too
  // (`t()` tracks `locale.current`, so the whole UI re-renders).
  window.addEventListener('storage', (event) => {
    if (event.key === LOCALE_STORAGE_KEY && isLocale(event.newValue)) {
      locale.current = event.newValue
      applyDocumentLang()
    }
  })
}
