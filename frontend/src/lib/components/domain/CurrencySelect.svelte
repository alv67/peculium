<script lang="ts">
  import { currencySymbol } from '#lib/format.js'
  import type { Currency } from '#lib/services/api.js'
  import Select from '#lib/components/ui/Select.svelte'

  /** Currency dropdown labelled with the shared currency symbols (EPIC E.3). */
  let {
    value = $bindable(''),
    currencies = [],
    disabled = false,
    class: className = '',
    ariaLabel = undefined,
  }: {
    value?: string
    currencies?: Currency[]
    disabled?: boolean
    class?: string
    ariaLabel?: string
  } = $props()
</script>

<Select bind:value {disabled} class={className} aria-label={ariaLabel}>
  {#each currencies as c (c.code)}
    <option value={c.code}>{currencySymbol(c.code)} {c.code}</option>
  {/each}
</Select>
