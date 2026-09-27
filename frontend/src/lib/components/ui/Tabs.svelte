<script lang="ts">
  import { page } from '$app/state'
  import { goto } from '$app/navigation'
  import { t } from '$lib/i18n/index.svelte'
  import { cx } from './utils'

  /**
   * Route-linked tabs for the tier-2 entity navigation (EPIC K.1c, redesign
   * spec §4.2/K.4): portfolio/asset sub-pages get their tab strip from here,
   * so every tab is a real `<a href>` — shareable, bookmarkable, and the
   * back button behaves. The active tab is derived from the current URL
   * (trailing-slash-insensitive), never passed in.
   *
   * ARIA follows the tab-with-links recipe (§9.1 keyboard rule): a
   * `role="tablist"` of `role="tab"` links inside a landmark `<nav>`,
   * `aria-selected` on the active tab, roving tabindex (the selected tab is
   * the single Tab stop; if the URL matches nothing, the first one is) and
   * Left/Right/Home/End arrows that move focus *and* activate the link via
   * `goto` with `keepFocus` (automatic activation: navigation is as cheap as
   * a click, and the strip must keep keyboard position across the routed
   * re-render). Enter on a focused tab is the anchor's own native activation.
   *
   * Callers pass `href`s they resolved with `$app/paths.resolve()` (repo
   * convention, see Button). `aria-controls`/tabpanels are intentionally not
   * emitted: the panel is the routed child page and gains the matching
   * wiring when K.4 adopts this component. The strip scrolls horizontally
   * when it overflows on small screens.
   */
  let {
    items,
    ariaLabel = undefined,
    class: className = '',
  }: {
    items: { href: string; label: string }[]
    /** Accessible name for the tablist (e.g. "Portfolio sections"). */
    ariaLabel?: string
    class?: string
  } = $props()

  const normalize = (path: string): string => path.replace(/\/+$/, '') || '/'
  const currentPath = $derived(normalize(page.url.pathname))

  function isActive(href: string): boolean {
    return normalize(href) === currentPath
  }

  const activeIndex = $derived(items.findIndex((item) => isActive(item.href)))

  /** Roving tabindex: exactly one stop — the active tab (or the first). */
  function isTabStop(index: number): boolean {
    return activeIndex === -1 ? index === 0 : index === activeIndex
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
    const tabs = Array.from(
      (event.currentTarget as HTMLElement).querySelectorAll<HTMLAnchorElement>('[role="tab"]'),
    )
    if (tabs.length === 0) return
    const current = tabs.indexOf(document.activeElement as HTMLAnchorElement)
    let next: number
    switch (event.key) {
      case 'ArrowRight':
        next = current < 0 ? 0 : (current + 1) % tabs.length
        break
      case 'ArrowLeft':
        next = current < 0 ? tabs.length - 1 : (current - 1 + tabs.length) % tabs.length
        break
      case 'Home':
        next = 0
        break
      default:
        next = tabs.length - 1
    }
    event.preventDefault()
    tabs[next].focus()
    // Roving focus that activates: the href comes from `items`, resolved by
    // the caller (see docstring), so the rule below cannot know it's safe.
    // eslint-disable-next-line svelte/no-navigation-without-resolve
    void goto(items[next].href, { keepFocus: true })
  }
</script>

<nav aria-label={ariaLabel ?? t('a11y.sections')} class={cx('min-w-0', className)}>
  <div
    role="tablist"
    tabindex="-1"
    onkeydown={handleKeydown}
    class="flex items-stretch gap-1 overflow-x-auto whitespace-nowrap border-b border-border"
  >
    {#each items as item, i (item.href)}
      <!-- Callers pass hrefs they resolved themselves ($app/paths `resolve()`),
           per the repo convention the rule below enforces on raw `<a href>`
           (same pattern as Button.svelte's anchor mode). -->
      <!-- eslint-disable svelte/no-navigation-without-resolve -->
      <a
        href={item.href}
        role="tab"
        aria-selected={isActive(item.href)}
        tabindex={isTabStop(i) ? 0 : -1}
        class={cx(
          'focus-ring inline-flex shrink-0 items-center border-b-2 px-3 py-2 text-sm font-medium transition-colors duration-fast ease-standard',
          isActive(item.href)
            ? 'border-accent text-foreground'
            : 'border-transparent text-muted-foreground hover:text-foreground',
        )}
      >
        {item.label}
      </a>
      <!-- eslint-enable svelte/no-navigation-without-resolve -->
    {/each}
  </div>
</nav>
