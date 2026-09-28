<script lang="ts">
  import { Ban, Check, KeyRound, UserCheck } from 'lucide-svelte'
  import { toast } from '$lib/stores/toast.svelte'
  import { auth } from '$lib/stores/auth.svelte'
  import { adminApi, type AdminUser } from '$lib/services/api'
  import { t, type MessageKey } from '$lib/i18n/index.svelte'
  import AdminGate from '$lib/components/domain/AdminGate.svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte'
  import EmptyState from '$lib/components/ui/EmptyState.svelte'
  import Field from '$lib/components/ui/Field.svelte'
  import Input from '$lib/components/ui/Input.svelte'
  import Modal from '$lib/components/ui/Modal.svelte'
  import Select from '$lib/components/ui/Select.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'

  /**
   * Admin → Users (issue #57 Phase A): every account with its role and
   * status, and the admin actions on each row — approve a pending signup,
   * disable / re-enable, change role, set a new password. Whole rows are
   * updated from the PATCH response (the backend returns the fresh record),
   * so no list refetch is needed after an action; when the edited row is the
   * signed-in admin's own, the auth store is synced too so the shell's admin
   * gating reacts immediately (a self-demote reveals the `AdminGate`
   * forbidden state — the server already enforced it, the UI just follows).
   *
   * The assignable roles are `viewer`/`editor`/`admin`; the legacy `owner`
   * value renders as a badge and stays selectable on the row that still
   * carries it (so a change away is possible without silently renaming it),
   * but is never offered for assignment elsewhere.
   *
   * Backend status codes map to localized toasts (409 last-active-admin, 404
   * unknown user); raw server strings never reach the UI.
   */

  // Role values an admin can assign; `owner` is legacy (never newly set).
  const ASSIGNABLE_ROLES = ['admin', 'editor', 'viewer'] as const

  const ROLE_KEYS: Record<string, MessageKey> = {
    owner: 'admin.roleOwner',
    admin: 'admin.roleAdmin',
    editor: 'admin.roleEditor',
    viewer: 'admin.roleViewer',
  }

  const STATUS_KEYS: Record<string, MessageKey> = {
    active: 'admin.statusActive',
    pending: 'admin.statusPending',
    disabled: 'admin.statusDisabled',
  }

  function roleLabel(role: string): string {
    return ROLE_KEYS[role] ? t(ROLE_KEYS[role]) : role
  }

  function statusLabel(status: string): string {
    return STATUS_KEYS[status] ? t(STATUS_KEYS[status]) : status
  }

  function roleVariant(role: string): 'accent' | 'neutral' {
    return role === 'owner' || role === 'admin' ? 'accent' : 'neutral'
  }

  function statusVariant(status: string): 'positive' | 'warning' | 'negative' {
    if (status === 'active') return 'positive'
    if (status === 'pending') return 'warning'
    return 'negative'
  }

  let users = $state<AdminUser[]>([])
  let loading = $state(true)
  let loadError = $state(false)
  /** Id of the row with an in-flight action: its controls lock meanwhile. */
  let busyId = $state<string | null>(null)

  // Disable confirmation dialog (single instance, targeted per row).
  let disableTarget = $state<AdminUser | null>(null)
  let disableOpen = $state(false)

  // Reset-password dialog.
  let resetTarget = $state<AdminUser | null>(null)
  let resetOpen = $state(false)
  let newPassword = $state('')
  let newPasswordError = $state<string | undefined>(undefined)
  let resetting = $state(false)

  async function fetchUsers(): Promise<void> {
    loading = true
    loadError = false
    try {
      const data = await adminApi.listUsers()
      // Defensive (issue #121 precedent): never store a non-array.
      users = Array.isArray(data) ? data : []
    } catch {
      loadError = true
      users = []
      toast.error(t('admin.loadFailed'))
    } finally {
      loading = false
    }
  }

  $effect(() => {
    void fetchUsers()
  })

  /** Map the admin PATCH's failures onto localized messages. */
  function adminError(err: unknown, fallbackKey: MessageKey): string {
    const status =
      err instanceof Error && 'status' in err
        ? (err as Error & { status: number }).status
        : undefined
    if (status === 409) return t('admin.lastAdmin')
    if (status === 404) return t('admin.userNotFound')
    return t(fallbackKey)
  }

  /** Splice the fresh record back into the list (and into the open dialog
   * targets, which hold row references) and sync the shell when it's me. */
  function applyUpdated(updated: AdminUser): void {
    const index = users.findIndex((u) => u.id === updated.id)
    if (index >= 0) users[index] = updated
    if (disableTarget && disableTarget.id === updated.id) disableTarget = updated
    if (resetTarget && resetTarget.id === updated.id) resetTarget = updated
    if (auth.user && auth.user.id === updated.id) {
      auth.user = { ...auth.user, ...updated }
    }
  }

  async function updateStatus(user: AdminUser, status: string, successKey: MessageKey): Promise<void> {
    busyId = user.id
    try {
      const updated = await adminApi.updateUser(user.id, { status })
      applyUpdated(updated)
      toast.success(t(successKey))
    } catch (err) {
      toast.error(adminError(err, 'admin.updateFailed'))
    } finally {
      busyId = null
    }
  }

  async function approve(user: AdminUser): Promise<void> {
    await updateStatus(user, 'active', 'admin.approved')
  }

  async function enable(user: AdminUser): Promise<void> {
    await updateStatus(user, 'active', 'admin.userEnabled')
  }

  async function confirmDisable(): Promise<void> {
    if (!disableTarget) return
    await updateStatus(disableTarget, 'disabled', 'admin.userDisabled')
    disableTarget = null
  }

  async function changeRole(user: AdminUser, next: string): Promise<void> {
    const prev = user.role
    if (!next || next === prev) return
    busyId = user.id
    // Optimistic: the `{#key}` wrapper around the select rebuilds it from
    // this value, so a failure can roll the visible selection back cleanly.
    user.role = next
    try {
      const updated = await adminApi.updateUser(user.id, { role: next })
      applyUpdated(updated)
      toast.success(t('admin.roleUpdated'))
    } catch (err) {
      user.role = prev
      toast.error(adminError(err, 'admin.updateFailed'))
    } finally {
      busyId = null
    }
  }

  function openReset(user: AdminUser): void {
    resetTarget = user
    newPassword = ''
    newPasswordError = undefined
    resetOpen = true
  }

  async function submitReset(): Promise<void> {
    if (!resetTarget) return
    if (newPassword.length < 8) {
      newPasswordError = t('password.tooShort')
      return
    }
    resetting = true
    try {
      await adminApi.resetPassword(resetTarget.id, newPassword)
      toast.success(t('admin.passwordReset'))
      resetOpen = false
      resetTarget = null
    } catch (err) {
      toast.error(adminError(err, 'admin.resetFailed'))
    } finally {
      resetting = false
    }
  }

  function formatDate(iso: string): string {
    const date = new Date(iso)
    return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString()
  }
</script>

{#snippet retryAction()}
  <Button variant="secondary" size="sm" onclick={fetchUsers}>{t('common.retry')}</Button>
{/snippet}

<AdminGate>
  <div class="mx-auto max-w-6xl p-4 lg:p-6">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-foreground">{t('nav.users')}</h1>
        <p class="text-muted-foreground">{t('admin.usersSubtitle')}</p>
      </div>
      <Button variant="secondary" onclick={fetchUsers} disabled={loading}>
        {t('health.refresh')}
      </Button>
    </div>

    {#if loading}
      <div class="flex justify-center py-12">
        <Spinner size="lg" class="text-accent-text" />
      </div>
    {:else if loadError}
      <EmptyState
        dashed
        title={t('admin.loadFailed')}
        description={t('admin.loadFailedHint')}
        action={retryAction}
      />
    {:else if users.length === 0}
      <EmptyState dashed title={t('admin.empty')} />
    {:else}
      <div class="overflow-hidden rounded-card border border-border bg-surface shadow-card">
        <div class="overflow-x-auto">
          <Table aria-label={t('nav.users')}>
            <THead>
              <Tr>
                <Th>{t('login.email')}</Th>
                <Th>{t('chartView.colName')}</Th>
                <Th>{t('admin.colRole')}</Th>
                <Th>{t('positions.colStatus')}</Th>
                <Th>{t('admin.colCreated')}</Th>
                <Th>{t('common.colActions')}</Th>
              </Tr>
            </THead>
            <TBody>
              {#each users as user (user.id)}
                <Tr class="hover:bg-muted/50">
                  <Td class="max-w-52 truncate font-medium text-foreground">{user.email}</Td>
                  <Td class="max-w-40 truncate text-muted-foreground">{user.name || '—'}</Td>
                  <Td>
                    <Badge variant={roleVariant(user.role)}>{roleLabel(user.role)}</Badge>
                  </Td>
                  <Td><Badge variant={statusVariant(user.status)}>{statusLabel(user.status)}</Badge></Td>
                  <Td class="whitespace-nowrap text-muted-foreground">{formatDate(user.created_at)}</Td>
                  <Td>
                    <div class="flex items-center gap-1">
                      <!-- Change-role select first: the current role also shows as a
                           badge, so the select is the action, not the display. -->
                      {#key user.role}
                        <Select
                          value={user.role}
                          aria-label={t('admin.roleNamed', { email: user.email })}
                          disabled={busyId === user.id}
                          class="w-36 shrink-0"
                          onchange={(event: Event) =>
                            void changeRole(user, (event.target as HTMLSelectElement).value)}
                        >
                          {#if user.role === 'owner'}
                            <option value="owner">{roleLabel('owner')}</option>
                          {/if}
                          {#each ASSIGNABLE_ROLES as role (role)}
                            <option value={role}>{roleLabel(role)}</option>
                          {/each}
                        </Select>
                      {/key}
                      {#if user.status === 'pending'}
                        <Button
                          variant="ghost"
                          size="icon"
                          class="text-positive"
                          disabled={busyId === user.id}
                          aria-label={t('admin.approveNamed', { email: user.email })}
                          title={t('admin.approve')}
                          onclick={() => void approve(user)}
                        >
                          <UserCheck class="h-4 w-4" />
                        </Button>
                      {:else if user.status === 'active'}
                        <Button
                          variant="ghost"
                          size="icon"
                          class="text-negative"
                          disabled={busyId === user.id}
                          aria-label={t('admin.disableNamed', { email: user.email })}
                          title={t('admin.disable')}
                          onclick={() => {
                            disableTarget = user
                            disableOpen = true
                          }}
                        >
                          <Ban class="h-4 w-4" />
                        </Button>
                      {:else}
                        <Button
                          variant="ghost"
                          size="icon"
                          class="text-positive"
                          disabled={busyId === user.id}
                          aria-label={t('admin.enableNamed', { email: user.email })}
                          title={t('admin.enable')}
                          onclick={() => void enable(user)}
                        >
                          <Check class="h-4 w-4" />
                        </Button>
                      {/if}
                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={busyId === user.id}
                        aria-label={t('admin.resetNamed', { email: user.email })}
                        title={t('admin.resetPassword')}
                        onclick={() => openReset(user)}
                      >
                        <KeyRound class="h-4 w-4" />
                      </Button>
                    </div>
                  </Td>
                </Tr>
              {/each}
            </TBody>
          </Table>
        </div>
      </div>
      <p class="mt-3 text-xs text-muted-foreground">{t('admin.ownerNote')}</p>
    {/if}
  </div>
</AdminGate>

<!-- Disable confirmation (destructive: the account cannot sign in). -->
<ConfirmDialog
  bind:open={disableOpen}
  title={t('admin.disableTitle')}
  message={disableTarget ? t('admin.disableConfirm', { email: disableTarget.email }) : ''}
  confirmLabel={t('admin.disable')}
  variant="danger"
  loading={disableTarget !== null && busyId === disableTarget.id}
  onconfirm={confirmDisable}
/>

<!-- Admin password reset: a new password (min 8) for the target account. -->
<Modal
  bind:open={resetOpen}
  title={resetTarget ? t('admin.resetTitle', { email: resetTarget.email }) : t('admin.resetPassword')}
  description={t('admin.resetHint')}
  size="sm"
  dismissible={!resetting}
>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (resetOpen = false)} disabled={resetting}>
      {t('common.cancel')}
    </Button>
    <Button onclick={submitReset} loading={resetting}>{t('admin.resetPassword')}</Button>
  {/snippet}
  <Field label={t('password.new')} error={newPasswordError} hint={t('password.minLengthHint')}>
    <Input
      type="password"
      bind:value={newPassword}
      autocomplete="new-password"
      error={newPasswordError}
      disabled={resetting}
      oninput={() => (newPasswordError = undefined)}
    />
  </Field>
</Modal>
