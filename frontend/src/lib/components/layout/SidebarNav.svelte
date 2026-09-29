<script lang="ts">
  import {
    Activity,
    Banknote,
    Briefcase,
    LayoutDashboard,
    ServerCog,
    Settings,
    Users,
  } from 'lucide-svelte'
  import { page } from '$app/state'
  import { resolve } from '$app/paths'
  import { t } from '$lib/i18n/index.svelte'
  import { isAdmin } from '$lib/stores/auth.svelte'
  import { cx } from '../ui/utils'

  /**
   * Navigation inside the sidebar (EPIC D.3, adaptive in K.2): main section
   * on top, Admin and Settings sections pinned to the bottom. `collapsed`
   * renders the icon-rail variant (64px): labels disappear, so each link
   * carries an `aria-label`/`title`. The shell uses the three surfaces of
   * this one component: desktop expandable sidebar, tablet forced rail and
   * the phone More sheet (drawer) — below `lg` every item grows to a 44px
   * minimum touch target.
   *
   * Labels are translated with `t()` (EPIC K.1b, decision D1): the items
   * carry `labelKey`s and every render (text + collapsed aria/title) reads
   * the reactive locale, so switching language re-renders the nav in place.
   *
   * The Admin section's management entries (Users, Server settings) render
   * only for admin-equivalent roles (#57 Phase A); `isAdmin()` reads the
   * session, so the derived admin list re-computes on any role change.
   *
   * The active entry is the item whose path is the *longest* prefix of the
   * current URL (with `/` matching exactly): only one link ever gets
   * `aria-current="page"`, so `/admin/health` highlights Data & Sync alone
   * and `/settings` alone highlights Settings.
   */
  type IconType = typeof LayoutDashboard

  let { collapsed = false }: { collapsed?: boolean } = $props()

  // `as const` keeps `to` as literal route types so the typed `resolve()`
  // accepts them (and the label keys against the `MessageKey` union); the
  // icon cast unifies the component type per list.
  const mainItems = [
    { to: '/', labelKey: 'nav.dashboard', icon: LayoutDashboard as IconType },
    { to: '/portfolios', labelKey: 'nav.portfolios', icon: Briefcase as IconType },
    { to: '/assets', labelKey: 'nav.assets', icon: Banknote as IconType },
  ] as const

  // Decision D7: this is the *single* config point for the "Data & Sync"
  // (ex "Health") entry — route, label key and icon live only here. The
  // admin management entries (#57 Phase A) join the same "Admin" section but
  // render only for `owner`/`admin` accounts; Data & Sync stays open to every
  // user (its `/health/prices` endpoint is not admin-gated).
  const adminOnlyItems = [
    { to: '/admin/users', labelKey: 'nav.users', icon: Users as IconType },
    { to: '/admin/settings', labelKey: 'nav.serverSettings', icon: ServerCog as IconType },
  ] as const

  const healthItems = [
    { to: '/admin/health', labelKey: 'nav.dataSync', icon: Activity as IconType },
  ] as const

  const adminItems = $derived(isAdmin() ? [...adminOnlyItems, ...healthItems] : [...healthItems])

  const settingsItems = [
    { to: '/settings', labelKey: 'nav.settings', icon: Settings as IconType },
  ] as const

  function matches(to: string, pathname: string): boolean {
    if (to === '/') return pathname === '/'
    return pathname === to || pathname.startsWith(`${to}/`)
  }

  const activeTo = $derived.by(() => {
    let best: string | null = null
    for (const item of [...mainItems, ...adminItems, ...settingsItems]) {
      if (
        matches(item.to, page.url.pathname) &&
        (best === null || item.to.length > best.length)
      ) {
        best = item.to
      }
    }
    return best
  })

  function itemClasses(to: string): string {
    return cx(
      // `max-lg:min-h-11`: 44px tap targets on touch devices (tablet rail,
      // phone More sheet); the desktop sidebar keeps its compact density.
      'focus-ring flex items-center gap-3 rounded-control py-2 text-sm transition-colors max-lg:min-h-11',
      collapsed ? 'justify-center px-0' : 'px-3',
      to === activeTo
        ? 'bg-accent/10 font-medium text-accent-text'
        : 'text-muted-foreground hover:bg-muted hover:text-foreground',
    )
  }
</script>

<nav aria-label={t('nav.main')} class="flex min-h-0 flex-1 flex-col">
  <div class="flex-1 space-y-1 overflow-y-auto p-3">
    {#each mainItems as item (item.to)}
      {@const Icon = item.icon}
      <a
        href={resolve(item.to)}
        class={itemClasses(item.to)}
        aria-current={item.to === activeTo ? 'page' : undefined}
        aria-label={collapsed ? t(item.labelKey) : undefined}
        title={collapsed ? t(item.labelKey) : undefined}
      >
        <Icon class="h-5 w-5 shrink-0" />
        {#if !collapsed}{t(item.labelKey)}{/if}
      </a>
    {/each}
  </div>

  <div class="border-t border-border p-3">
    {#if !collapsed}
      <div class="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {t('nav.sectionAdmin')}
      </div>
    {/if}
    <div class="space-y-1">
      {#each adminItems as item (item.to)}
        {@const Icon = item.icon}
        <a
          href={resolve(item.to)}
          class={itemClasses(item.to)}
          aria-current={item.to === activeTo ? 'page' : undefined}
          aria-label={collapsed ? t(item.labelKey) : undefined}
          title={collapsed ? t(item.labelKey) : undefined}
        >
          <Icon class="h-5 w-5 shrink-0" />
          {#if !collapsed}{t(item.labelKey)}{/if}
        </a>
      {/each}
    </div>
  </div>

  <div class="border-t border-border p-3">
    {#if !collapsed}
      <div class="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {t('nav.sectionSettings')}
      </div>
    {/if}
    <div class="space-y-1">
      {#each settingsItems as item (item.to)}
        {@const Icon = item.icon}
        <a
          href={resolve(item.to)}
          class={itemClasses(item.to)}
          aria-current={item.to === activeTo ? 'page' : undefined}
          aria-label={collapsed ? t(item.labelKey) : undefined}
          title={collapsed ? t(item.labelKey) : undefined}
        >
          <Icon class="h-5 w-5 shrink-0" />
          {#if !collapsed}{t(item.labelKey)}{/if}
        </a>
      {/each}
    </div>
  </div>
</nav>
