<script lang="ts">
  import type { Asset } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import Input from '$lib/components/ui/Input.svelte'
  import { cx } from '$lib/components/ui/utils'

  /**
   * Filterable combobox over the already-registered assets (EPIC E.2): picks
   * the asset of a transaction. Yahoo ticker lookup lives in
   * AssetSearchAutocomplete.
   */
  let {
    assets = [],
    value = $bindable(''),
    placeholder = t('asset.comboboxPlaceholder'),
    disabled = false,
    error = undefined,
    inputId = undefined,
  }: {
    assets?: Asset[]
    /** Selected asset id. */
    value?: string
    placeholder?: string
    disabled?: boolean
    /** Validation message forwarded to the inner `ui/Input`. */
    error?: string
    /** Set when the caller wraps this control in a `Field` with `for`. */
    inputId?: string
  } = $props()

  const MAX_VISIBLE = 8

  let wrapper = $state<HTMLDivElement | null>(null)
  let query = $state('')
  let typed = $state(false)
  let open = $state(false)

  const selected = $derived(assets.find((a) => a.id === value) ?? null)
  const label = $derived(
    selected ? (selected.name ? `${selected.ticker} - ${selected.name}` : selected.ticker) : '',
  )
  const filter = $derived(typed ? query.trim().toLowerCase() : '')
  const visible = $derived(
    (
      filter
        ? assets.filter(
            (a) =>
              a.ticker.toLowerCase().includes(filter) ||
              a.name.toLowerCase().includes(filter),
          )
        : assets
    ).slice(0, MAX_VISIBLE),
  )

  function handleFocus(event: FocusEvent): void {
    open = true
    // When the field shows the selected label, highlight it so the next
    // keystroke replaces it while the unfiltered list is already expanded.
    if (selected) (event.target as HTMLInputElement).select()
  }

  function handleInput(event: Event): void {
    query = (event.currentTarget as HTMLInputElement).value
    typed = true
    open = true
    if (!query.trim()) value = ''
  }

  function handleBlur(event: FocusEvent): void {
    // Keep the list open while focus moves to one of the suggestion buttons.
    const next = event.relatedTarget
    if (next instanceof Node && wrapper?.contains(next)) return
    close()
  }

  function pick(asset: Asset): void {
    // mousedown (blur-safe, also on browsers that don't focus buttons on
    // click) and click (keyboard) can both fire for one mouse press; pick is
    // idempotent and the list is gone after the first call.
    value = asset.id
    close()
  }

  function close(): void {
    open = false
    typed = false
    query = ''
  }
</script>

<div bind:this={wrapper} class="relative">
  <Input
    id={inputId}
    value={typed ? query : label}
    {placeholder}
    {disabled}
    {error}
    oninput={handleInput}
    onfocus={handleFocus}
    onblur={handleBlur}
  />

  {#if open}
    {#if visible.length > 0}
      <div
        class="absolute z-10 mt-1 max-h-48 w-full overflow-auto rounded-card border border-border bg-surface shadow-raised"
      >
        {#each visible as a (a.id)}
          <button
            type="button"
            onmousedown={() => pick(a)}
            onclick={() => pick(a)}
            class={cx(
              'flex w-full items-center gap-3 px-3 py-2 text-left text-sm hover:bg-muted',
              a.id === value && 'bg-muted',
            )}
          >
            <span class="font-medium">{a.ticker}</span>
            <span class="flex-1 truncate text-muted-foreground">{a.name}</span>
          </button>
        {/each}
      </div>
    {:else}
      <div
        class="absolute z-10 mt-1 w-full rounded-card border border-border bg-surface p-3 text-center text-sm text-muted-foreground shadow-raised"
      >
        {t('asset.noAssetsFound')}
      </div>
    {/if}
  {/if}
</div>
