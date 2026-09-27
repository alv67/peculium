<script lang="ts">
  import { goto } from '$app/navigation'
  import { resolve } from '$app/paths'
  import { t } from '$lib/i18n/index.svelte'
  import Select from '../ui/Select.svelte'

  /**
   * Wealth ⇄ portfolio scope switcher (EPIC K.3a, decision D3): it is
   * navigation, not a filter — picking a portfolio `goto()`s its detail page
   * (which renders the same analytics at portfolio scope), while the wealth
   * option is the current page. Built from the dashboard payload's own
   * portfolio list, so it costs no extra request.
   *
   * A native `<select>` (through the `ui/Select` recipe) is the accessibility
   * floor: keyboard, mobile and screen-reader behaviour come for free, which
   * is exactly what the spec asks before a custom dropdown is ever worth
   * building.
   */
  let {
    portfolios,
    class: className = '',
  }: {
    /** Minimal id+name projection so any portfolio summary shape fits. */
    portfolios: readonly { portfolio_id: string; portfolio_name: string }[]
    class?: string
  } = $props()

  function handleScopeChange(event: Event): void {
    const id = (event.currentTarget as HTMLSelectElement).value
    // Selecting the wealth option (empty value) stays on the current page;
    // any portfolio selection leaves, so the select never needs resetting.
    if (id) void goto(resolve(`/portfolios/${id}`))
  }
</script>

<div class={className}>
  <Select aria-label={t('scope.label')} onchange={handleScopeChange}>
    <option value="">{t('scope.all')}</option>
    {#each portfolios as p (p.portfolio_id)}
      <option value={p.portfolio_id}>{p.portfolio_name}</option>
    {/each}
  </Select>
</div>
