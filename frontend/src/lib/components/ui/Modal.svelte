<script lang="ts">
  import type { Snippet } from 'svelte'
  import { X } from 'lucide-svelte'
  import { t } from '$lib/i18n/index.svelte'
  import Button from './Button.svelte'
  import { cx, OVERLAY_OPEN_GRACE_MS } from './utils'

  /**
   * Accessible modal dialog (EPIC D.2).
   *
   * Follows the `fixed inset-0` overlay pattern already used by the inline
   * exposure modals (no portal library): the overlay carries the
   * `role="dialog" aria-modal` semantics and the panel sits centered on top.
   *
   * Behavior:
   * - Esc and backdrop clicks close it while `dismissible` (default true);
   *   the header ✕ button is the explicit close affordance.
   * - Body scroll is locked while open.
   * - Basic focus management: the overlay receives focus on open (so screen
   *   readers announce `aria-labelledby`), Tab is trapped inside the panel,
   *   and focus is restored to the trigger on close.
   * - `open` is `$bindable`: `bind:open={state}` on the caller drives the
   *   dialog, and any internal close writes `false` back through the binding.
   *
   * Part of the z-index scale in ./utils.ts: modals live at z-40 (toasts 50).
   */
  let {
    open = $bindable(false),
    title,
    description = undefined,
    size = 'md',
    dismissible = true,
    footer = undefined,
    children = undefined,
    onclose = undefined,
    class: className = '',
  }: {
    open?: boolean
    title: string
    description?: string
    size?: 'sm' | 'md' | 'lg' | 'xl'
    /** When false, Esc/backdrop/✕ are all disabled (e.g. busy states). */
    dismissible?: boolean
    footer?: Snippet
    children?: Snippet
    /** Called on every user-initiated close (Esc, backdrop, ✕). */
    onclose?: () => void
    /** Extra classes on the panel (width overrides etc.). */
    class?: string
  } = $props()

  const sizes = {
    sm: 'max-w-sm',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl',
  } as const

  const uid = $props.id()
  const titleId = `${uid}-title`
  const descId = `${uid}-description`

  let overlay = $state<HTMLDivElement | null>(null)
  let panel = $state<HTMLDivElement | null>(null)
  let lastFocused: HTMLElement | null = null

  // Timestamp of the last open, used to swallow the synthetic click a touch
  // tap fires right after opening (click-through — see utils).
  let openedAt = 0

  const FOCUSABLE =
    'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

  function close(): void {
    if (!dismissible) return
    onclose?.()
    open = false
  }

  function handleBackdropClick(event: MouseEvent): void {
    // Only clicks that land on the overlay itself (not bubbled from the panel),
    // and not the ghost click that immediately follows the opening tap.
    if (event.target !== event.currentTarget) return
    if (performance.now() - openedAt < OVERLAY_OPEN_GRACE_MS) return
    close()
  }

  function handleWindowKeydown(event: KeyboardEvent): void {
    if (open && event.key === 'Escape') close()
  }

  function handleTabKeydown(event: KeyboardEvent): void {
    if (event.key !== 'Tab' || !panel) return
    const focusables = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE))
    if (focusables.length === 0) {
      event.preventDefault()
      return
    }
    const first = focusables[0]
    const last = focusables[focusables.length - 1]
    const active = document.activeElement
    if (event.shiftKey && (active === first || active === overlay)) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && active === last) {
      event.preventDefault()
      first.focus()
    }
  }

  // Scroll-lock + focus capture/restore, keyed on `open`. The cleanup form
  // also runs when the dialog is destroyed while open (page navigation),
  // which a plain `else` branch would miss.
  $effect(() => {
    if (!open) return
    openedAt = performance.now()
    lastFocused =
      document.activeElement instanceof HTMLElement ? document.activeElement : null
    document.body.style.overflow = 'hidden'
    overlay?.focus()
    return () => {
      document.body.style.overflow = ''
      lastFocused?.focus()
      lastFocused = null
    }
  })
</script>

<svelte:window onkeydown={handleWindowKeydown} />

{#if open}
  <div
    bind:this={overlay}
    class="fixed inset-0 z-40 flex items-center justify-center bg-overlay/50 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby={titleId}
    aria-describedby={description ? descId : undefined}
    tabindex="-1"
    onclick={handleBackdropClick}
    onkeydown={handleTabKeydown}
  >
    <div
      bind:this={panel}
      class={cx(
        'relative max-h-[90vh] w-full overflow-y-auto rounded-card bg-surface-raised p-6 shadow-raised',
        sizes[size],
        className,
      )}
    >
      <div class="mb-4 flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h2 id={titleId} class="text-lg font-semibold text-foreground">{title}</h2>
          {#if description}
            <p id={descId} class="mt-1 text-sm text-muted-foreground">{description}</p>
          {/if}
        </div>
        {#if dismissible}
          <Button variant="ghost" size="icon" onclick={close} aria-label={t('a11y.closeDialog')}>
            <X class="h-5 w-5" />
          </Button>
        {/if}
      </div>

      {@render children?.()}

      {#if footer}
        <div class="mt-6 flex flex-wrap items-center justify-end gap-2">{@render footer()}</div>
      {/if}
    </div>
  </div>
{/if}
