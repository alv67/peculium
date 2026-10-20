<script lang="ts">
  import { ChevronDown, LogOut } from 'lucide-svelte'
  import { afterNavigate } from '$app/navigation'
  import { auth, logout } from '#lib/stores/auth.svelte.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import Button from '../ui/Button.svelte'
  import { cx } from '../ui/utils'

  /**
   * Account menu (EPIC D.3): initials avatar + minimal popup with the user
   * identity (name/email) and Sign out. Settings/Health navigation lives in
   * the sidebar (`SidebarNav`), not here, to avoid duplicate entry points.
   * Rendered twice by the shell — bottom of the desktop
   * sidebar (`direction="up"`, optional `compact` for the icon rail) and in
   * the header on mobile (`direction="down"`, `align="end"`, `compact`) —
   * hence the placement props instead of a hardcoded position.
   *
   * Closes on Escape / outside pointerdown / route change; Escape returns
   * focus to the trigger. Popup sits on z-20 (dropdown tier).
   */
  let {
    /** Icon-only trigger (rail / header): hides name + chevron. */
    compact = false,
    /** Popup opens above (sidebar bottom) or below (header) the trigger. */
    direction = 'up',
    /** Popup edge aligned with the trigger. */
    align = 'start',
    class: className = '',
  }: {
    compact?: boolean
    direction?: 'up' | 'down'
    align?: 'start' | 'end'
    class?: string
  } = $props()

  let open = $state(false);
  let root = $state<HTMLDivElement | null>(null);
  let trigger = $state<HTMLButtonElement | null>(null);
  const name = $derived(auth.user?.name || auth.user?.email || t('user.fallbackName'));

  // Non-compact: the visible name *is* the accessible label (label-in-name);
  // the compact trigger renders only initials, so it needs an explicit one.
  const triggerLabel = $derived(compact ? t('user.accountMenu') : undefined)
  const initials = $derived.by(() => {
    const parts = auth.user?.name?.trim().split(/\s+/).filter(Boolean);

    if (parts?.length) return (parts[0][0] + (parts[1]?.[0] ?? '')).toUpperCase();

    const email = auth.user?.email?.trim();

    return email ? email[0].toUpperCase() : '?';
  });

  function toggle(): void {
    open = !open
  }

  function handleWindowKeydown(event: KeyboardEvent): void {
    if (!open) return
    if (event.key === 'Escape') {
      open = false
      trigger?.focus()
    }
  }

  afterNavigate(({ shallow, type }) => {
    if (shallow && type === 'goto') return;

    return open = false;
  });

  $effect(() => {
    if (!open) return
    function handlePointerdown(event: PointerEvent): void {
      if (root && event.target instanceof Node && !root.contains(event.target)) {
        open = false
      }
    }
    window.addEventListener('pointerdown', handlePointerdown)
    return () => window.removeEventListener('pointerdown', handlePointerdown)
  })
</script>

<svelte:window onkeydown={handleWindowKeydown}></svelte:window>

<div class={cx('relative', className)} bind:this={root}>
  <button
    bind:this={trigger}
    type="button"
    aria-haspopup="true"
    aria-expanded={open}
    aria-label={triggerLabel}
    onclick={toggle}
    class={cx(
      'focus-ring flex items-center gap-2 rounded-control text-sm text-foreground transition-colors hover:bg-muted',
      // Rail width is 64px with a 12px footer pad → 40px usable: `p-1` keeps
      // the avatar button (32px + 8px) exactly inside it.
      compact ? 'justify-center p-1' : 'w-full px-3 py-2 text-left',
    )}
  >
    <span
      class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-accent text-xs font-semibold text-accent-foreground"
      aria-hidden="true"
    >
      {initials}
    </span>
    {#if !compact}
      <span class="flex-1 truncate font-medium">{name}</span>
      <ChevronDown
        class={cx(
          'h-4 w-4 shrink-0 text-muted-foreground transition-transform',
          open && 'rotate-180',
        )}
      />
    {/if}
  </button>

  {#if open}
    <div
      class={cx(
        'absolute z-20 w-56 overflow-hidden rounded-card border border-border bg-surface shadow-raised',
        direction === 'up' ? 'bottom-full mb-2' : 'top-full mt-2',
        align === 'end' ? 'left-auto right-0' : 'left-0',
      )}
    >
      <div class="border-b border-border px-4 py-3">
        <div class="truncate text-sm font-medium text-foreground">{auth.user?.name || t('user.fallbackName')}</div>
        <div class="truncate text-xs text-muted-foreground">{auth.user?.email}</div>
      </div>
      <div class="flex flex-col p-1">
        <Button variant="ghost" class="w-full" onclick={logout}>
          <span class="flex w-full items-center gap-2 text-negative">
            <LogOut class="h-4 w-4 shrink-0" />
            {t('user.signOut')}
          </span>
        </Button>
      </div>
    </div>
  {/if}
</div>
