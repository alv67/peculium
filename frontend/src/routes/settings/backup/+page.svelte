<script lang="ts">
  import { Download, Upload } from 'lucide-svelte'
  import { toast } from '$lib/stores/toast.svelte'
  import {
    backupApi,
    type BackupRestoreMode,
    type BackupRestoreSummary,
  } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import SettingsTabs from '$lib/components/domain/SettingsTabs.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import Card from '$lib/components/ui/Card.svelte'
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte'

  /**
   * Settings → Backup & restore (issue #56): export the whole account as one
   * JSON bundle and re-import it, choosing how it lands. Download goes
   * through `backupApi.download` (raw authenticated fetch, blob) and triggers
   * the browser save with the server-suggested filename when present. Restore
   * reads a chosen file client-side first (valid JSON, supported version) so
   * obvious mistakes are caught before any request, then POSTs
   * `/backup/restore?mode=…`: `add` is the non-destructive default; `replace`
   * deletes the account's portfolios first and therefore requires an
   * explicit danger confirmation before sending. A successful restore shows
   * the counts summary inline (and as a success toast); server 400s map to
   * the localized invalid-bundle message — raw backend strings never reach
   * the UI. Assets are global: the backend reuses existing ones and
   * recreates missing ones, nothing asset-level is ever deleted.
   */

  // Download state.
  let downloading = $state(false)

  // Restore state: the parsed bundle + the file it came from, the import
  // strategy (defaults to the non-destructive one), and the in-flight flag.
  let fileInput = $state<HTMLInputElement | null>(null)
  let bundle = $state<unknown>(null)
  let bundleName = $state('')
  let mode = $state<BackupRestoreMode>('add')
  let restoring = $state(false)
  let confirmOpen = $state(false)
  let result = $state<BackupRestoreSummary | null>(null)

  function errorStatus(err: unknown): number | undefined {
    return err instanceof Error && 'status' in err
      ? (err as Error & { status: number }).status
      : undefined
  }

  async function downloadBackup(): Promise<void> {
    downloading = true
    try {
      const { blob, filename } = await backupApi.download()
      // Same create/revoke dance as the portfolio export (client-side blob
      // download); the name only ever comes from the server's
      // Content-Disposition or a local date-stamped fallback.
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download =
        filename || `peculium-backup-${new Date().toISOString().split('T')[0]}.json`
      a.click()
      URL.revokeObjectURL(url)
      toast.success(t('backup.downloaded'))
    } catch {
      // A 401 was already handled by the refresh flow inside `backupApi`;
      // anything else surfaces as the localized failure (raw backend
      // strings never reach the UI).
      toast.error(t('backup.downloadFailed'))
    } finally {
      downloading = false
    }
  }

  async function onFileSelected(e: Event): Promise<void> {
    const input = e.target as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file) return
    result = null
    // JSON.parse and the shape gate get distinct messages: an unparseable
    // file is `invalidFile`, parseable-but-wrong is `invalidBundle`.
    let parsed: unknown
    try {
      parsed = JSON.parse(await file.text()) as unknown
    } catch {
      bundle = null
      bundleName = ''
      toast.error(t('backup.invalidFile'))
      return
    }
    // Light client-side gate mirroring the server's own checks: a bundle
    // must be an object at the supported format version (anything else is
    // a guaranteed 400 — say so before touching the API).
    if (
      typeof parsed !== 'object' ||
      parsed === null ||
      Array.isArray(parsed) ||
      (parsed as { version?: unknown }).version !== 1
    ) {
      bundle = null
      bundleName = ''
      toast.error(t('backup.invalidBundle'))
      return
    }
    bundle = parsed
    bundleName = file.name
    // Every fresh file starts on the safe strategy.
    mode = 'add'
  }

  function chooseFile(): void {
    fileInput?.click()
  }

  function requestRestore(): void {
    if (!bundle || restoring) return
    // Destructive strategy: never sent without the explicit confirmation.
    if (mode === 'replace') {
      confirmOpen = true
      return
    }
    void submitRestore()
  }

  async function submitRestore(): Promise<void> {
    if (!bundle) return
    restoring = true
    try {
      result = await backupApi.restore(bundle, mode)
      toast.success(t('backup.done'))
      // The loaded file is spent; force an explicit re-pick before any
      // second restore could fire.
      bundle = null
      bundleName = ''
      mode = 'add'
    } catch (err: unknown) {
      // 400 = the server did not accept the bundle (version/format); other
      // statuses fall back to the generic localized failure.
      toast.error(errorStatus(err) === 400 ? t('backup.invalidBundle') : t('backup.restoreFailed'))
    } finally {
      restoring = false
    }
  }
</script>

<div class="p-6">
  <h1 class="mb-6 text-2xl font-bold">{t('nav.settings')}</h1>

  <SettingsTabs class="mb-6" />

  <div class="max-w-2xl space-y-6">
    <Card class="p-6">
      <h2 class="mb-1 font-semibold">{t('backup.downloadTitle')}</h2>
      <p class="mb-4 text-sm text-muted-foreground">{t('backup.downloadHint')}</p>
      <Button onclick={downloadBackup} loading={downloading}>
        <Download class="h-4 w-4" />
        {t('backup.downloadAction')}
      </Button>
    </Card>

    <Card class="p-6">
      <h2 class="mb-1 font-semibold">{t('backup.restoreTitle')}</h2>
      <p class="mb-4 text-sm text-muted-foreground">{t('backup.restoreHint')}</p>

      {#if bundle}
        <p class="mb-4 rounded-control bg-muted px-3 py-2 text-sm">
          <span class="text-muted-foreground">{t('backup.selectedFile')}</span>
          <span class="font-medium break-all">{bundleName}</span>
        </p>

        <fieldset class="mb-4 space-y-2">
          <legend class="sr-only">{t('backup.restoreAction')}</legend>
          <label class="flex items-start gap-2 text-sm">
            <input type="radio" class="mt-1" bind:group={mode} value="add" disabled={restoring} />
            <span>
              <span class="font-medium">{t('backup.modeAdd')}</span>
              <span class="block text-muted-foreground">{t('backup.modeAddHint')}</span>
            </span>
          </label>
          <label class="flex items-start gap-2 text-sm">
            <input
              type="radio"
              class="mt-1"
              bind:group={mode}
              value="replace"
              disabled={restoring}
            />
            <span>
              <span class="font-medium">{t('backup.modeReplace')}</span>
              <span class="block text-muted-foreground">{t('backup.modeReplaceHint')}</span>
            </span>
          </label>
        </fieldset>

        <!-- Stacked full-width on phones (same pattern as the other settings
             forms), inline from `sm`. -->
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
          <Button variant="secondary" onclick={chooseFile} disabled={restoring} class="w-full sm:w-auto">
            <Upload class="h-4 w-4" />
            {t('backup.chooseAnotherFile')}
          </Button>
          <Button
            variant={mode === 'replace' ? 'danger' : 'primary'}
            onclick={requestRestore}
            loading={restoring}
            class="w-full sm:w-auto"
          >
            {t('backup.restoreAction')}
          </Button>
        </div>
      {:else}
        <div class="flex flex-col items-center gap-3 py-4 text-center">
          <p class="text-sm text-muted-foreground">{t('backup.restorePickHint')}</p>
          <Button variant="secondary" onclick={chooseFile} disabled={restoring}>
            <Upload class="h-4 w-4" />
            {t('backup.chooseFile')}
          </Button>
        </div>
      {/if}

      {#if result}
        <div class="mt-4 rounded-control border border-border p-4">
          <p class="mb-2 text-sm font-semibold text-foreground">{t('backup.resultTitle')}</p>
          <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-4">
            <div>
              <dt class="text-muted-foreground">{t('backup.resultPortfolios')}</dt>
              <dd class="font-medium tabular-nums">{result.portfolios_created}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">{t('backup.resultTransactions')}</dt>
              <dd class="font-medium tabular-nums">{result.transactions_created}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">{t('backup.resultAssetsCreated')}</dt>
              <dd class="font-medium tabular-nums">{result.assets_created}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">{t('backup.resultAssetsReused')}</dt>
              <dd class="font-medium tabular-nums">{result.assets_reused}</dd>
            </div>
          </dl>
        </div>
      {/if}
    </Card>
  </div>
</div>

<!-- Hidden restore picker, one per page (ImportPortfolioModal pattern). -->
<input
  type="file"
  accept=".json,application/json"
  class="hidden"
  bind:this={fileInput}
  onchange={onFileSelected}
/>

<!-- Destructive replace-mode gate. -->
<ConfirmDialog
  bind:open={confirmOpen}
  title={t('backup.confirmTitle')}
  message={t('backup.confirmMessage')}
  confirmLabel={t('backup.restoreAction')}
  variant="danger"
  loading={restoring}
  onconfirm={submitRestore}
/>
