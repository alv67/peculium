<script lang="ts">
  import { onMount } from 'svelte'
  import { toast } from '$lib/stores/toast.svelte'
  import { login, register } from '$lib/stores/auth.svelte'
  import { loadAppVersion } from '$lib/stores/appVersion.svelte'
  import { errorStatus } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import Field from '$lib/components/ui/Field.svelte'
  import Input from '$lib/components/ui/Input.svelte'
  import SegmentedControl from '$lib/components/ui/SegmentedControl.svelte'
  import VersionChip from '$lib/components/layout/VersionChip.svelte'

  const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

  // Reactive so switching language re-renders the mode pills (and the CTA,
  // which reuses the same labels) in place.
  const modeItems = $derived([
    { value: 'signin', label: t('login.signIn') },
    { value: 'register', label: t('login.register') },
  ])

  let mode = $state('signin')
  let email = $state('')
  let password = $state('')
  let confirmPassword = $state('')
  let name = $state('')
  let submitting = $state(false)

  let emailError = $state<string | undefined>(undefined)
  let passwordError = $state<string | undefined>(undefined)
  let confirmError = $state<string | undefined>(undefined)
  let nameError = $state<string | undefined>(undefined)

  const isRegister = $derived(mode === 'register')

  // The build chip below the card reads the shared store; `/version` is
  // public, so it works already on this pre-login page (#198).
  onMount(() => {
    void loadAppVersion()
  })

  // The two modes have different password/name constraints, so validation
  // state never carries over a mode switch.
  $effect(() => {
    if (mode) {
      emailError = undefined
      passwordError = undefined
      confirmError = undefined
      nameError = undefined
    }
  })

  function validate(): boolean {
    let valid = true
    if (!email.trim()) {
      emailError = t('login.emailRequired')
      valid = false
    } else if (!EMAIL_REGEX.test(email)) {
      emailError = t('login.emailInvalid')
      valid = false
    }
    // Length is only enforced on register: legacy accounts may have shorter passwords.
    if (!password) {
      passwordError = t('login.passwordRequired')
      valid = false
    } else if (isRegister && password.length < 8) {
      passwordError = t('login.passwordTooShort')
      valid = false
    }
    if (isRegister && (!confirmPassword || confirmPassword !== password)) {
      confirmError = t('login.passwordMismatch')
      valid = false
    }
    if (isRegister && !name.trim()) {
      nameError = t('login.nameRequired')
      valid = false
    }
    return valid
  }

  async function handleSubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault()
    if (!validate()) return
    submitting = true
    try {
      if (isRegister) {
        const user = await register(email, name, password)
        // Approval flow (#57 Phase A): with auto-approve off the account is
        // created as `pending` and sign-in would be refused — say so instead
        // of the generic "you can now log in".
        toast.success(user.status === 'pending' ? t('login.registeredPending') : t('login.registered'))
        mode = 'signin'
      } else {
        await login(email, password)
      }
    } catch (err: unknown) {
      const status = errorStatus(err)
      const raw = err instanceof Error ? err.message : ''
      // The backend answers sign-ins on non-active accounts with 403 +
      // "account pending approval" / "account disabled" (#57): surface those
      // as clear localized messages; everything else keeps the previous
      // behavior (backend message, generic fallback).
      let message: string
      if (status === 403 && raw.includes('disabled')) {
        message = t('login.accountDisabled')
      } else if (status === 403 && raw.includes('pending')) {
        message = t('login.accountPending')
      } else {
        message = raw || t('common.somethingWentWrong')
      }
      toast.error(message)
    } finally {
      submitting = false
    }
  }
</script>

<div class="grid min-h-dvh place-items-center bg-background p-4">
  <div class="w-full max-w-sm rounded-card border border-border bg-surface p-6 shadow-raised sm:p-8">
    <div class="mb-6 flex flex-col items-center gap-3">
      <img src="/peculium.svg" alt="" class="h-12 w-12" />
      <h1 class="text-2xl font-bold text-foreground">Peculium</h1>
      <p class="text-center text-sm text-muted-foreground">{t('login.tagline')}</p>
    </div>
    <p class="mb-6 text-center text-sm text-muted-foreground">
      {isRegister ? t('login.createAccount') : t('login.signInToAccount')}
    </p>
    <SegmentedControl items={modeItems} bind:value={mode} ariaLabel={t('login.authMode')} class="mb-6 w-full" />
    <!-- `novalidate` keeps the submit on our inline validation; `required` stays for a11y. -->
    <form onsubmit={handleSubmit} class="space-y-4" novalidate>
      {#if isRegister}
        <Field label={t('login.name')} error={nameError}>
          <Input
            bind:value={name}
            type="text"
            autocomplete="name"
            required
            error={nameError}
            oninput={() => (nameError = undefined)}
          />
        </Field>
      {/if}
      <Field label={t('login.email')} error={emailError}>
        <Input
          bind:value={email}
          type="email"
          autocomplete="email"
          required
          error={emailError}
          oninput={() => (emailError = undefined)}
        />
      </Field>
      <Field
        label={t('login.password')}
        error={passwordError}
        hint={isRegister ? t('login.passwordHint') : undefined}
      >
        <Input
          bind:value={password}
          type="password"
          autocomplete={isRegister ? 'new-password' : 'current-password'}
          required
          error={passwordError}
          oninput={() => (passwordError = undefined)}
        />
      </Field>
      {#if isRegister}
        <Field label={t('login.confirmPassword')} error={confirmError}>
          <Input
            bind:value={confirmPassword}
            type="password"
            autocomplete="new-password"
            required
            error={confirmError}
            oninput={() => (confirmError = undefined)}
          />
        </Field>
      {/if}
      <Button type="submit" class="w-full" loading={submitting}>
        {isRegister ? t('login.register') : t('login.signIn')}
      </Button>
    </form>
  </div>

  <div class="mt-4 w-full max-w-sm text-center">
    <VersionChip />
  </div>
</div>
