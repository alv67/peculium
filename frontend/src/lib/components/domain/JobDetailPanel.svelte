<script lang="ts">
  import { AlertTriangle } from 'lucide-svelte'
  import { formatDuration } from '#lib/format.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import type { ActivityItem, HealthEvent } from '#lib/services/api.js'
  import { viewport } from '#lib/stores/viewport.svelte.js'
  import AsyncCard from '#lib/components/ui/AsyncCard.svelte'
  import Badge from '#lib/components/ui/Badge.svelte'
  import Drawer from '#lib/components/ui/Drawer.svelte'
  import EmptyState from '#lib/components/ui/EmptyState.svelte'
  import Sheet from '#lib/components/ui/Sheet.svelte'
  import { JOB_TARGET_KEYS, JOB_TYPE_KEYS, labeled, statusBadge } from './job-status'

  /**
   * Job detail panel (issue #226): the events and outcome of one queue job,
   * opened from the `/admin/health` activity table. Renders as the right-side
   * `ui/Drawer` (40rem) at ≥ `lg` and the bottom `ui/Sheet` below it — the
   * same drill-down split as `AllocationDrillPanel` (spec §8.3, decision D4).
   *
   * Dumb component: the page owns the data and the fetches, we only present
   * the `job` snapshot (an activity row of kind `job`) plus its `events`
   * through the `ui/AsyncCard` states (skeleton / one-line error + Retry /
   * content). The events render as a list, not a `Table`: six columns never
   * fit 32–40rem, while here each item gets its own block with the full,
   * wrapping message.
   */
  let {
    open,
    onClose,
    job,
    events,
    loading,
    error = null,
    onRetry = undefined,
  }: {
    open: boolean
    /** Called on Esc, backdrop click and the ✕ (forwarded to Drawer/Sheet). */
    onClose: () => void
    /** The inspected job (an activity-feed row); `null` while the panel is closed. */
    job: ActivityItem | null
    /** The job's health events, newest first, owned by the page. */
    events: HealthEvent[]
    loading: boolean
    /** One-line cause from the failed events fetch. */
    error?: string | null
    /** Retry action for the error state. */
    onRetry?: () => void
  } = $props()

  const title = $derived(job ? labeled(JOB_TYPE_KEYS, job.type) : '')
</script>

{#if viewport.isDesktop}
  <Drawer {open} {onClose} {title} width="40rem" closeLabel={t('common.close')}>
    {@render body()}
  </Drawer>
{:else}
  <Sheet {open} {onClose} {title} closeLabel={t('common.close')}>
    {@render body()}
  </Sheet>
{/if}

{#snippet body()}
  {#if job}
    <AsyncCard {loading} {error} {onRetry}>
      <!-- Job header: status, target, duration and start time. -->
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <Badge variant={statusBadge(job.status).variant}>
            {#if job.status === 'running'}
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current" aria-hidden="true"></span>
            {/if}
            {statusBadge(job.status).label}
          </Badge>
          <p class="mt-2 text-sm text-muted-foreground">
            {labeled(JOB_TARGET_KEYS, job.target_type) || '—'}
            {#if job.target_label}
              <span class="ml-1 font-medium text-foreground" title={job.target_id ?? undefined}>{job.target_label}</span>
            {:else if job.target_id}
              <span class="ml-1 break-all font-mono text-xs" title={job.target_id}>
                {job.target_id.slice(0, 8)}
              </span>
            {/if}
          </p>
        </div>
        <div class="shrink-0 text-right text-xs text-muted-foreground">
          <p class="tabular-nums">{formatDuration(job.duration_ms)}</p>
          <time datetime={job.at}>{new Date(job.at).toLocaleString()}</time>
        </div>
      </div>

      {#if job.total > 0}
        <div class="mt-3">
          <p class="text-sm tabular-nums">
            {t('jobs.progressOf', { processed: job.processed, total: job.total })}
          </p>
          <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-muted" aria-hidden="true">
            <div
              class="h-full rounded-full bg-accent"
              style="width: {Math.min(100, Math.round((job.processed / job.total) * 100))}%"
            ></div>
          </div>
        </div>
      {/if}

      <!-- The full error, wrapping across the whole panel width. -->
      {#if job.error}
        <div class="mt-3 flex items-start gap-2 {job.status === 'failed' ? 'text-negative' : 'text-warning'}">
          <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <code class="break-words font-mono text-xs">{job.error}</code>
        </div>
      {/if}

      {#if events.length === 0}
        <EmptyState dashed title={t('jobs.noEventsYet')} class="mt-4" />
      {:else}
        <p class="mb-1 mt-4 text-sm text-muted-foreground">
          {t('jobs.detailEvents', { count: events.length })}
        </p>
        <ul class="divide-y divide-border">
          {#each events as event (event.id)}
            <li class="py-2">
              <div class="flex items-center justify-between gap-2">
                <span class="flex min-w-0 items-center gap-1.5">
                  <Badge variant={statusBadge(event.status).variant}>
                    {#if event.status === 'running'}
                      <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current" aria-hidden="true"></span>
                    {/if}
                    {statusBadge(event.status).label}
                  </Badge>
                  <span class="truncate text-sm font-medium">{event.event_type}</span>
                </span>
                <span class="shrink-0 text-xs tabular-nums text-muted-foreground">
                  {formatDuration(event.duration_ms)}
                </span>
              </div>
              <p class="mt-1 whitespace-normal text-sm text-foreground">{event.message || '—'}</p>
              <p class="mt-0.5 text-xs text-muted-foreground">
                {new Date(event.created_at).toLocaleString()}
                {#if event.code}
                  <span class="font-mono">· {event.code}</span>
                {/if}
              </p>
            </li>
          {/each}
        </ul>
      {/if}
    </AsyncCard>
  {/if}
{/snippet}
