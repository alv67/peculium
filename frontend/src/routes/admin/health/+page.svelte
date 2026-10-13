<script lang="ts">
  import { untrack } from 'svelte'
  import { toast } from '$lib/stores/toast.svelte'
  import { api, adminApi, jobsApi, type HealthEvent, type Job } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import { formatDuration } from '$lib/format'
  import { priceRefresh, refreshPrices } from '$lib/stores/priceRefresh.svelte'
  import { viewport } from '$lib/stores/viewport.svelte'
  import { AlertTriangle, ChevronRight, ListChecks, ListFilter } from 'lucide-svelte'
  import AsyncCard from '$lib/components/ui/AsyncCard.svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import Card from '$lib/components/ui/Card.svelte'
  import CardHeader from '$lib/components/ui/CardHeader.svelte'
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte'
  import EmptyState from '$lib/components/ui/EmptyState.svelte'
  import JobDetailPanel from '$lib/components/domain/JobDetailPanel.svelte'
  import { JOB_TARGET_KEYS, JOB_TYPE_KEYS, labeled, statusBadge } from '$lib/components/domain/job-status'
  import SegmentedControl from '$lib/components/ui/SegmentedControl.svelte'
  import Select from '$lib/components/ui/Select.svelte'
  import Skeleton from '$lib/components/ui/Skeleton.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
  import StatCard from '$lib/components/ui/StatCard.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'

  interface HealthSummary {
    successes: number
    failures: number
    success_rate: number
    rate_limited: number
    has_data: boolean
    period?: string
  }

  const PAGE_SIZE = 50

  // ponytail: the job detail panel's events are filtered client-side over the
  // newest 500 rows; move to a server-side job_id query only if a job ever
  // emits more.
  const JOB_FILTER_LIMIT = 500

  // `$derived` so a locale switch live-updates the segmented control labels
  // (same recipe as the preferences page theme items).
  const periodItems = $derived([
    { value: 'today', label: t('health.periodToday') },
    { value: '24h', label: t('health.periodLast24h') },
    { value: '100', label: t('health.periodLast100') },
  ])

  let period = $state('today')
  let offset = $state(0)
  /** Server-side event filters (`''` = "All", omitted from the query). */
  let filterType = $state('')
  let filterStatus = $state('')
  // shortcut: hard-coded list of backend event types; move to a
  // server-provided distinct list if it ever drifts.
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
  // `$derived` so a locale switch live-updates the option labels; values are
  // the exact `status` strings accepted by the health query.
  const STATUS_OPTIONS = $derived([
    { value: 'success', label: t('health.statusSuccess') },
    { value: 'failure', label: t('health.statusFailure') },
    { value: 'running', label: t('health.statusRunning') },
    { value: 'partial', label: t('health.statusPartial') },
  ])
  let summary = $state<HealthSummary | null>(null)
  let events = $state<HealthEvent[]>([])
  let eventsTotal = $state(0)
  let loading = $state(true)
  /** Open flag of the destructive "Clear events" confirm (#206). */
  let clearOpen = $state(false)

  let jobs = $state<Job[]>([])
  let jobsLoading = $state(true)
  let jobsError = $state(false)
  let jobsAt = $state<Date | null>(null)
  /** Job inspected in the detail panel (#226), a snapshot BY VALUE: the jobs
   *  list is a newest-25 sliding window, so the panel keys on `openJobId` and
   *  merges each poll's fresh copy by id instead of storing only an id. */
  let openJob = $state<Job | null>(null)
  const openJobId = $derived(openJob?.id)
  let jobEvents = $state<HealthEvent[]>([])
  let jobEventsLoading = $state(false)
  let jobEventsError = $state<string | null>(null)
  /** True when the events total exceeded the client-side 500-row window. */
  let jobEventsCapped = $state(false)

  const periodLabel = $derived(periodItems.find((item) => item.value === period)?.label ?? period)
  const rangeLabel = $derived(
    events.length === 0
      ? t('common.rangeEmpty', { total: eventsTotal })
      : t('common.rangeLabel', { from: offset + 1, to: offset + events.length, total: eventsTotal }),
  )
  // Any open queue job (or an app-wide price refresh in flight) keeps the
  // 3s auto-refresh alive; the pill and the interval both key off this.
  const hasOpenJob = $derived(
    priceRefresh.refreshing || jobs.some((j) => j.status === 'queued' || j.status === 'running'),
  )
  const openCounts = $derived({
    running: jobs.filter((j) => j.status === 'running').length,
    queued: jobs.filter((j) => j.status === 'queued').length,
    failed: jobs.filter((j) => j.status === 'failed').length,
  })
  /** No data in the period: the summary tiles all show `N/A` together. */
  const noData = $derived(!summary?.has_data)

  async function fetchHealth(silent = false) {
    if (!silent) loading = true
    try {
      let url = `/health/prices?period=${period}&limit=${PAGE_SIZE}&offset=${offset}`
      if (filterType) url += `&type=${filterType}`
      if (filterStatus) url += `&status=${filterStatus}`
      const data = (await api.get(url)) as {
        summary: HealthSummary
        events: HealthEvent[]
        events_total: number
      }
      summary = data.summary
      events = data.events ?? []
      eventsTotal = data.events_total ?? 0
    } catch {
      // Poll errors stay silent: the previous data keeps rendering.
      if (!silent) toast.error(t('health.loadFailed'))
    } finally {
      if (!silent) loading = false
    }
  }

  async function fetchJobs(silent = false) {
    if (!silent) {
      jobsLoading = true
      jobsError = false
    }
    try {
      jobs = await jobsApi.list({ limit: 25 })
      jobsAt = new Date()
    } catch {
      if (!silent) jobsError = true
    } finally {
      if (!silent) jobsLoading = false
    }
  }

  // The open job's events: the same period-scoped feed, filtered client-side
  // down to the job's rows (see the `JOB_FILTER_LIMIT` ponytail note).
  async function fetchJobEvents(silent = false): Promise<void> {
    const job = openJob
    if (!job) return
    if (!silent) {
      jobEventsLoading = true
      jobEventsError = null
    }
    try {
      const data = (await api.get(
        `/health/prices?period=${period}&limit=${JOB_FILTER_LIMIT}`,
      )) as { summary: HealthSummary; events: HealthEvent[]; events_total: number }
      // The panel was closed or swapped to another job while this ran: drop.
      if (openJob?.id !== job.id) return
      jobEvents = (data.events ?? []).filter((e) => e.job_id === job.id)
      jobEventsCapped = (data.events_total ?? 0) > JOB_FILTER_LIMIT
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

  $effect(() => {
    fetchHealth()
  })

  // Initial load runs through the visible `jobsLoading` skeleton; locale
  // changes only re-render the labels (`jobsError` stays a boolean flag).
  $effect(() => {
    fetchJobs()
  })

  $effect(() => {
    if (!hasOpenJob) return
    const timer = setInterval(() => {
      void fetchJobs(true)
      // A running/queued open job refreshes its events on the SAME tick —
      // no second interval. Reads here are outside the effect's tracking.
      const job = openJob
      if (job && (job.status === 'running' || job.status === 'queued')) void fetchJobEvents(true)
    }, 3000)
    return () => clearInterval(timer)
  })

  // Open → none transition: ONE silent refetch so the just-finished job's
  // lifecycle rows land in the feed and, when open, in the detail panel.
  let hadOpenJob = false
  $effect(() => {
    const open = hasOpenJob
    if (hadOpenJob && !open) {
      void fetchHealth(true)
      void fetchJobEvents(true)
    }
    hadOpenJob = open
  })

  // `openJobId` (a stable string, not the merged snapshot) and `period`
  // (the events window is period-scoped) load the panel's events; `untrack`
  // keeps the 3s snapshot merges from re-firing this fetch.
  $effect(() => {
    const id = openJobId
    void period
    if (!id) return
    untrack(() => void fetchJobEvents())
  })

  // Keep the open job's snapshot live: `jobsApi.list` is a newest-25 sliding
  // window, so a burst of new jobs would otherwise slide the open one out and
  // freeze its progress. Assigning the found copy settles the effect (the
  // next run resolves the same object reference).
  $effect(() => {
    const fresh = jobs.find((j) => j.id === openJob?.id)
    if (fresh && fresh !== openJob) openJob = fresh
  })

  function refreshAll() {
    fetchHealth()
    fetchJobs()
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
      await fetchHealth()
      await fetchJobs()
    } catch {
      toast.error(t('health.clearEventsFailed'))
    }
  }

  function openJobDetail(job: Job): void {
    openJob = job
    jobEvents = []
    jobEventsError = null
    jobEventsCapped = false
  }

  function getPeriod() {
    return period
  }

  function setPeriod(value: string) {
    period = value
    offset = 0
  }

  function formatRate(val: number | null | undefined) {
    if (val === null || val === undefined || Number.isNaN(val)) return 'N/A'
    return (val * 100).toFixed(1) + '%'
  }
</script>

<div class="mx-auto max-w-6xl p-6">
  <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold text-foreground">{t('health.title')}</h1>
      <p class="text-muted-foreground">{t('health.subtitle')}</p>
    </div>
    <div class="flex flex-wrap items-center gap-3">
      <Button variant="secondary" onclick={refreshAll} disabled={loading || jobsLoading}>
        {loading ? t('health.refreshing') : t('health.refresh')}
      </Button>
      <Button variant="secondary" onclick={() => (clearOpen = true)} disabled={loading}>
        {t('health.clearEvents')}
      </Button>
    </div>
  </div>

  <Card class="mb-8">
    <CardHeader title={t('jobs.title')} subtitle={t('jobs.scopeNote')}>
      {#snippet actions()}
        {#if hasOpenJob}
          <Badge variant="accent">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current" aria-hidden="true"></span>
            {t('jobs.live')}
          </Badge>
        {/if}
        {#if jobsAt}
          <span class="text-xs tabular-nums text-muted-foreground">
            {t('jobs.lastUpdate', { time: jobsAt.toLocaleTimeString() })}
          </span>
        {/if}
      {/snippet}
    </CardHeader>
    <div class="px-6 pt-3">
      <AsyncCard
        loading={jobsLoading}
        error={jobsError ? t('jobs.loadFailed') : null}
        onRetry={() => fetchJobs()}
        empty={jobs.length === 0}
      >
        {#snippet skeleton()}
          <div class="space-y-2 pb-6" aria-hidden="true">
            <Skeleton class="h-8 w-full" />
            <Skeleton class="h-8 w-full" />
            <Skeleton class="h-8 w-full" />
            <Skeleton class="h-8 w-2/3" />
          </div>
        {/snippet}
        {#snippet emptyContent()}
          <EmptyState icon={ListChecks} title={t('jobs.empty')} description={t('jobs.emptyHint')} class="p-6">
            {#snippet action()}
              <Button
                variant="secondary"
                size="sm"
                onclick={() => void refreshPrices({ announceSuccess: true })}
              >
                {t('quickActions.refreshPrices')}
              </Button>
            {/snippet}
          </EmptyState>
        {/snippet}

        <div class="flex flex-wrap items-center gap-x-4 gap-y-1 pb-3 text-xs">
          {#if openCounts.running > 0}
            <span class="font-medium text-accent-text">{t('jobs.countRunning', { count: openCounts.running })}</span>
          {/if}
          {#if openCounts.queued > 0}
            <span class="text-muted-foreground">{t('jobs.countQueued', { count: openCounts.queued })}</span>
          {/if}
          {#if openCounts.failed > 0}
            <span class="text-negative">{t('jobs.countFailed', { count: openCounts.failed })}</span>
          {/if}
          {#if openCounts.running === 0 && openCounts.queued === 0}
            <span class="text-muted-foreground">{t('jobs.idle')}</span>
          {/if}
        </div>

        <div class="overflow-x-auto pb-6">
          <Table class="max-sm:block table-fixed">
            <caption class="sr-only">{t('jobs.caption')}</caption>
            <THead class="max-sm:hidden">
              <Tr>
                <Th class="w-52">{t('common.colType')}</Th>
                <Th class="w-36">{t('jobs.colTarget')}</Th>
                <Th class="w-28">{t('positions.colStatus')}</Th>
                <Th class="w-48">{t('jobs.colProgress')}</Th>
                <Th class="w-24">{t('health.colDuration')}</Th>
                <Th class="w-44">{t('jobs.colStarted')}</Th>
                <Th class="w-12">
                  <span class="sr-only">{t('common.colActions')}</span>
                </Th>
              </Tr>
            </THead>
            <TBody class="max-sm:block">
              {#each jobs as job (job.id)}
                <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2 align-top">
                  <Td class="max-sm:order-1 max-sm:col-span-2 max-sm:py-0.5 min-w-0">
                    <span class="font-medium" title={labeled(JOB_TYPE_KEYS, job.type)}>
                      {labeled(JOB_TYPE_KEYS, job.type)}
                    </span>
                  </Td>
                  <Td class="max-sm:order-2 max-sm:col-span-2 max-sm:py-0.5 min-w-0 text-muted-foreground">
                    {#if job.target_type || job.target_id}
                      <span class="whitespace-nowrap">{labeled(JOB_TARGET_KEYS, job.target_type)}</span>
                      {#if job.target_label}
                        <span class="ml-1 font-medium text-foreground" title={job.target_id}>
                          {job.target_label}
                        </span>
                      {:else if job.target_id}
                        <span class="ml-1 break-all font-mono text-xs" title={job.target_id}>
                          {job.target_id.slice(0, 8)}
                        </span>
                      {/if}
                    {:else}
                      —
                    {/if}
                  </Td>
                  <Td class="max-sm:order-3 max-sm:py-0.5">
                    <Badge variant={statusBadge(job.status).variant}>
                      {#if job.status === 'running'}
                        <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current" aria-hidden="true"></span>
                      {/if}
                      {statusBadge(job.status).label}
                    </Badge>
                  </Td>
                  <Td class="max-sm:order-5 max-sm:col-span-2 max-sm:py-0.5 min-w-0">
                    {#if job.total > 0}
                      <div class="flex items-center gap-2">
                        <span class="whitespace-nowrap tabular-nums">
                          {t('jobs.progressOf', { processed: job.processed, total: job.total })}
                        </span>
                        {#if job.status === 'running'}
                          <div class="h-1.5 w-full max-w-28 overflow-hidden rounded-full bg-muted" aria-hidden="true">
                            <div
                              class="h-full rounded-full bg-accent"
                              style="width: {Math.min(100, Math.round((job.processed / job.total) * 100))}%"
                            ></div>
                          </div>
                        {/if}
                      </div>
                    {:else}
                      <span class="tabular-nums">{job.processed > 0 ? job.processed : '—'}</span>
                    {/if}
                    {#if job.summary}
                      <div class="text-xs text-muted-foreground">
                        {t('jobs.outcome', { ok: job.summary.ok, failed: job.summary.failed })}
                      </div>
                    {/if}
                  </Td>
                  <Td class="max-sm:order-4 max-sm:py-0.5 text-muted-foreground tabular-nums">
                    {formatDuration(job.duration_ms)}
                  </Td>
                  <Td class="max-sm:order-6 max-sm:col-span-2 max-sm:py-0.5 whitespace-nowrap text-muted-foreground">
                    {#if job.started_at}
                      <time datetime={job.started_at}>{new Date(job.started_at).toLocaleString()}</time>
                    {:else}
                      —
                    {/if}
                  </Td>
                  <Td class="max-sm:order-7 max-sm:col-span-2 max-sm:py-0.5">
                    <!-- Phones get a labeled button: the 36px icon target is
                         under the 44px touch guideline. -->
                    {#if viewport.isPhone}
                      <Button size="sm" onclick={() => openJobDetail(job)}>
                        {t('jobs.viewDetail')}
                        <span class="sr-only">: {labeled(JOB_TYPE_KEYS, job.type)}</span>
                      </Button>
                    {:else}
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={t('jobs.viewDetailNamed', { type: labeled(JOB_TYPE_KEYS, job.type) })}
                        onclick={() => openJobDetail(job)}
                      >
                        <ChevronRight class="h-4 w-4" aria-hidden="true" />
                      </Button>
                    {/if}
                  </Td>
                </Tr>
                {#if (job.status === 'failed' || job.status === 'partial') && job.error}
                  <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
                    <Td colspan={7} class="max-sm:col-span-2 max-sm:block max-sm:py-0.5">
                      <div
                        class="flex items-start gap-2 py-1 {job.status === 'failed' ? 'text-negative' : 'text-warning'}"
                      >
                        <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
                        <code class="break-words font-mono text-xs">{job.error}</code>
                      </div>
                    </Td>
                  </Tr>
                {/if}
              {/each}
            </TBody>
          </Table>
        </div>
      </AsyncCard>
    </div>
  </Card>

  {#if loading}
    <div class="flex justify-center py-12">
      <Spinner size="lg" class="text-accent-text" />
    </div>
  {:else if !summary}
    <div class="py-12 text-center text-muted-foreground">
      {t('health.noData')}
    </div>
  {:else}
    <!-- The period selector scopes only this card (summary + feed), so it
         lives in the card header, not in the page header. -->
    <div class="overflow-hidden rounded-card border border-border bg-surface shadow-card">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border bg-muted px-6 py-4">
        <div>
          <h2 class="font-semibold text-foreground">{t('health.recentEvents')}</h2>
          <p class="text-sm text-muted-foreground">{t('health.periodLabel', { period: periodLabel })}</p>
        </div>
        <SegmentedControl items={periodItems} bind:value={getPeriod, setPeriod} ariaLabel={t('health.periodAria')} />
      </div>
      <div class="grid grid-cols-1 gap-4 px-6 pt-4 md:grid-cols-4">
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
      <!-- Server-side event filters (#91): changing either resets the
           pagination window; the tracked fetchHealth effect refetches. -->
      <div class="flex flex-wrap items-center gap-x-4 gap-y-2 px-6 pt-4">
        <label class="flex items-center gap-2 text-sm text-muted-foreground">
          {t('common.colType')}
          <Select bind:value={filterType} onchange={() => (offset = 0)} class="w-40">
            <option value="">{t('health.filterAll')}</option>
            {#each EVENT_TYPES as et (et)}
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
      <div class="overflow-x-auto px-6 pb-4">
        <Table aria-label={t('health.recentEvents')}>
          <THead>
            <Tr>
              <Th>{t('health.colTimestamp')}</Th>
              <Th>{t('common.colType')}</Th>
              <Th>{t('positions.colStatus')}</Th>
              <Th class="max-sm:hidden">{t('common.colCode')}</Th>
              <Th>{t('health.colMessage')}</Th>
              <Th align="right" class="max-sm:hidden">{t('health.colDuration')}</Th>
            </Tr>
          </THead>
          <TBody>
            {#each events as event (event.id)}
              <Tr class="hover:bg-muted">
                <Td class="text-muted-foreground">{new Date(event.created_at).toLocaleString()}</Td>
                <Td class="font-medium">
                  <div class="flex items-center gap-1.5">
                    {#if event.event_type === 'job'}
                      <Badge variant="outline">job</Badge>
                    {:else}
                      <span>{event.event_type}</span>
                    {/if}
                    {#if event.job_id}
                      {@const job = jobs.find((j) => j.id === event.job_id)}
                      {#if job}
                        <Button
                          variant="ghost"
                          size="icon"
                          class="h-6 w-6"
                          aria-label={t('jobs.viewDetailNamed', { type: labeled(JOB_TYPE_KEYS, job.type) })}
                          onclick={() => openJobDetail(job)}
                        >
                          <ListFilter class="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
                      {/if}
                    {/if}
                  </div>
                </Td>
                <Td>
                  <Badge variant={statusBadge(event.status).variant}>
                    {#if event.status === 'running'}
                      <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current" aria-hidden="true"></span>
                    {/if}
                    {statusBadge(event.status).label}
                  </Badge>
                </Td>
                <Td class="font-mono text-xs max-sm:hidden">{event.code || '—'}</Td>
                <Td class="max-w-md truncate text-muted-foreground" title={event.message}>
                  {event.message || '—'}
                </Td>
                <Td align="right" class="text-muted-foreground max-sm:hidden">
                  {formatDuration(event.duration_ms)}
                </Td>
              </Tr>
            {/each}
          </TBody>
        </Table>
      </div>
      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-border px-6 py-4">
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
            disabled={offset + PAGE_SIZE >= eventsTotal}
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
     (`fetchJobEvents` + the live `openJob` snapshot). -->
<JobDetailPanel
  open={openJob !== null}
  onClose={() => (openJob = null)}
  job={openJob}
  events={jobEvents}
  loading={jobEventsLoading}
  error={jobEventsError}
  onRetry={() => void fetchJobEvents()}
  capNote={jobEventsCapped ? t('health.jobEventsCap', { limit: JOB_FILTER_LIMIT }) : undefined}
/>

<!-- Destructive wipe of the whole health events table (#206): the dialog
     shows the current total and the busy lock while the DELETE runs. -->
<ConfirmDialog
  bind:open={clearOpen}
  title={t('health.clearEventsTitle')}
  message={t('health.clearEventsConfirm', { total: eventsTotal })}
  confirmLabel={t('health.clearEvents')}
  variant="danger"
  onconfirm={confirmClearEvents}
/>
