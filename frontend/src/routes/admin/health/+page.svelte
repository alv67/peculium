<script lang="ts">
  import { toast } from '$lib/stores/toast.svelte'
  import { api, jobsApi, type Job } from '$lib/services/api'
  import { t, type MessageKey } from '$lib/i18n/index.svelte'
  import { formatDuration } from '$lib/format'
  import { priceRefresh, refreshPrices } from '$lib/stores/priceRefresh.svelte'
  import { AlertTriangle, ListChecks, ListFilter, X } from 'lucide-svelte'
  import AsyncCard from '$lib/components/ui/AsyncCard.svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import Card from '$lib/components/ui/Card.svelte'
  import CardHeader from '$lib/components/ui/CardHeader.svelte'
  import EmptyState from '$lib/components/ui/EmptyState.svelte'
  import SegmentedControl from '$lib/components/ui/SegmentedControl.svelte'
  import Skeleton from '$lib/components/ui/Skeleton.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
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

  interface HealthEvent {
    id: string
    asset_id: string
    event_type: string
    status: string
    code: string
    message: string
    duration_ms: number
    error_code: string
    created_at: string
    /** Set on lifecycle/health rows emitted by a queue job (issue #195). */
    job_id?: string
  }

  const PAGE_SIZE = 50

  // ponytail: job events are filtered client-side over the newest 500 rows;
  // move to a server-side job_id query only if a job ever emits more.
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
  let summary = $state<HealthSummary | null>(null)
  let events = $state<HealthEvent[]>([])
  let eventsTotal = $state(0)
  let loading = $state(true)

  let jobs = $state<Job[]>([])
  let jobsLoading = $state(true)
  let jobsError = $state(false)
  let jobsAt = $state<Date | null>(null)
  /** Active job filter on the events card (`type` is the localized label). */
  let selectedJob = $state<{ id: string; type: string } | null>(null)

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
  const visibleEvents = $derived(
    selectedJob ? events.filter((e) => e.job_id === selectedJob?.id) : events,
  )

  async function fetchHealth(silent = false) {
    if (!silent) loading = true
    try {
      const limit = selectedJob ? JOB_FILTER_LIMIT : PAGE_SIZE
      const off = selectedJob ? 0 : offset
      const data = (await api.get(
        `/health/prices?period=${period}&limit=${limit}&offset=${off}`,
      )) as {
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
    const timer = setInterval(() => void fetchJobs(true), 3000)
    return () => clearInterval(timer)
  })

  // Open → none transition: ONE silent events refetch so the just-closed
  // job's lifecycle rows land in the events card (not on every poll).
  let hadOpenJob = false
  $effect(() => {
    const open = hasOpenJob
    if (hadOpenJob && !open) void fetchHealth(true)
    hadOpenJob = open
  })

  function refreshAll() {
    fetchHealth()
    fetchJobs()
  }

  function filterToJob(id?: string, type?: string) {
    if (!id) return
    selectedJob = { id, type: type || 'job' }
    // Default (auto) scroll: no motion to neutralise for prefers-reduced-motion.
    document.getElementById('events-card')?.scrollIntoView()
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

  // The badge is the one backend value we localise, with ONE map shared by
  // the jobs and the events tables: known statuses map to a Badge variant +
  // dictionary label (the raw `running` no longer renders red here), the 429
  // heuristic covers rate-limited codes, anything unexpected falls back to
  // the raw value in the negative tone. The label text is always present.
  type BadgeVariant = 'neutral' | 'accent' | 'positive' | 'negative' | 'warning'
  const STATUS_BADGES: Record<string, { variant: BadgeVariant; key: MessageKey }> = {
    done: { variant: 'positive', key: 'jobs.statusDone' },
    success: { variant: 'positive', key: 'health.statusSuccess' },
    running: { variant: 'accent', key: 'jobs.statusRunning' },
    queued: { variant: 'neutral', key: 'jobs.statusQueued' },
    partial: { variant: 'warning', key: 'jobs.statusPartial' },
    failed: { variant: 'negative', key: 'jobs.statusFailed' },
    failure: { variant: 'negative', key: 'health.statusFailure' },
    rate_limited: { variant: 'warning', key: 'health.statusRateLimited' },
  }

  function statusBadge(status: string): { variant: BadgeVariant; label: string } {
    if (status.includes('429')) return { variant: 'warning', label: t('health.statusRateLimited') }
    const known = STATUS_BADGES[status]
    if (known) return { variant: known.variant, label: t(known.key) }
    return { variant: 'negative', label: status }
  }

  const JOB_TYPE_KEYS: Record<string, MessageKey> = {
    price_refresh: 'jobs.typePriceRefresh',
    history_backfill: 'jobs.typeHistoryBackfill',
    meta_backfill: 'jobs.typeMetaBackfill',
    splits_fetch: 'jobs.typeSplitsFetch',
  }

  const JOB_TARGET_KEYS: Record<string, MessageKey> = {
    asset: 'jobs.targetAsset',
    portfolio: 'jobs.targetPortfolio',
    global: 'jobs.targetGlobal',
  }

  function labeled(keys: Record<string, MessageKey>, value: string | undefined) {
    if (!value) return ''
    const key = keys[value]
    return key ? t(key) : value
  }
</script>

<div class="mx-auto max-w-6xl p-6">
  <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold text-foreground">{t('health.title')}</h1>
      <p class="text-muted-foreground">{t('health.subtitle')}</p>
    </div>
    <div class="flex flex-wrap items-center gap-3">
      <SegmentedControl items={periodItems} bind:value={getPeriod, setPeriod} ariaLabel={t('health.periodAria')} />
      <Button variant="secondary" onclick={refreshAll} disabled={loading || jobsLoading}>
        {loading ? t('health.refreshing') : t('health.refresh')}
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
                <Th class="w-40">{t('common.colType')}</Th>
                <Th class="w-36">{t('jobs.colTarget')}</Th>
                <Th class="w-28">{t('positions.colStatus')}</Th>
                <Th class="w-48">{t('jobs.colProgress')}</Th>
                <Th class="w-24">{t('health.colDuration')}</Th>
                <Th>{t('jobs.colStarted')}</Th>
              </Tr>
            </THead>
            <TBody class="max-sm:block">
              {#each jobs as job (job.id)}
                <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2 align-top">
                  <Td class="max-sm:order-1 max-sm:col-span-2 max-sm:py-0.5 min-w-0">
                    <div class="flex items-center justify-between gap-2">
                      <span class="truncate font-medium">
                        {labeled(JOB_TYPE_KEYS, job.type)}
                      </span>
                      <Button
                        variant="ghost"
                        size="icon"
                        class="shrink-0"
                        aria-label={t('jobs.viewEventsNamed', { type: labeled(JOB_TYPE_KEYS, job.type) })}
                        onclick={() => filterToJob(job.id, labeled(JOB_TYPE_KEYS, job.type))}
                      >
                        <ListFilter class="h-4 w-4" aria-hidden="true" />
                      </Button>
                    </div>
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
                </Tr>
                {#if (job.status === 'failed' || job.status === 'partial') && job.error}
                  <Tr class="max-sm:grid max-sm:grid-cols-2 max-sm:gap-x-4 max-sm:py-2">
                    <Td colspan={6} class="max-sm:col-span-2 max-sm:block max-sm:py-0.5">
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
    <div class="mb-3 text-sm text-muted-foreground">{t('health.periodLabel', { period: periodLabel })}</div>
    <div class="mb-8 grid grid-cols-1 gap-4 md:grid-cols-4">
      <div class="rounded-card border border-border bg-surface p-4 shadow-card">
        <div class="mb-1 text-sm text-muted-foreground">{t('health.successRate')}</div>
        {#if !summary.has_data}
          <div class="text-2xl font-bold tabular-nums text-muted-foreground">N/A</div>
        {:else}
          <div class="text-2xl font-bold tabular-nums {summary.success_rate > 0.9 ? 'text-positive' : 'text-warning'}">
            {formatRate(summary.success_rate)}
          </div>
        {/if}
      </div>
      <div class="rounded-card border border-border bg-surface p-4 shadow-card">
        <div class="mb-1 text-sm text-muted-foreground">{t('health.totalSuccesses')}</div>
        <div class="text-2xl font-bold tabular-nums text-foreground">{summary.successes}</div>
      </div>
      <div class="rounded-card border border-border bg-surface p-4 shadow-card">
        <div class="mb-1 text-sm text-muted-foreground">{t('health.totalFailures')}</div>
        <div class="text-2xl font-bold tabular-nums text-negative">{summary.failures}</div>
      </div>
      <div class="rounded-card border border-border bg-surface p-4 shadow-card">
        <div class="mb-1 text-sm text-muted-foreground">{t('health.rateLimited')}</div>
        <div class="text-2xl font-bold tabular-nums text-warning">{summary.rate_limited}</div>
      </div>
    </div>

    <div id="events-card" class="overflow-hidden rounded-card border border-border bg-surface shadow-card">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border bg-muted px-6 py-4">
        <h2 class="font-semibold text-foreground">{t('health.recentEvents')}</h2>
        {#if selectedJob}
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="accent">
              {t('health.jobFilter', { type: selectedJob.type })}
              · {t('health.jobFilterCount', { count: visibleEvents.length })}
            </Badge>
            {#if eventsTotal > JOB_FILTER_LIMIT}
              <span class="text-xs text-muted-foreground">
                {t('health.jobFilterCap', { limit: JOB_FILTER_LIMIT })}
              </span>
            {/if}
            <Button
              variant="ghost"
              size="icon"
              title={t('health.jobFilterClear')}
              aria-label={t('health.jobFilterClearNamed')}
              onclick={() => (selectedJob = null)}
            >
              <X class="h-4 w-4" aria-hidden="true" />
            </Button>
          </div>
        {/if}
      </div>
      <div class="overflow-x-auto px-6 pb-4">
        {#if selectedJob && visibleEvents.length === 0}
          <p class="py-8 text-center text-sm text-muted-foreground">
            {t('health.jobFilterNone', { limit: JOB_FILTER_LIMIT })}
          </p>
        {:else}
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
              {#each visibleEvents as event (event.id)}
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
                        <Button
                          variant="ghost"
                          size="icon"
                          class="h-6 w-6"
                          aria-label={t('health.filterByJob')}
                          onclick={() =>
                            filterToJob(
                              event.job_id,
                              labeled(JOB_TYPE_KEYS, jobs.find((j) => j.id === event.job_id)?.type),
                            )}
                        >
                          <ListFilter class="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
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
                  <Td class="max-w-xs truncate text-muted-foreground">{event.message || '—'}</Td>
                  <Td align="right" class="text-muted-foreground max-sm:hidden">
                    {formatDuration(event.duration_ms)}
                  </Td>
                </Tr>
              {/each}
            </TBody>
          </Table>
        {/if}
      </div>
      {#if !selectedJob}
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
      {/if}
    </div>
  {/if}
</div>
