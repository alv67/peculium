<script lang="ts">
  import { t } from '$lib/i18n/index.svelte'
  import { appVersion } from '$lib/stores/appVersion.svelte'
  import { toast } from '$lib/stores/toast.svelte'
  import Button from '../ui/Button.svelte'
  import Card from '../ui/Card.svelte'
  import { cx } from '../ui/utils'

  /**
   * Build/version indicator (#198): one component, two weights.
   * - chip (`detail=false`): static mono text, never focusable (no
   *   button/anchor), for the sidebar footer and the login page; when both
   *   version and build are absent it renders nothing;
   * - card (`detail=true`): the Settings → Preferences definition list with
   *   the commit link and a copy-for-issue button; empty fields show the
   *   "not available" label instead of hiding the row.
   * Purely presentational: the shell/login call `loadAppVersion()` once on
   * mount and this component reads the shared store.
   */
  let {
    detail = false,
    /** Rail variant of the chip: only the version, truncated (the caller
     * decides — it knows its own width). */
    compact = false,
  }: { detail?: boolean; compact?: boolean } = $props()

  // `built_at` is ISO UTC; `Date` parses it and `toLocaleString()` renders
  // the user's local date+time. Unparseable strings render nothing.
  const builtAtLocal = $derived.by(() => {
    if (!appVersion.builtAt) return ''
    const date = new Date(appVersion.builtAt)
    return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
  })

  const display = $derived(
    compact
      ? appVersion.version
      : [appVersion.version, builtAtLocal].filter(Boolean).join(' · '),
  )

  /** One-liner for issue reports, e.g.
   * `Peculium v1.0.0 (a1b2c3d) built 2026-10-03T12:00:00Z`. */
  const buildInfo = $derived(
    [
      'Peculium',
      appVersion.version,
      appVersion.commit && `(${appVersion.commit})`,
      appVersion.builtAt && `built ${appVersion.builtAt}`,
    ]
      .filter(Boolean)
      .join(' '),
  )

  async function copyBuildInfo(): Promise<void> {
    try {
      await navigator.clipboard.writeText(buildInfo)
      toast.success(t('about.copied'))
    } catch {
      toast.error(t('about.copyFailed'))
    }
  }

  const unavailable = $derived(t('about.unavailable'))
</script>

{#if detail}
  <Card class="mt-6 max-w-lg p-6">
    <h2 class="mb-4 font-semibold">{t('about.title')}</h2>
    <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
      <dt class="text-muted-foreground">{t('about.version')}</dt>
      <dd class={cx('font-mono', !appVersion.version && 'text-muted-foreground')}>
        {appVersion.version || unavailable}
      </dd>
      <dt class="text-muted-foreground">{t('about.commit')}</dt>
      <dd class={cx('font-mono', !appVersion.commit && 'text-muted-foreground')}>
        {#if appVersion.commit}
          <a
            class="text-accent-text hover:underline"
            href={`https://github.com/alv67/peculium/commit/${appVersion.commit}`}
            target="_blank"
            rel="noopener noreferrer"
          >
            {appVersion.commit}
            <span class="sr-only">{t('about.openCommit')}</span>
          </a>
        {:else}
          {unavailable}
        {/if}
      </dd>
      <dt class="text-muted-foreground">{t('about.build')}</dt>
      <dd class={cx(!builtAtLocal && 'text-muted-foreground')}>{builtAtLocal || unavailable}</dd>
    </dl>
    <div class="mt-4">
      <Button variant="secondary" onclick={copyBuildInfo}>{t('about.copy')}</Button>
      <p class="mt-2 text-xs text-muted-foreground">{t('about.hint')}</p>
    </div>
  </Card>
{:else if display}
  <!-- Static text, never in the tab order: the sr-only span carries the
       announced accessible name (the technical glyph run is aria-hidden so
       readers don't spell the separators/dates). -->
  <span class="block min-w-0 truncate font-mono text-xs text-muted-foreground" title={display}>
    <span class="sr-only">{t('about.chip', { version: display })}</span>
    <span aria-hidden="true">{display}</span>
  </span>
{/if}
