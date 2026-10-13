<script lang="ts">
  import { untrack } from 'svelte'
  import { toast } from '$lib/stores/toast.svelte'
  import {
    adminApi,
    healthApi,
    type ActivityItem,
    type HealthEvent,
    type HealthSummary,
  } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import { formatDuration } from '$lib/format'
  import { priceRefresh } from '$lib/stores/priceRefresh.svelte'
  import { ChevronRight } from 'lucide-svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte'
  import JobDetailPanel from '$lib/components/domain/JobDetailPanel.svelte'
  import { JOB_TARGET_KEYS, JOB_TYPE_KEYS, labeled, statusBadge } from '$lib/components/domain/job-status'
  import SegmentedControl from '$lib/components/ui/SegmentedControl.svelte'
  import Select from '$lib/components/ui/Select.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
  import StatCard from '$lib/components/ui/StatCard.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'

  const PAGE_SIZE = 50

  // `$derived` so a locale switch live-updates the segmented control labels
  // (same recipe as the preferences page theme items). The period scopes both
  // the summary tiles and the activity feed, so the numbers always match the
  // listed rows. `all` = no lower bound.
  const periodItems = $derived([
    { value: '24h', label: t('health.period24h') },
    { value: '7d', label: t('health.period7d') },
    { value: 'all', label: t('health.periodAll') },
  ])

  let period = $state('all')
  let offset = $state(0)
  /** Server-side activity filters (`''` = "All", omitted from the query). */
  let filterType = $state('')
  let filterStatus = $state('')

  // shortcut: hard-coded union of the job types (`JOB_TYPE_KEYS`) and the
  // backend event types; move to a server-provided distinct list if it drifts.
  const EVENT_TYPES = [
    'price_refresh',
    'fx_fetch',
    'fx_history_fetch',
    'fx_history_save',
    'history_fetch',
    'history_save',
    'history_backfill',
    'split_fetch',
    'split_save',
    'search',
    'asset_meta',
    'asset_profile',
    'meta_backfill',
    'etf_exposure',
    'etf_morningstar_exposure',
    'etf_search',
  ]
  const ACTIVITY_TYPES = [...new Set([...Object.keys(JOB_TYPE_KEYS), ...EVENT_TYPES])]

  // The statuses of BOTH feed kinds, deduplicated (`running`/`partial` are
  // shared); labels come from the one shared status→badge map. `$derived` so a
  // locale switch live-updates the option labels.
  const ACTIVITY_STATUSES = ['queued', 'running', 'done', 'failed', 'partial', 'success', 'failure']
  const STATUS_OPTIONS = $derived(ACTIVITY_STATUSES.map((s) => ({ value: s, label: statusBadge(s).label })))

  let summary = $state<HealthSummary | null>(null)
  let activity = $state<ActivityItem[]>([])
  let activityTotal = $state(0)
  let loading = $state(true)
  /** Open flag of the destructive "Clear activity" confirm (#206). */
  let clearOpen = $state(false)

  /** Job inspected in the detail panel (#226), a snapshot BY VALUE of the
   *  activity row: the feed pages newest-first, so the panel keys on
   *  `openJobId` and merges each poll's fresh copy by id. */
  let openJob = $state<ActivityItem | null>(null)
  const openJobId = $derived(openJob?.id)
  let jobEvents = $state<HealthEvent[]>([])
  let jobEventsLoading = $state(false)
  let jobEventsError = $state<string | null>(null)

  const rangeLabel = $derived(
    activity.length === 0
      ? t('common.rangeEmpty', { total: activityTotal })
      : t('common.rangeLabel', { from: offset + 1, to: offset + activity.length, total: activityTotal }),
  )
  // Any open queue job in the feed (or an app-wide price refresh in flight)
  // keeps the 3s auto-refresh alive; the pill and the interval both key off it.
  const hasOpenJob = $derived(
    priceRefresh.refreshing ||
      activity.some((item) => item.kind === 'job' && (item.status === 'queued' || item.status === 'running')),
  )
  /** No data in the period: the summary tiles all show `N/A` together. */
  const noData = $derived(!summary?.has_data)

  /** The Messaggio cell: a job shows its error, else the progress text; an
   *  event shows its message (empty renders the shared em-dash). */
  function activityMessage(item: ActivityItem): string {
    if (item.kind === 'event') return item.message
    if (item.error) return item.error
    return item.total > 0 ? t('jobs.progressOf', { processed: item.processed, total: item.total }) : ''
  }

  async function fetchSummary(silent = false) {
    if (!silent) loading = true
    try {
      summary = (await healthApi.summary(period)).summary
    } catch {
      // Poll errors stay silent: the previous data keeps rendering.
      if (!silent) toast.error(t('health.loadFailed'))
    } finally {
      if (!silent) loading = false
    }
  }

  async function fetchActivity(silent = false) {
    if (!silent) loading = true
    try {
      const data = await healthApi.activity({ period, limit: PAGE_SIZE, offset, type: filterType, status: filterStatus })
      activity = data.items ?? []
      activityTotal = data.items_total ?? 0
    } catch {
      if (!silent) toast.error(t('health.loadFailed'))
    } finally {
      if (!silent) loading = false
    }
  }

  // The open job's events: scoped server-side by `job_id`, no client filter.
  async function fetchJobEvents(silent = false): Promise<void> {
    const job = openJob
    if (!job) return
    if (!silent) {
      jobEventsLoading = true
      jobEventsError = null
    }
    try {
      const data = await healthApi.jobEvents(job.id)
      // The panel was closed or swapped to another job while this ran: drop.
      if (openJob?.id !== job.id) return
      jobEvents = data.events ?? []
      jobEventsError = null
    } catch (e) {
      // Silent poll errors keep the previous list rendering (like the feed).
      if (!silent && openJob?.id === job.id) {
        jobEventsError = e instanceof Error ? e.message : String(e)
      }
    } finally {
      if (!silent && openJob?.id === job.id) jobEventsLoading = false
    }
  }

  // Period tracks both the summary and the feed; offset/filters track the feed.
  $effect(() => {
    void fetchSummary()
  })
  $effect(() => {
    void fetchActivity()
  })

  $effect(() => {
    if (!hasOpenJob) return
    const timer = setInterval(() => {
      void fetchActivity(true)
      // A running/queued open job refreshes its events on the SAME tick —
      // no second interval. Reads here are outside the effect's tracking.
      const job = openJob
      if (job && (job.status === 'running' || job.status === 'queued')) void fetchJobEvents(true)
    }, 3000)
    return () => clearInterval(timer)
  })

  // Open → none transition: ONE silent refetch so the just-finished job's
  // lifecycle rows land in the open panel (the feed row itself is already
  // terminal: this tick's poll is the fetch that flipped the flag).
  let hadOpenJob = false
  $effect(() => {
    const open = hasOpenJob
    if (hadOpenJob && !open) void fetchJobEvents(true)
    hadOpenJob = open
  })

  // `openJobId` (a stable string, not the merged snapshot) loads the panel's
  // events; `untrack` keeps the 3s snapshot merges from re-firing this fetch.
  $effect(() => {
    const id = openJobId
    if (!id) return
    untrack(() => void fetchJobEvents())
  })

  // Keep the open job's snapshot live: merge the polled page's fresh copy of
  // the open row by id (a row off the current page keeps its last snapshot).
  $effect(() => {
    const fresh = activity.find((item) => item.kind === 'job' && item.id === openJob?.id)
    if (fresh && fresh !== openJob) openJob = fresh
  })

  function refreshAll() {
    void fetchSummary()
    void fetchActivity()
  }

  // Deletes every health event and sync job row, then reloads both views from
  // page one (the non-GET already invalidated the client GET cache). The jobs
  // are gone, so a detail panel open on one closes with them.
  async function confirmClearEvents(): Promise<void> {
    try {
      const { deleted } = await adminApi.clearHealthEvents()
      toast.success(t('health.clearEventsDone', { count: deleted }))
      offset = 0
      openJob = null
      await fetchSummary()
      await fetchActivity()
    } catch {
      toast.error(t('health.clearEventsFailed'))
    }
  }

  function openJobDetail(item: ActivityItem): void {
    openJob = item
    jobEvents = []
    jobEventsError = null
  }

  function getPeriod() {
    return period
  }

  function setPeriod(value: string) {
    // The feed is windowed by the period too: back to page one, and the
    // tracked fetchActivity effect (which reads `period`) refetches.
    period = value
    offset = 0
  }

  function formatRate(val: number | null | undefined) {
    if (val === null || val === undefined || Number.isNaN(val)) return 'N/A'
    return (val * 100).toFixed(1) + '%'
  }
</script>

<div class="mx-auto max-w-6xl p-3 sm:p-6">
  <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold text-foreground">{t('health.title')}</h1>
      <p class="text-muted-foreground">{t('health.subtitle')}</p>
    </div>
    <div class="flex flex-wrap items-center gap-3">
      <Button variant="secondary" onclick={refreshAll} disabled={loading}>
        {loading ? t('health.refreshing') : t('health.refresh')}
      </Button>
      <Button variant="secondary" onclick={() => (clearOpen = true)} disabled={loading}>
        {t('health.clearEvents')}
      </Button>
    </div>
  </div>

  {#if loading}
    <div class="flex justify-center py-12">
      <Spinner size="lg" class="text-accent-text" />
    </div>
  {:else if !summary}
    <div class="py-12 text-center text-muted-foreground">
      {t('health.noData')}
    </div>
  {:else}
    <div class="overflow-hidden rounded-card border border-border bg-surface shadow-card">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border bg-muted px-2 py-4 sm:px-6">
        <div class="flex items-center gap-2">
          <h2 class="font-semibold text-foreground">{t('health.activity')}</h2>
          {#if hasOpenJob}
            <Badge variant="accent">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current" aria-hidden="true"></span>
              {t('jobs.live')}
            </Badge>
          {/if}
        </div>
        <!-- The period selector scopes the summary tiles AND the activity
             feed below, so the numbers always match the listed rows. -->
        <SegmentedControl items={periodItems} bind:value={getPeriod, setPeriod} ariaLabel={t('health.periodAria')} />
      </div>
      <div class="grid grid-cols-1 gap-4 px-2 pt-4 sm:px-6 md:grid-cols-4 max-sm:grid-cols-2 max-sm:gap-2">
        <StatCard
          label={t('health.successRate')}
          value={noData ? 'N/A' : formatRate(summary.success_rate)}
          valueClass={noData
            ? 'text-2xl text-muted-foreground'
            : `text-2xl ${summary.success_rate > 0.9 ? 'text-positive' : 'text-warning'}`}
        />
        <StatCard
          label={t('health.totalSuccesses')}
          value={noData ? 'N/A' : String(summary.successes)}
          valueClass={noData ? 'text-2xl text-muted-foreground' : 'text-2xl'}
        />
        <StatCard
          label={t('health.totalFailures')}
          value={noData ? 'N/A' : String(summary.failures)}
          valueClass={noData ? 'text-2xl text-muted-foreground' : 'text-2xl text-negative'}
        />
        <StatCard
          label={t('health.rateLimited')}
          value={noData ? 'N/A' : String(summary.rate_limited)}
          valueClass={noData ? 'text-2xl text-muted-foreground' : 'text-2xl text-warning'}
        />
      </div>
      <!-- Server-side activity filters: the feed shows every job plus the
           events not tied to one; changing either filter resets to page one
           and the tracked fetchActivity effect refetches. -->
      <div class="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-border px-2 pt-4 sm:px-6">
        <label class="flex items-center gap-2 text-sm text-muted-foreground">
          {t('common.colType')}
          <Select bind:value={filterType} onchange={() => (offset = 0)} class="w-40">
            <option value="">{t('health.filterAll')}</option>
            {#each ACTIVITY_TYPES as et (et)}
              <option value={et}>{et}</option>
            {/each}
          </Select>
        </label>
        <label class="flex items-center gap-2 text-sm text-muted-foreground">
          {t('positions.colStatus')}
          <Select bind:value={filterStatus} onchange={() => (offset = 0)} class="w-32">
            <option value="">{t('health.filterAll')}</option>
            {#each STATUS_OPTIONS as opt (opt.value)}
              <option value={opt.value}>{opt.label}</option>
            {/each}
          </Select>
        </label>
      </div>
      <div class="overflow-x-auto px-1.5 pb-4 sm:px-6">
        <!-- Mobile (< sm): the 6-column table collapses to a two-line grid
             row — line 1 = job badge + type + "- " + destination, line 2 =
             date/time | duration/status | truncated message. The six Td cells
             keep their desktop markup and are re-placed with col-start/
             row-start; duration joins the status cell (duration first, badge
             second) and the duration column hides. -->
        <Table class="max-sm:block" aria-label={t('health.activity')}>
          <THead class="max-sm:hidden">
            <Tr>
              <Th class="whitespace-nowrap">{t('health.colTimestamp')}</Th>
              <Th>{t('common.colType')}</Th>
              <Th>{t('jobs.colTarget')}</Th>
              <Th>{t('positions.colStatus')}</Th>
              <Th>{t('health.colMessage')}</Th>
              <Th align="right">{t('health.colDuration')}</Th>
            </Tr>
          </THead>
          <TBody class="max-sm:block">
            {#each activity as item (item.id)}
              <Tr class="max-sm:grid max-sm:grid-cols-[auto_auto_1fr] max-sm:gap-x-2 max-sm:py-2">
                <Td class="whitespace-nowrap text-muted-foreground max-sm:whitespace-normal max-sm:py-0.5 max-sm:text-xs max-sm:col-start-1 max-sm:row-start-2">
                  <time datetime={item.at}>
                    <span class="max-sm:hidden">{new Date(item.at).toLocaleString()}</span>
                    <span class="hidden max-sm:block">{new Date(item.at).toLocaleDateString()}</span>
                    <span class="hidden max-sm:block">{new Date(item.at).toLocaleTimeString()}</span>
                  </time>
                </Td>
                <Td class="font-medium max-sm:py-0.5 max-sm:text-[11px] max-sm:col-start-1 max-sm:col-span-2 max-sm:row-start-1 max-sm:min-w-0">
                  {#if item.kind === 'job'}
                    <div class="flex items-center gap-1.5">
                      <Badge variant="outline">job</Badge>
                      <!-- The real <button> is the job-row affordance (rows are
                           never click-only); `h-8` keeps the touch target usable. -->
                      <Button
                        variant="link"
                        class="h-8 max-sm:text-[11px]"
                        aria-label={t('jobs.viewDetailNamed', { type: labeled(JOB_TYPE_KEYS, item.type) })}
                        onclick={() => openJobDetail(item)}
                      >
                        {labeled(JOB_TYPE_KEYS, item.type)}
                        <ChevronRight class="h-3.5 w-3.5" aria-hidden="true" />
                      </Button>
                      <!-- Line 1 reads "type - destination": the separator is
                           mobile-only, the destination lives in its own cell. -->
                      <span aria-hidden="true" class="hidden max-sm:inline">-</span>
                    </div>
                  {:else}
                    <span class="max-sm:flex max-sm:items-center max-sm:gap-1.5">
                      {item.type}
                      <span aria-hidden="true" class="hidden max-sm:inline">-</span>
                    </span>
                  {/if}
                </Td>
                <Td class="text-muted-foreground max-sm:min-w-0 max-sm:py-0.5 max-sm:text-xs max-sm:col-start-3 max-sm:row-start-1">
                  {#if item.target_type === 'global'}
                    {t('jobs.targetGlobal')}
                  {:else}
                    {#if item.target_type}
                      <span class="whitespace-nowrap max-sm:whitespace-normal">{labeled(JOB_TARGET_KEYS, item.target_type)}</span>
                    {/if}
                    {#if item.target_label}
                      <span class="ml-1 font-medium text-foreground max-sm:ml-0" title={item.target_id ?? undefined}>
                        {item.target_label}
                      </span>
                    {/if}
                    {#if !item.target_type && !item.target_label}—{/if}
                  {/if}
                </Td>
                <Td class="max-sm:flex max-sm:min-w-0 max-sm:flex-col max-sm:items-start max-sm:gap-0.5 max-sm:py-0.5 max-sm:text-xs max-sm:col-start-2 max-sm:row-start-2">
                  <!-- Durata stays visible on mobile: the first line of the
                       status section (the duration column itself is hidden). -->
                  <span class="hidden max-sm:inline max-sm:text-muted-foreground max-sm:tabular-nums">
                    {formatDuration(item.duration_ms)}
                  </span>
                  <Badge variant={statusBadge(item.status).variant}>
                    {#if item.status === 'running'}
                      <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current" aria-hidden="true"></span>
                    {/if}
                    {statusBadge(item.status).label}
                  </Badge>
                </Td>
                <Td class="max-w-md truncate text-muted-foreground max-sm:max-w-none max-sm:min-w-0 max-sm:py-0.5 max-sm:text-xs max-sm:col-start-3 max-sm:row-start-2" title={activityMessage(item)}>
                  {activityMessage(item) || '—'}
                </Td>
                <Td align="right" class="text-muted-foreground tabular-nums max-sm:hidden">
                  {formatDuration(item.duration_ms)}
                </Td>
              </Tr>
            {/each}
          </TBody>
        </Table>
      </div>
      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-border px-2 py-4 sm:px-6">
        <span class="text-sm text-muted-foreground tabular-nums">{rangeLabel}</span>
        <div class="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            disabled={offset === 0}
            onclick={() => (offset = Math.max(0, offset - PAGE_SIZE))}
          >
            {t('common.previous')}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            disabled={offset + PAGE_SIZE >= activityTotal}
            onclick={() => (offset += PAGE_SIZE)}
          >
            {t('common.next')}
          </Button>
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Job detail panel (#226): Drawer ≥ lg / Sheet below, fed by this page
     (`fetchJobEvents` scoped by `job_id` + the live `openJob` snapshot). -->
<JobDetailPanel
  open={openJob !== null}
  onClose={() => (openJob = null)}
  job={openJob}
  events={jobEvents}
  loading={jobEventsLoading}
  error={jobEventsError}
  onRetry={() => void fetchJobEvents()}
/>

<!-- Destructive wipe of the whole health events table (#206): the dialog
     shows the current total and the busy lock while the DELETE runs. -->
<ConfirmDialog
  bind:open={clearOpen}
  title={t('health.clearEventsTitle')}
  message={t('health.clearEventsConfirm', { total: activityTotal })}
  confirmLabel={t('health.clearEvents')}
  variant="danger"
  onconfirm={confirmClearEvents}
/>
