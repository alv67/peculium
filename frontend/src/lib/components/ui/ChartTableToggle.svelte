<script lang="ts">
  import SegmentedControl from './SegmentedControl.svelte'
  import { t } from '#lib/i18n/index.svelte.js'

  /**
   * Chart ⇄ Table segmented toggle (EPIC K.5b, redesign spec §9.1): the
   * shared control embedded in every data-bearing chart wrapper so the
   * card's numbers can also be read as an accessible table (which doubles
   * as the mobile data experience and the screen-reader experience).
   *
   * Only the control lives here — the wrappers own the `view` state and
   * the table markup, and unmount the canvas in table mode, so the chart
   * automatically leaves the accessibility tree while its table is shown.
   *
   * `name` (the chart's own heading, when it has one) is interpolated into
   * the tablist's accessible name so several toggles on one page never
   * collapse to the same label. Keyboard: SegmentedControl renders real
   * buttons with `aria-selected`, so tab + Enter/Space work out of the box.
   */
  let {
    name = undefined,
    view = $bindable('chart'),
    class: className = '',
  }: {
    /** Chart/card title interpolated into the accessible name. */
    name?: string
    /** Bindable active view; wrappers branch their body on it. */
    view?: 'chart' | 'table'
    class?: string
  } = $props()

  const items = $derived([
    { value: 'chart', label: t('chartView.chart') },
    { value: 'table', label: t('chartView.table') },
  ])
  const ariaLabel = $derived(name
    ? t('chartView.ariaNamed', { name })
    : t('chartView.aria'))

  // SegmentedControl binds a plain string; the accessors keep the union.
  const getView = (): string => view
  const setView = (next: string): void => {
    if (next === 'chart' || next === 'table') view = next
  }
</script>

<SegmentedControl
  {items}
  bind:value={getView, setView}
  {ariaLabel}
  class={className}
/>
