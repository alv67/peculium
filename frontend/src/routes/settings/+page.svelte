<script lang="ts">
  import { onMount } from 'svelte'
  import { toast } from '#lib/stores/toast.svelte.js'
  import { auth, updateProfile } from '#lib/stores/auth.svelte.js'
  import { settingsApi, type Currency } from '#lib/services/api.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import SettingsTabs from '#lib/components/domain/SettingsTabs.svelte'
  import CurrencySelect from '#lib/components/domain/CurrencySelect.svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import Card from '#lib/components/ui/Card.svelte'
  import Field from '#lib/components/ui/Field.svelte'
  import Input from '#lib/components/ui/Input.svelte'

  let name = $state(auth.user?.name ?? '')
  let email = $state(auth.user?.email ?? '')
  // Base currency for consolidated views (EPIC I.1); defaults to EUR for
  // users whose record predates the field.
  let baseCurrency = $state(auth.user?.base_currency ?? 'EUR')
  let currencies = $state<Currency[]>([])
  let savingProfile = $state(false)

  onMount(async () => {
    try {
      const res = await settingsApi.listCurrencies()
      currencies = res.currencies
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('currencies.loadFailed')
      toast.error(message)
    }
  })

  async function saveProfile(): Promise<void> {
    savingProfile = true
    try {
      await updateProfile(name, email, baseCurrency)
      toast.success(t('profile.updated'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('profile.updateFailed')
      toast.error(message)
    } finally {
      savingProfile = false
    }
  }
</script>

<div class="p-6">
  <h1 class="mb-6 text-2xl font-bold">{t('nav.settings')}</h1>

  <SettingsTabs class="mb-6" />

  <Card class="max-w-lg p-6">
    <h2 class="mb-4 font-semibold">{t('settingsTabs.profile')}</h2>
    <div class="space-y-4">
      <Field label={t('chartView.colName')}>
        <Input type="text" bind:value={name} autocomplete="name" />
      </Field>
      <Field label={t('profile.email')}>
        <Input type="email" bind:value={email} autocomplete="email" />
      </Field>
      <Field
        label={t('profile.baseCurrency')}
        hint={t('profile.baseCurrencyHint')}
      >
        <CurrencySelect bind:value={baseCurrency} {currencies} ariaLabel={t('profile.baseCurrency')} />
      </Field>
      <Button onclick={saveProfile} disabled={savingProfile || !name || !email}>
        {savingProfile ? t('common.saving') : t('common.save')}
      </Button>
    </div>
  </Card>
</div>
