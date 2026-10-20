<script lang="ts">
  import { toast } from '#lib/stores/toast.svelte.js'
  import { authApi } from '#lib/services/api.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import SettingsTabs from '#lib/components/domain/SettingsTabs.svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import Card from '#lib/components/ui/Card.svelte'
  import Field from '#lib/components/ui/Field.svelte'
  import Input from '#lib/components/ui/Input.svelte'

  let currentPassword = $state('')
  let newPassword = $state('')
  let confirmPassword = $state('')
  let savingPassword = $state(false)

  let currentError = $state<string | undefined>(undefined)
  let newPasswordError = $state<string | undefined>(undefined)
  let confirmPasswordError = $state<string | undefined>(undefined)

  function errorStatus(err: unknown): number | undefined {
    return err instanceof Error && 'status' in err ? (err as Error & { status: number }).status : undefined
  }

  function validate(): boolean {
    if (!currentPassword) currentError = t('password.currentRequired')
    if (newPassword.length < 8) newPasswordError = t('password.tooShort')
    if (confirmPassword !== newPassword) confirmPasswordError = t('password.mismatch')
    return !currentError && !newPasswordError && !confirmPasswordError
  }

  async function savePassword(): Promise<void> {
    currentError = undefined
    newPasswordError = undefined
    confirmPasswordError = undefined
    if (!validate()) return
    savingPassword = true
    try {
      await authApi.changePassword({
        current_password: currentPassword,
        new_password: newPassword,
      })
      toast.success(t('password.changed'))
      currentPassword = ''
      newPassword = ''
      confirmPassword = ''
    } catch (err: unknown) {
      const status = errorStatus(err)
      const message = err instanceof Error ? err.message : t('password.changeFailed')
      if (status === 401) {
        currentError = t('password.currentIncorrect')
      } else if (status === 400) {
        newPasswordError = message
      } else {
        toast.error(message)
      }
    } finally {
      savingPassword = false
    }
  }
</script>

<div class="p-6">
  <h1 class="mb-6 text-2xl font-bold">{t('nav.settings')}</h1>

  <SettingsTabs class="mb-6" />

  <Card class="max-w-lg p-6">
    <h2 class="mb-4 font-semibold">{t('password.change')}</h2>
    <div class="space-y-4">
      <Field label={t('password.current')} error={currentError}>
        <Input
          type="password"
          bind:value={currentPassword}
          error={currentError}
          autocomplete="current-password"
          oninput={() => { currentError = undefined }}
        />
      </Field>
      <Field label={t('password.new')} error={newPasswordError} hint={t('password.minLengthHint')}>
        <Input
          type="password"
          bind:value={newPassword}
          error={newPasswordError}
          autocomplete="new-password"
          oninput={() => { newPasswordError = undefined }}
        />
      </Field>
      <Field label={t('password.confirm')} error={confirmPasswordError}>
        <Input
          type="password"
          bind:value={confirmPassword}
          error={confirmPasswordError}
          autocomplete="new-password"
          oninput={() => { confirmPasswordError = undefined }}
        />
      </Field>
      <Button
        onclick={savePassword}
        disabled={savingPassword || !currentPassword || !newPassword || !confirmPassword}
      >
        {savingPassword ? t('common.saving') : t('password.change')}
      </Button>
    </div>
  </Card>
</div>
