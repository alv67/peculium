<script lang="ts">
  import { Banknote, PanelLeft, Search } from 'lucide-svelte'
  import { browser } from '$app/environment'
  import { resolve } from '$app/paths'
  import { t } from '$lib/i18n/index.svelte'
  import Button from '../ui/Button.svelte'
  import PriceRefreshButton from './PriceRefreshButton.svelte'
  import ThemeToggle from './ThemeToggle.svelte'
  import UserMenu from './UserMenu.svelte'

  /**
   * Sticky top bar (EPIC D.3, adaptive since K.2). Left: the sidebar
   * collapse toggle — `lg`+ only, since phones have no sidebar (bottom nav
   * + More sheet, decision D2) and tablets are a forced icon rail — plus the
   * brand link shown below `lg` (the rail/drawer carry it otherwise).
   * Right: the global price-freshness control (`PriceRefreshButton` —
   * stamp + manual refresh trigger, always immediately before the search
   * button), the command-palette trigger (EPIC K.5a — a search affordance
   * at every size, spec §8.1; on `lg`+ it grows into a labelled pill with the
   * ⌘K chord hint, on smaller viewports it stays the icon-only button, so
   * phones get palette access without a 5th bottom-nav item), the theme
   * toggle at every size (the rail and the More sheet have no theme control
   * of their own) and the user menu, phone-only
   * (`sm:hidden`): from `sm` up the sidebar rail's footer hosts one already
   * so the header must not duplicate it, while on phones — where the More
   * sheet is navigation only — account and theme stay in the header (spec
   * §5.2). The control
   * aria-labels go through `t()` (EPIC K.1b, decision D1); the "Peculium"
   * brand is a proper noun and stays as-is.
   *
   * Condensing (spec §5.1): the shell measures its main scroll container and
   * publishes the live bar height as `--app-header-h` (3.5rem → 2.75rem,
   * i.e. 56px → 44px) on the scroll column — this bar and the entity sticky
   * headers below it read the same variable, so they always stay flush. The
   * height transitions with plain CSS, which the global
   * `prefers-reduced-motion` rule in app.css already neutralises. The
   * trigger is `size="icon"` (h-9), so it survives the shrink unchanged.
   *
   * z-20: same tier as the dropdowns it hosts (header and its popovers must
   * both stay under the drawer/sheet tier, z-30).
   */
  let {
    collapsed,
    ontogglecollapse,
    onopenpalette,
  }: {
    /** Sidebar rail state (drives the collapse toggle's label/pressed). */
    collapsed: boolean
    ontogglecollapse: () => void
    /** Opens the K.5a command palette (the shell owns its bindable state). */
    onopenpalette?: () => void
  } = $props()

  // The ⌘K hint carries the platform's real chord (⌘ on Apple, Ctrl
  // elsewhere); the palette component handles both chords identically.
  const chord =
    browser && /Mac|iPhone|iPad|iPod/i.test(navigator.userAgent) ? '⌘K' : 'Ctrl K'
</script>

<header
  class="sticky top-0 z-20 flex h-[var(--app-header-h)] shrink-0 items-center justify-between gap-2 border-b border-border bg-surface/90 px-3 backdrop-blur transition-[height] duration-base ease-standard lg:px-4"
>
  <div class="flex min-w-0 items-center gap-1">
    <Button
      variant="ghost"
      size="icon"
      class="hidden lg:inline-flex"
      aria-label={collapsed ? t('header.expandSidebar') : t('header.collapseSidebar')}
      aria-pressed={collapsed}
      onclick={ontogglecollapse}
    >
      <PanelLeft class="h-5 w-5" />
    </Button>
    <a
      href={resolve('/')}
      class="focus-ring flex items-center gap-2 rounded-control px-1 lg:hidden"
    >
      <Banknote class="h-5 w-5 shrink-0 text-accent-text" aria-hidden="true" />
      <span class="truncate text-base font-bold">Peculium</span>
    </a>
  </div>

  <div class="flex shrink-0 items-center gap-1">
    <!-- Global price freshness + manual refresh trigger (always visible,
         immediately before the search button). -->
    <PriceRefreshButton />
    <!-- Command-palette trigger (K.5a): icon-only below `lg`, a labelled
         pill with the chord hint from `lg`. aria-haspopup mirrors the Fab's
         quick-actions wiring: it opens a dialog, not a menu. -->
    <Button
      variant="ghost"
      size="icon"
      class="lg:w-auto lg:gap-2 lg:px-3"
      aria-label={t('commandPalette.trigger')}
      aria-haspopup="dialog"
      onclick={() => onopenpalette?.()}
    >
      <Search class="h-5 w-5 shrink-0" aria-hidden="true" />
      <span class="hidden text-sm lg:inline">{t('commandPalette.inputLabel')}</span>
      <kbd class="hidden rounded-md border border-border bg-muted px-1.5 py-0.5 font-mono text-micro text-muted-foreground lg:inline" aria-hidden="true"
        >{chord}</kbd
      >
    </Button>
    <ThemeToggle />
    <UserMenu compact direction="down" align="end" class="sm:hidden" />
  </div>
</header>
