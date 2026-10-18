<script lang="ts">
  import { Banknote, Briefcase, Ellipsis, LayoutDashboard } from 'lucide-svelte'
  import { page } from '$app/state'
  import { resolve } from '$app/paths'
  import { t } from '$lib/i18n/index.svelte'
  import { cx } from '../ui/utils'

  /**
   * Phone bottom navigation (EPIC K.2, decision D2): a fixed bar with four
   * destinations — Overview, Portfolios, Assets and More. The hamburger is
   * gone: the former MobileDrawer survives as the "More" sheet and this bar
   * is its only trigger, while quick actions live in the adjacent Fab.
   *
   * The active item follows the longest-prefix rule established by
   * `SidebarNav` (with `/` matching exactly), so `/portfolios/3` highlights
   * Portfolios alone. "More" lights up for the destinations it hosts (Data &
   * Sync, Settings — the same config `SidebarNav` renders): those pages have
   * no slot of their own at this size. `aria-current="page"` goes on real
   * links only; the More control opens a dialog, so it carries
   * `aria-haspopup`/`aria-expanded` instead.
   *
   * 56px-tall flex-1 columns keep every tap target ≥ 44px; the safe-area
   * bottom padding lifts the bar above home indicators, and `sm:hidden`
   * mirrors the AppShell's `viewport.isPhone` mount guard.
   */
  type IconType = typeof LayoutDashboard

  let {
    moreOpen,
    onopenmore,
  }: {
    /** More-sheet state, surfaced on the trigger via `aria-expanded`. */
    moreOpen: boolean
    /** Opens the More sheet (the repurposed MobileDrawer). */
    onopenmore: () => void
  } = $props()

  const items = [
    { to: '/', labelKey: 'nav.overview', icon: LayoutDashboard as IconType },
    { to: '/portfolios', labelKey: 'nav.portfolios', icon: Briefcase as IconType },
    { to: '/assets', labelKey: 'nav.assets', icon: Banknote as IconType },
  ] as const

  // "More" lights up for every route the sheet hosts: the Settings area and
  // the whole admin area (both live under distinct prefixes).
  const moreActive = $derived(
    page.url.pathname.startsWith('/admin') || page.url.pathname.startsWith('/settings'),
  )

  function matches(to: string, pathname: string): boolean {
    if (to === '/') return pathname === '/'
    return pathname === to || pathname.startsWith(`${to}/`)
  }

  const activeTo = $derived.by(() => {
    let best: string | null = null
    for (const item of items) {
      if (
        matches(item.to, page.url.pathname) &&
        (best === null || item.to.length > best.length)
      ) {
        best = item.to
      }
    }
    return best
  })

  function itemClasses(active: boolean): string {
    return cx(
      'focus-ring flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 px-1 text-micro transition-colors',
      active ? 'font-medium text-accent-text' : 'text-muted-foreground',
    )
  }
</script>

<nav
  aria-label={t('nav.bottomNav')}
  class="fixed inset-x-0 bottom-0 z-20 flex border-t border-border bg-surface/95 pb-[env(safe-area-inset-bottom)] backdrop-blur sm:hidden"
>
  {#each items as item (item.to)}
    {@const Icon = item.icon}
    <a
      href={resolve(item.to)}
      class={itemClasses(item.to === activeTo)}
      aria-current={item.to === activeTo ? 'page' : undefined}
    >
      <Icon class="h-5 w-5 shrink-0" />
      {t(item.labelKey)}
    </a>
  {/each}

  <button
    type="button"
    class={itemClasses(moreActive)}
    aria-label={t('nav.more')}
    aria-haspopup="dialog"
    aria-expanded={moreOpen}
    onclick={onopenmore}
  >
    <Ellipsis class="h-5 w-5 shrink-0" />
    {t('nav.more')}
  </button>
</nav>
