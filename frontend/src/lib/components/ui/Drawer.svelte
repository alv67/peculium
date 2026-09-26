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
   * Right-side inspection drawer (EPIC K.1c, decision D4): the ≥ `lg` half of
   * the drawer/sheet pair — drill-downs (§8.3) keep the parent page mounted,
   * so this overlays instead of navigating. The consumer decides *when* to
   * render this vs `Sheet` (below `lg`); the component itself is
   * breakpoint-agnostic and full width-safe by default.
   *
   * Follows the Modal/MobileDrawer overlay recipe (no portal library):
   * dimmed backdrop (click closes), `role="dialog"` + `aria-modal` labelled
   * by the title (falls back to the localized details aria-label when
   * untitled), the
   * shared `focusTrap` action for Tab cycling + focus restore, and the
   * `duration-slow` (320 ms) motion token for entrance/exit. Body scroll is
   * locked while open. Lives on the drawer tier (z-30) of the z-index scale
   * (./utils.ts), below modals and toasts.
   *
   * Controlled: `open` is a plain prop and every dismissal path (Esc,
   * backdrop, ✕) calls `onClose` — the caller owns the state (and the
   * drill-down context that titles the panel).
   */
  let {
    open,
    onClose,
    title = undefined,
    width = '32rem',
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
    /** Panel width (CSS size); the panel never exceeds the viewport. */
    width?: string
    class?: string
    /** Accessible name of the ✕ button; defaults through the dictionary and
     *  callers may override (e.g. with `t('common.close')`) — mirrors
     *  `Sheet.svelte`, the < lg counterpart sharing this API). */
    closeLabel?: string
    /** Snippet between title and ✕ (e.g. "open full page" link, D4). */
    headerActions?: Snippet
    footer?: Snippet
    /** Drawer body, scrolled independently from the header/footer. */
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

  // Scroll lock keyed on `open`; the cleanup form also runs when the drawer
  // unmounts while open (page navigation, logout), which an else-branch misses.
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
      style:width
      class={cx(
        'absolute inset-y-0 right-0 flex max-w-full flex-col bg-surface-raised shadow-raised outline-none',
        className,
      )}
      transition:edgeSlide={{ edge: 'right' }}
    >
      <div class="flex items-center justify-between gap-2 border-b border-border px-5 py-4">
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

      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">{@render children?.()}</div>

      {#if footer}
        <div class="border-t border-border px-5 py-4">{@render footer()}</div>
      {/if}
    </div>
  </div>
{/if}
