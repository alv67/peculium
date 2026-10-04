/**
 * Session price-refresh outcome (EPIC K.3a, spec §8.8), module-scoped so it
 * survives SPA navigations: the shell triggers the refresh at most once per
 * session on any landing page and every consumer reads the result back here
 * (otherwise the freshness stamp / quality strip vanish after navigating
 * away and back, issue #119). Reactive so an in-flight refresh that resolves
 * after a remount still lands in the UI.
 *
 * `refreshPrices` is the SINGLE refresh path for the whole app: the automatic
 * session trigger, the header button, the Fab and the command palette all go
 * through it, so one shared `revision` counter lets price-derived page data
 * refetch on completion and concurrent triggers de-duplicate into the one
 * in-flight POST (never a double Yahoo hit while a refresh is already
 * running).
 */
import { t } from '$lib/i18n/index.svelte'
import { jobsApi, pricesApi, type Job } from '$lib/services/api'
import { toast } from '$lib/stores/toast.svelte'

export const priceRefresh = $state({
  /** Once-per-session guard: true as soon as the refresh has been triggered. */
  started: false,
  /** True across the whole async flow: the enqueue POST plus the job poll. */
  refreshing: false,
  /** `Job.finished_at` of the last completed refresh; empty renders nothing. */
  finishedAt: '',
  /** The last refresh finished `partial` (some quotes failed upstream).
   * The job no longer carries per-issue detail — the health page does. */
  partial: false,
  failed: false,
  /** Bumped on every completed refresh (success or failure): pages watch it
   * to refetch price-derived data after a refresh they did not trigger. */
  revision: 0,
})

/** Shared in-flight job (see the de-duplication note below). */
let inFlight: Promise<Job | null> | null = null

/**
 * Trigger a price refresh through the shared path (the POST now enqueues a
 * worker job) and publish the polled outcome on `priceRefresh`. Concurrent
 * calls return the one in-flight promise. Toast policy: the partial warning
 * always fires — the user must know the numbers may be stale — while the
 * plain success toast is opt-in (manual triggers) so the automatic session
 * refresh stays silent, and the error toast (failed job or poll timeout)
 * can be silenced by the shell's background trigger (the persistent strip
 * surfaces the failure instead). Resolves with the job, or `null` when the
 * enqueue itself failed.
 */
export function refreshPrices(opts?: {
  portfolioId?: string
  announceSuccess?: boolean
  announceError?: boolean
}): Promise<Job | null> {
  if (inFlight) return inFlight

  const run = async (): Promise<Job | null> => {
    priceRefresh.started = true
    priceRefresh.refreshing = true
    try {
      const enqueued = await pricesApi.refresh(opts?.portfolioId)
      const job = await jobsApi.wait(enqueued.job_id)
      if (job.finished_at) priceRefresh.finishedAt = job.finished_at
      priceRefresh.partial = job.status === 'partial'
      priceRefresh.failed = job.status !== 'done' && job.status !== 'partial'
      if (job.status === 'partial') {
        toast.warning(t('quickActions.refreshPartial'))
      } else if (job.status === 'done') {
        if (opts?.announceSuccess) toast.success(t('quickActions.refreshSuccess'))
      } else if (opts?.announceError !== false) {
        toast.error(t('quickActions.refreshError'))
      }
      return job
    } catch {
      priceRefresh.partial = false
      priceRefresh.failed = true
      if (opts?.announceError !== false) toast.error(t('quickActions.refreshError'))
      return null
    } finally {
      priceRefresh.refreshing = false
      priceRefresh.revision += 1
    }
  }

  const promise = run()
  inFlight = promise
  // Identity-guarded release: a stale settled promise must never block the
  // next trigger (an already-resolved `.finally` still lands as a microtask,
  // so this also covers a body that completes without suspending).
  void promise.finally(() => {
    if (inFlight === promise) inFlight = null
  })
  return promise
}
