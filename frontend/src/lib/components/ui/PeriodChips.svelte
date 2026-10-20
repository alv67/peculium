<script lang="ts">
  import { cx } from './utils'

  /**
   * Compact period selector for charts (EPIC K.1c, redesign spec §8.2): the
   * chips live on/next to the chart they drive, never inside a menu. The
   * visual sibling of `SegmentedControl` (chip style instead of a pill track)
   * with true radio semantics: `role="radiogroup"` + `aria-checked`, roving
   * tabindex on the selected chip, arrow/Home/End keys move selection.
   *
   * Controlled: the active chip is `value`; selecting one calls `onchange`.
   * `periods` accepts `{ value, label }` objects or plain strings (used as
   * both value and label — tokens like `1Y`/`ALL` are already readable).
   *
   * NOTE: §8.2 also asks for per-scope persistence of the last-used period;
   * that lives in the consuming card (localStorage), not in this primitive.
   */
  type PeriodItem = { value: string; label: string }

  let {
    periods,
    value,
    onchange,
    ariaLabel = undefined,
    class: className = '',
  }: {
    periods: PeriodItem[] | string[]
    value: string
    onchange: (value: string) => void
    /** Accessible name for the whole group (e.g. "Performance period"). */
    ariaLabel?: string
    class?: string
  } = $props()

  const chips = $derived<PeriodItem[]>(
    periods.map((p) => (typeof p === 'string' ? { value: p, label: p } : p)),
  )
  const selectedIndex = $derived(chips.findIndex((p) => p.value === value))

  /** Button elements by index, for keyboard focus moves. */
  let chipRefs: (HTMLButtonElement | null)[] = $state([])

  function select(index: number): void {
    const chip = chips[index]
    if (!chip || chip.value === value) return
    onchange(chip.value)
    // Radios move focus together with selection; the keyed buttons survive
    // the re-render triggered by `onchange`, so focusing directly sticks.
    chipRefs[index]?.focus()
  }

  function handleKeydown(event: KeyboardEvent): void {
    const from = chips.findIndex((p) => p.value === (event.target as HTMLElement).dataset.value)
    const current = from === -1 ? selectedIndex : from
    let next: number
    switch (event.key) {
      case 'ArrowRight':
      case 'ArrowDown':
        next = (current + 1) % chips.length
        break
      case 'ArrowLeft':
      case 'ArrowUp':
        next = (current - 1 + chips.length) % chips.length
        break
      case 'Home':
        next = 0
        break
      case 'End':
        next = chips.length - 1
        break
      default:
        return
    }
    event.preventDefault()
    select(next)
  }
</script>

<div
  role="radiogroup"
  aria-label={ariaLabel}
  tabindex="-1"
  class={cx('flex flex-wrap items-center gap-1.5', className)}
  onkeydown={handleKeydown}
>
  {#each chips as chip, i (chip.value)}
    <button
      type="button"
      role="radio"
      bind:this={chipRefs[i]}
      aria-checked={chip.value === value}
      tabindex={chip.value === value ? 0 : -1}
      data-value={chip.value}
      onclick={() => select(i)}
      class={cx(
        'focus-ring rounded-full px-3 py-1 text-xs font-medium tabular-nums transition-colors duration-fast ease-standard',
        chip.value === value
          ? 'bg-accent text-accent-foreground'
          : 'border border-border text-muted-foreground hover:bg-muted hover:text-foreground',
      )}
    >
      {chip.label}
    </button>
  {/each}
</div>
