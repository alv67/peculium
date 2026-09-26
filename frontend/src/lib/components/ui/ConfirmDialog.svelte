<script lang="ts">
  import Button from './Button.svelte'
  import Modal from './Modal.svelte'
  import { t } from '$lib/i18n/index.svelte'

  /**
   * Promise-friendly replacement for `window.confirm()` (EPIC D.2).
   *
   * The message is passed to `Modal`'s `description` so the dialog is
   * announced in full via `aria-describedby`. `onconfirm` may be async: the
   * dialog locks both buttons (and Esc/backdrop) and spins the confirm one
   * while the promise runs, then closes on resolve. A rejection keeps it
   * open so the user can retry — page handlers are expected to catch and
   * surface errors via `toast.error`, exactly like before.
   */
  let {
    open = $bindable(false),
    title,
    message,
    confirmLabel = t('common.confirm'),
    cancelLabel = t('common.cancel'),
    variant = 'primary',
    loading = false,
    onconfirm,
  }: {
    open?: boolean
    title: string
    message: string
    confirmLabel?: string
    cancelLabel?: string
    /** Confirm button styling; `danger` for destructive actions. */
    variant?: 'danger' | 'primary'
    /** Caller-managed busy flag, merged with the internal one. */
    loading?: boolean
    onconfirm?: () => void | Promise<void>
  } = $props()

  let busy = $state(false)
  const pending = $derived(loading || busy)

  function cancel(): void {
    if (pending) return
    open = false
  }

  async function confirm(): Promise<void> {
    if (pending) return
    busy = true
    try {
      await onconfirm?.()
      open = false
    } finally {
      busy = false
    }
  }
</script>

{#snippet buttons()}
  <Button variant="secondary" onclick={cancel} disabled={pending}>{cancelLabel}</Button>
  <Button {variant} onclick={confirm} loading={pending}>{confirmLabel}</Button>
{/snippet}

<Modal
  bind:open
  {title}
  description={message}
  size="sm"
  dismissible={!pending}
  footer={buttons}
></Modal>
