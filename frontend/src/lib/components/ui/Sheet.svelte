<script lang="ts">
  import type { Snippet } from 'svelte'
  import { fade } from 'svelte/transition'
  import { X } from 'lucide-svelte'
  import { t } from '$lib/i18n/index.svelte'
  import Button from './Button.svelte'
  import { focusTrap } from './focus-trap'
  import { backdropFade, edgeSlide } from './transitions'
  import { cx, OVERLAY_OPEN_GRACE_MS } from './utils'

  /**
   * Bottom sheet (EPIC K.1c, decision D4): the < `lg` counterpart of
   * `Drawer.svelte` — same API and behavior, anchored to the bottom edge,
   * full width, top-rounded (20 px per the shape rule, §7.2) with a
   * drag-handle affordance on top.
   *
   * The handle is currently decorative (grab affordance only): real
   * drag-to-dismiss is a §8.7 mobile gesture and lands with the K.2/K.4
   * adoption, where it must keep a visible button equivalent (the ✕).
   *
   * Dismissal paths, focus trap/restore, Esc/backdrop handling, scroll lock,
   * the z-30 drawer tier and the 320 ms motion token are all shared with
   * `Drawer` via ./focus-trap and ./transitions. Controlled: every close
   * calls `onClose`; the caller owns `open`.
   */
  let {
    open,
    onClose,
    title = undefined,
    class: className = '',
    closeLabel = t('a11y.closePanel'),
    headerActions = undefined,
    footer = undefined,
    children = undefined,
  }: {
    open: boolean
    /** Called on Esc, backdrop click and the ✕ button. */
    onClose: () => void
    title?: string
    class?: string
    /** Accessible name of the ✕ button; defaults through the dictionary and
     *  callers may override (e.g. with `t('common.close')`). */
    closeLabel?: string
    /** Snippet between title and ✕ (e.g. a secondary action). */
    headerActions?: Snippet
    footer?: Snippet
    /** Sheet body, scrolled independently from the header/footer. */
    children?: Snippet
  } = $props()

  const uid = $props.id()
  const titleId = `${uid}-title`

  // Timestamp of the last open, used to swallow the synthetic click a touch
  // tap fires right after opening (click-through — see utils).
  let openedAt = 0

  function handleBackdropClick(event: MouseEvent): void {
    // Only clicks that land on the overlay itself (not bubbled from the panel),
    // and not the ghost click that immediately follows the opening tap.
    if (event.target !== event.currentTarget) return
    if (performance.now() - openedAt < OVERLAY_OPEN_GRACE_MS) return
    onClose()
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') onClose()
  }

  // Scroll lock keyed on `open`; cleanup also runs when the sheet unmounts
  // while open (see Drawer for the rationale).
  $effect(() => {
    if (!open) return
    openedAt = performance.now()
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = ''
    }
  })
</script>

{#if open}
  <div
    class="fixed inset-0 z-30 bg-overlay/50"
    role="dialog"
    aria-modal="true"
    aria-labelledby={title ? titleId : undefined}
    aria-label={title ? undefined : t('a11y.details')}
    tabindex="-1"
    transition:fade={backdropFade()}
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
  >
    <div
      use:focusTrap
      tabindex="-1"
      class={cx(
        'absolute inset-x-0 bottom-0 flex max-h-[85dvh] flex-col rounded-t-[20px] bg-surface-raised shadow-raised outline-none',
        className,
      )}
      transition:edgeSlide={{ edge: 'bottom' }}
    >
      <!-- Decorative drag-handle affordance (see docstring). -->
      <div class="flex justify-center pb-1 pt-2.5" aria-hidden="true">
        <span class="h-1 w-10 rounded-full bg-muted-foreground/40"></span>
      </div>

      <div class="flex items-center justify-between gap-2 border-b border-border px-4 py-3">
        {#if title}
          <h2 id={titleId} class="min-w-0 truncate text-base font-semibold text-foreground">
            {title}
          </h2>
        {/if}
        <div class="flex shrink-0 items-center gap-1">
          {@render headerActions?.()}
          <Button variant="ghost" size="icon" onclick={onClose} aria-label={closeLabel}>
            <X class="h-5 w-5" />
          </Button>
        </div>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto px-4 pt-4 pb-[calc(1rem+env(safe-area-inset-bottom))]">
        {@render children?.()}
      </div>

      {#if footer}
        <div class="border-t border-border px-4 py-4">{@render footer()}</div>
      {/if}
    </div>
  </div>
{/if}
