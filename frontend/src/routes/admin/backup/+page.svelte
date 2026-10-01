<script lang="ts">
  import { Download, Upload } from 'lucide-svelte'
  import { toast } from '$lib/stores/toast.svelte'
  import { adminApi, errorStatus, saveBlob, type DBRestoreSummary } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import AdminGate from '$lib/components/domain/AdminGate.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import Card from '$lib/components/ui/Card.svelte'
  import Field from '$lib/components/ui/Field.svelte'
  import Input from '$lib/components/ui/Input.svelte'
  import Modal from '$lib/components/ui/Modal.svelte'

  /**
   * Admin → Server backup (issue #57 Phase D): whole-database dump and
   * restore for this instance, via the admin-only `GET /admin/db/backup` and
   * `POST /admin/db/restore` (the gate also keeps the page presentationally
   * admin-only). Download streams the custom-format archive through the same
   * blob path as the per-user backup (`adminApi.dbBackup`) and saves it
   * under the server-suggested filename.
   *
   * The restore is DESTRUCTIVE SERVER-WIDE: it rewrites every table, users
   * included, so the current session may stop working — the success result
   * keeps a persistent re-login notice, and the request only goes out after
   * the admin types the confirmation phrase (`admin.dbConfirmPhrase`,
   * localized) into a danger modal. On failure the backend message is
   * surfaced on purpose: 500 carries the sanitized `pg_restore` stderr,
   * because a partially applied restore must be readable in the UI, not
   * hunted in server logs; the localized generic is only the fallback.
   *
   * Multipart contract (`dump` file + `confirm=replace` field) lives in
   * `adminApi.dbRestore`; a .dump archive is never inspected client-side.
   */

  const confirmPhrase = $derived(t('admin.dbConfirmPhrase'))

  let downloading = $state(false)

  // Restore state: the chosen archive (opaque), the in-flight flag, the
  // type-to-confirm modal and the last successful summary.
  let fileInput = $state<HTMLInputElement | null>(null)
  let dumpFile = $state<File | null>(null)
  let restoring = $state(false)
  let confirmOpen = $state(false)
  let typedConfirm = $state('')
  let result = $state<DBRestoreSummary | null>(null)

  async function downloadDump(): Promise<void> {
    downloading = true
    try {
      const { blob, filename } = await adminApi.dbBackup()
      saveBlob(blob, `peculium-db-${new Date().toISOString().split('T')[0]}.dump`, filename)
      toast.success(t('admin.dbDownloaded'))
    } catch {
      toast.error(t('admin.dbDownloadFailed'))
    } finally {
      downloading = false
    }
  }

  function onFileSelected(e: Event): void {
    const input = e.target as HTMLInputElement
    dumpFile = input.files?.[0] ?? null
    input.value = ''
    // A fresh pick invalidates any previous result/confirmation.
    result = null
    typedConfirm = ''
  }

  function chooseFile(): void {
    fileInput?.click()
  }

  function openConfirm(): void {
    if (!dumpFile || restoring) return
    typedConfirm = ''
    confirmOpen = true
  }

  async function submitRestore(): Promise<void> {
    if (!dumpFile) return
    restoring = true
    try {
      result = await adminApi.dbRestore(dumpFile)
      toast.success(t('admin.dbRestored'))
      // The archive is spent; force a new, deliberate pick before a second
      // restore can fire — especially since the session may now be invalid.
      dumpFile = null
      confirmOpen = false
    } catch (err: unknown) {
      // Server-side failures (400/500) carry the message the admin must read;
      // errors without an HTTP status (network) fall back to the generic one.
      const message =
        errorStatus(err) && err instanceof Error && err.message
          ? err.message
          : t('admin.dbRestoreFailed')
      toast.error(message)
    } finally {
      restoring = false
    }
  }

  /** Archive-size label for the result panel. Display-only and used once;
   * the money/percent centralization rules of `lib/format.ts` don't apply. */
  function formatBytes(bytes: number): string {
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.min(units.length - 1, Math.max(0, Math.floor(Math.log2(bytes + 1) / 10)))
    return `${i === 0 ? bytes : (bytes / 1024 ** i).toFixed(1)} ${units[i]}`
  }
</script>

<AdminGate>
  <div class="mx-auto max-w-2xl p-4 lg:p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-foreground">{t('nav.serverBackup')}</h1>
      <p class="text-muted-foreground">{t('admin.dbSubtitle')}</p>
    </div>

    <div class="space-y-6">
      <Card class="p-6">
        <h2 class="mb-1 font-semibold text-foreground">{t('admin.dbDownloadTitle')}</h2>
        <p class="mb-4 text-sm text-muted-foreground">{t('admin.dbDownloadHint')}</p>
        <Button onclick={downloadDump} loading={downloading}>
          <Download class="h-4 w-4" />
          {t('admin.dbDownloadAction')}
        </Button>
        {#if downloading}
          <p class="mt-3 text-xs text-muted-foreground">{t('admin.dbLongOperation')}</p>
        {/if}
      </Card>

      <!-- Destructive action card: the negative banner follows the same
           recipe as the import modal's error strip. -->
      <Card class="p-6">
        <h2 class="mb-1 font-semibold text-foreground">{t('admin.dbRestoreTitle')}</h2>
        <p class="mb-4 rounded-control border border-negative/20 bg-negative/10 px-4 py-2 text-sm text-negative">
          {t('admin.dbRestoreDanger')}
        </p>
        <p class="mb-4 text-sm text-muted-foreground">{t('admin.dbRestoreHint')}</p>

        {#if dumpFile}
          <p class="mb-4 rounded-control bg-muted px-3 py-2 text-sm">
            <span class="text-muted-foreground">{t('backup.selectedFile')}</span>
            <span class="font-medium break-all">{dumpFile.name}</span>
          </p>
          <!-- Stacked full-width on phones (the /settings/backup pattern). -->
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
            <Button variant="secondary" onclick={chooseFile} disabled={restoring} class="w-full sm:w-auto">
              <Upload class="h-4 w-4" />
              {t('admin.dbChooseAnotherFile')}
            </Button>
            <Button variant="danger" onclick={openConfirm} loading={restoring} class="w-full sm:w-auto">
              {t('admin.dbRestoreAction')}
            </Button>
          </div>
        {:else}
          <div class="flex flex-col items-center gap-3 py-4 text-center">
            <p class="text-sm text-muted-foreground">{t('admin.dbRestorePickHint')}</p>
            <Button variant="secondary" onclick={chooseFile} disabled={restoring}>
              <Upload class="h-4 w-4" />
              {t('admin.dbChooseFile')}
            </Button>
          </div>
        {/if}

        {#if restoring}
          <p class="mt-3 text-xs text-muted-foreground">{t('admin.dbLongOperation')}</p>
        {/if}

        {#if result}
          <div class="mt-4 rounded-control border border-border p-4">
            <p class="mb-1 text-sm font-semibold text-positive">{t('admin.dbRestored')}</p>
            <p class="text-sm text-foreground">
              {t('admin.dbDumpSize', { bytes: formatBytes(result.dump_bytes) })}
            </p>
            <p class="mt-2 text-xs text-warning">{t('admin.dbReloginNotice')}</p>
          </div>
        {/if}
      </Card>
    </div>
  </div>
</AdminGate>

<!-- Hidden dump picker, one per page (same pattern as the other file flows). -->
<input
  type="file"
  accept=".dump,application/octet-stream"
  class="hidden"
  bind:this={fileInput}
  onchange={onFileSelected}
/>

<!-- Type-to-confirm gate of the destructive server-wide restore. -->
<Modal
  bind:open={confirmOpen}
  title={t('admin.dbConfirmTitle')}
  description={t('admin.dbConfirmWarning')}
  size="sm"
  dismissible={!restoring}
>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (confirmOpen = false)} disabled={restoring}>
      {t('common.cancel')}
    </Button>
    <Button
      variant="danger"
      onclick={submitRestore}
      loading={restoring}
      disabled={typedConfirm !== confirmPhrase}
    >
      {t('admin.dbRestoreAction')}
    </Button>
  {/snippet}
  <Field label={t('admin.dbConfirmTypeLabel', { phrase: confirmPhrase })}>
    <Input
      bind:value={typedConfirm}
      autocomplete="off"
      spellcheck="false"
      autocapitalize="none"
      placeholder={confirmPhrase}
      disabled={restoring}
    />
  </Field>
</Modal>
