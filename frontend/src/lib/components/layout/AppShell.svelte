<script lang="ts">
  import type { Snippet } from 'svelte'
  import { onMount } from 'svelte'
  import { portfolioApi } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import { priceRefresh, refreshPrices } from '$lib/stores/priceRefresh.svelte'
  import { applyDashboardStatus, dashboardStatus } from '$lib/stores/dashboardStatus.svelte'
  import { viewport } from '$lib/stores/viewport.svelte'
  import DataQualityStrip from '../domain/DataQualityStrip.svelte'
  import AppHeader from './AppHeader.svelte'
  import BottomNav from './BottomNav.svelte'
  import CommandPalette from './CommandPalette.svelte'
  import Fab from './Fab.svelte'
  import MobileDrawer from './MobileDrawer.svelte'
  import Sidebar from './Sidebar.svelte'
  import UserMenu from './UserMenu.svelte'

  /**
   * Responsive application shell (EPIC D.3, adaptive since EPIC K.2) —
   * replaces the old fixed-width Layout.svelte.
   *
   * Three device classes (spec §5.1, decision D2):
   * - phone (< `sm`): no sidebar at all — a fixed `BottomNav` (Overview,
   *   Portfolios, Assets, More) and the quick-actions `Fab` mount over the
   *   content column, and `<main>` gets an extra bottom pad so nothing hides
   *   behind the bar (56px + safe area + breathing room). The hamburger is
   *   gone: `MobileDrawer` survives as the "More" sheet, opened only by the
   *   bottom nav;
   * - tablet (`sm`–`lg`): the sidebar is always the 64px icon rail — the
   *   persisted expand preference deliberately only applies at `lg`+;
   * - desktop (`lg`+): unchanged — expandable/collapsible sidebar, sticky
   *   header with the collapse toggle, user menu in the sidebar footer.
   *
   * State owned here and pushed down:
   * - `collapsed` — persisted under `peculium-sidebar` so the rail survives
   *   reloads (the app is SPA-only: `export const ssr = false`, so reading
   *   localStorage at init never runs on the server);
    * - `moreOpen` — the phone More sheet (the drawer handles its own
    *   close-on-navigation + focus trap);
     * - `condensed` — measured on the main scroll container past a small
     *   threshold and published as the `--app-header-h` custom property (the
     *   header height and every sticky page header derive their offsets from
     *   it, §5.1 "condenses on scroll");
    * - `paletteOpen` — the K.5a command palette (⌘K/Ctrl+K chord, the header
    *   search trigger and Esc all share this one bindable state).
    */
  const SIDEBAR_STORAGE_KEY = 'peculium-sidebar'
  // Legacy storage key: when the new key is absent, the value stored here is
  // adopted and written forward once, so no saved preference is lost.
  const LEGACY_SIDEBAR_STORAGE_KEY = 'vaultlab-sidebar'

  /** Scroll distance (px) after which the sticky header condenses. */
  const CONDENSE_THRESHOLD = 16

  let { children }: { children: Snippet } = $props()

  function readCollapsed(): boolean {
    try {
      let stored = localStorage.getItem(SIDEBAR_STORAGE_KEY)
      if (stored === null) {
        // One-time migration: adopt the legacy value and write it forward.
        const legacy = localStorage.getItem(LEGACY_SIDEBAR_STORAGE_KEY)
        if (legacy !== null) {
          localStorage.setItem(SIDEBAR_STORAGE_KEY, legacy)
          stored = legacy
        }
      }
      return stored === 'true'
    } catch {
      // Storage unavailable (e.g. private mode): always start expanded.
      return false
    }
  }

  let collapsed = $state(readCollapsed())
  let moreOpen = $state(false)
  let condensed = $state(false)
  let paletteOpen = $state(false)
  let scrollContainer = $state<HTMLElement | null>(null)

  // Tablet forces the rail (spec §5.1: on the `sm`–`lg` classes the sidebar
  // *is* the rail); desktop keeps the persisted preference. Below `sm` the
  // sidebar is not rendered at all.
  const railCollapsed = $derived(viewport.isDesktop ? collapsed : true)

  function toggleCollapsed(): void {
    collapsed = !collapsed
    try {
      localStorage.setItem(SIDEBAR_STORAGE_KEY, String(collapsed))
    } catch {
      // Storage unavailable: the state still applies for this visit.
    }
  }

  // The scrollable element is this shell column (not the window), so the
  // condensing state must be measured here; passive listener because the
  // handler never cancels the scroll (K.2 header).
  $effect(() => {
    const el = scrollContainer
    if (!el) return
    const update = () => (condensed = el.scrollTop > CONDENSE_THRESHOLD)
    update()
    el.addEventListener('scroll', update, { passive: true })
    return () => el.removeEventListener('scroll', update)
  })

  // Live header height, published as a custom property on this column —
  // the closest ancestor of both `AppHeader` and every page (so entity
  // sticky headers stack with `top-[var(--app-header-h)]` and never leave a
  // gap while the bar condenses 56px → 44px). Keep in sync with the
  // `h-*`/`top-*` utilities and the `:root` fallback in app.css.
  const headerHeight = $derived(condensed ? '2.75rem' : '3.5rem')

  // Global session boot (shell = mounted once per authenticated visit, on
  // ANY landing page): fire the once-per-session price refresh through the
  // shared store path — silent, both toasts are disabled because the header
  // freshness control and the strip below render the outcome persistently —
  // and seed the strip's FX counters from the vault dashboard payload (the
  // dashboard page keeps them fresh afterwards via `applyDashboardStatus`).
  onMount(() => {
    if (!priceRefresh.started) void refreshPrices({ announceSuccess: false, announceError: false })
    if (!dashboardStatus.loaded) {
      portfolioApi
        .dashboard()
        .then(applyDashboardStatus)
        .catch(() => {
          // Silent: the strip simply has nothing to report yet.
        })
    }
  })
</script>

<a
  href="#content"
  class="focus-ring sr-only rounded-control bg-surface px-4 py-2 text-sm font-medium text-foreground focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50"
>
  {t('nav.skipToContent')}
</a>

<div
  class="fixed inset-x-0 top-0 flex h-dvh overflow-hidden bg-background text-foreground"
>
  <aside class="hidden shrink-0 sm:flex">
    <Sidebar collapsed={railCollapsed}>
      {#snippet footer()}
        <UserMenu compact={railCollapsed} />
      {/snippet}
    </Sidebar>
  </aside>

  <div
    bind:this={scrollContainer}
    class="flex min-w-0 flex-1 flex-col overflow-y-auto"
    style={`--app-header-h: ${headerHeight}`}
  >
    <AppHeader
      {collapsed}
      ontogglecollapse={toggleCollapsed}
      onopenpalette={() => (paletteOpen = true)}
    />
    <!-- Global data-quality strip (was the dashboard hero's): a thin sticky
         band under the condensing header, rendered only when at least one
         vault counter or refresh outcome has something to report, so the
         warnings travel with the user on every page. -->
    {#if dashboardStatus.fxMissingCount > 0 || priceRefresh.rateLimited || priceRefresh.issueCount > 0 || priceRefresh.failed}
      <div class="sticky top-[var(--app-header-h)] z-10 border-b border-border bg-background px-4 py-2 lg:px-6">
        <DataQualityStrip
          currency={dashboardStatus.currency}
          fxMissingCount={dashboardStatus.fxMissingCount}
          fxMissingValue={dashboardStatus.fxMissingValue}
          rateLimited={priceRefresh.rateLimited}
          issueCount={priceRefresh.issueCount}
          refreshFailed={priceRefresh.failed}
        />
      </div>
    {/if}
    <!-- Longhand padding utilities only: `p-*` shorthand would fight the
         phone-only `pb-[…]` clearance below the fixed bottom nav. -->
    <main
      id="content"
      class="min-w-0 flex-1 px-4 pt-4 pb-[calc(6rem_+_env(safe-area-inset-bottom))] outline-none sm:pb-4 lg:px-6 lg:pt-6 lg:pb-6"
      tabindex="-1"
    >
      {@render children()}
    </main>
  </div>

  <!-- Phone bottom chrome lives INSIDE the shell. The shell is
       `position: fixed`, which forms a stacking context: keeping the bottom
       nav here (instead of as a sibling) puts it in the same context as the
       page overlays, so a `z-30` Drawer/Sheet paints above the `z-20` bar
       rather than being trapped underneath it. Both are `position: fixed`,
       so they still anchor to the viewport and do not join the flex row. -->
  {#if viewport.isPhone}
    <BottomNav {moreOpen} onopenmore={() => (moreOpen = true)} />
    <Fab />
  {/if}
</div>

<MobileDrawer bind:open={moreOpen}>
  <Sidebar collapsed={false} />
</MobileDrawer>

<!-- Global command palette (EPIC K.5a, spec §8.1): mounted once here — the
     ⌘K/Ctrl+K chord lives in the component, the header search button opens
     it, and it borrows `toggleCollapsed` for its Toggle-sidebar action. -->
<CommandPalette bind:open={paletteOpen} ontogglesidebar={toggleCollapsed} />
