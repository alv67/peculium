/** Payload of the public `GET /api/v1/version` (extra fields tolerated). */
interface VersionPayload {
  version?: string
  commit?: string
  built_at?: string
}

/**
 * Build metadata of the running app (#198), module-scoped like
 * `dashboardStatus`: one shared fetch, every surface (sidebar rail, More
 * sheet, login, Settings card) reads the same reactive state.
 *
 * Plain `fetch` instead of `request`: the endpoint is public (works
 * pre-login, no token) and immutable for the whole visit, so it must not
 * join the investment GET cache — whose whole design is being cleared on
 * every mutation. Failure is silent and terminal for the visit: the empty
 * state simply means "nothing to render".
 */
export const appVersion = $state({
  /** Tag/branch (`v1.0.0`, `develop`, `dev`); empty when absent. */
  version: '',
  /** Short commit SHA; may be empty. */
  commit: '',
  /** Raw ISO-8601 UTC build time; may be empty. */
  builtAt: '',
})

let loading: Promise<void> | null = null

/** Idempotent loader: the first call fires the single GET, every later call
 * returns the same promise. Never throws, never toasts. */
export function loadAppVersion(): Promise<void> {
  loading ??= fetch('/api/v1/version')
    .then((res) => (res.ok ? (res.json() as Promise<VersionPayload>) : null))
    .then((data) => {
      appVersion.version = data?.version ?? ''
      appVersion.commit = data?.commit ?? ''
      appVersion.builtAt = data?.built_at ?? ''
    })
    .catch(() => {
      // Unreachable / older backend: stay absent (no chip, no error).
    })
  return loading
}
