<script lang="ts">
  import { toast } from '$lib/stores/toast.svelte'
  import { adminApi, type ServerSettings } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import AdminGate from '$lib/components/domain/AdminGate.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import Card from '$lib/components/ui/Card.svelte'
  import EmptyState from '$lib/components/ui/EmptyState.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
  import Switch from '$lib/components/ui/Switch.svelte'

  /**
   * Admin → Server settings (issue #57 Phase A): the one server-wide switch
   * for now — `auto_approve_registrations`. Toggling it applies immediately
   * (optimistic write through `PATCH /admin/settings`, rolled back with an
   * error toast on failure), matching the live-apply pattern of the user
   * Preferences page. When off, new signups land as `pending` and are
   * approved from the Users page. The gate keeps the page presentationally
   * admin-only; the endpoints are 403-gated server-side anyway.
   */

  let settings = $state<ServerSettings | null>(null)
  let loading = $state(true)
  let loadError = $state(false)
  let saving = $state(false)

  async function fetchSettings(): Promise<void> {
    loading = true
    loadError = false
    try {
      settings = await adminApi.getSettings()
    } catch {
      loadError = true
      toast.error(t('admin.settingsLoadFailed'))
    } finally {
      loading = false
    }
  }

  $effect(() => {
    void fetchSettings()
  })

  async function toggleAutoApprove(next: boolean): Promise<void> {
    if (!settings) return
    const previous = settings.auto_approve_registrations
    settings.auto_approve_registrations = next
    saving = true
    try {
      settings = await adminApi.updateSettings(next)
      toast.success(t('admin.settingSaved'))
    } catch {
      settings = { ...settings, auto_approve_registrations: previous }
      toast.error(t('admin.settingSaveFailed'))
    } finally {
      saving = false
    }
  }

  const updatedLabel = $derived(
    settings
      ? t('admin.lastUpdated', {
          time: new Date(settings.updated_at).toLocaleString(),
        })
      : '',
  )
</script>

{#snippet retryAction()}
  <Button variant="secondary" size="sm" onclick={fetchSettings}>{t('common.retry')}</Button>
{/snippet}

<AdminGate>
  <div class="mx-auto max-w-2xl p-4 lg:p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-foreground">{t('nav.serverSettings')}</h1>
      <p class="text-muted-foreground">{t('admin.settingsSubtitle')}</p>
    </div>

    {#if loading}
      <div class="flex justify-center py-12">
        <Spinner size="lg" class="text-accent-text" />
      </div>
    {:else if loadError || !settings}
      <EmptyState
        dashed
        title={t('admin.settingsLoadFailed')}
        description={t('admin.loadFailedHint')}
        action={retryAction}
      />
    {:else}
      <Card class="p-6">
        <div class="flex items-start justify-between gap-6">
          <div class="min-w-0">
            <h2 class="font-semibold text-foreground">{t('admin.autoApprove')}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{t('admin.autoApproveHint')}</p>
            <p class="mt-2 text-xs text-muted-foreground">{updatedLabel}</p>
          </div>
          <Switch
            checked={settings.auto_approve_registrations}
            onchange={toggleAutoApprove}
            ariaLabel={t('admin.autoApprove')}
            disabled={saving}
          />
        </div>
      </Card>
    {/if}
  </div>
</AdminGate>
