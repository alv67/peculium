<script lang="ts">
  import { Search } from 'lucide-svelte'
  import { assetApi, type AssetLookupResult } from '$lib/services/api'
  import { t } from '$lib/i18n/index.svelte'
  import Input from '$lib/components/ui/Input.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'

  /**
   * Yahoo ticker lookup field (EPIC E.3): debounced autocomplete over
   * `assetApi.lookup`, extracted from the assets page inline form.
   */
  let {
    ticker = $bindable(''),
    onselect,
    disabled = false,
    placeholder = t('asset.tickerPlaceholder'),
    inputId = undefined,
  }: {
    ticker?: string
    onselect: (result: AssetLookupResult) => void
    disabled?: boolean
    placeholder?: string
    /** Set when the caller wraps this control in a `Field` with `for`. */
    inputId?: string
  } = $props()

  let wrapper = $state<HTMLDivElement | null>(null)
  let results = $state<AssetLookupResult[] | null>(null)
  let loading = $state(false)
  let open = $state(false)
  let selected = $state(false)
  let debounceTimer: ReturnType<typeof setTimeout> | undefined

  function handleInput(event: Event): void {
    const value = (event.currentTarget as HTMLInputElement).value.toUpperCase()
    ticker = value
    selected = false
    open = value.length >= 2

    clearTimeout(debounceTimer)
    if (value.length < 2) {
      results = null
      loading = false
      return
    }

    loading = true
    debounceTimer = setTimeout(async () => {
      try {
        results = await assetApi.lookup(value)
      } catch {
        results = []
      } finally {
        loading = false
      }
    }, 350)
  }

  function handleBlur(event: FocusEvent): void {
    // Keep the list open while focus moves to one of the suggestion buttons.
    const next = event.relatedTarget
    if (next instanceof Node && wrapper?.contains(next)) return
    open = false
  }

  function pick(result: AssetLookupResult): void {
    // mousedown (blur-safe) and click (keyboard) can both fire for one mouse
    // press: the selected guard makes the second call a no-op.
    if (selected && ticker === result.ticker) return
    ticker = result.ticker
    selected = true
    open = false
    onselect(result)
  }

  $effect(() => () => clearTimeout(debounceTimer))
</script>

<div bind:this={wrapper} class="relative">
  <Input
    id={inputId}
    value={ticker}
    {placeholder}
    {disabled}
    class="pr-8"
    oninput={handleInput}
    onfocus={() => {
      if (ticker.length >= 2) open = true
    }}
    onblur={handleBlur}
  />

  {#if loading}
    <Spinner
      size="sm"
      class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground"
    />
  {:else if ticker && !selected}
    <Search
      class="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
    />
  {/if}

  {#if open && !selected && results}
    {#if results.length > 0}
      <div
        class="absolute z-10 mt-1 max-h-48 w-full overflow-auto rounded-card border border-border bg-surface shadow-raised"
      >
        {#each results as r (r.ticker)}
          <button
            type="button"
            onmousedown={() => pick(r)}
            onclick={() => pick(r)}
            class="flex w-full items-center gap-3 px-3 py-2 text-left text-sm hover:bg-muted"
          >
            <span class="font-medium">{r.ticker}</span>
            <span class="flex-1 truncate text-muted-foreground">{r.name}</span>
            <span class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground"
              >{r.exchange}</span
            >
            <span class="text-xs text-muted-foreground">{r.type}</span>
          </button>
        {/each}
      </div>
    {:else}
      <div
        class="absolute z-10 mt-1 w-full rounded-card border border-border bg-surface p-3 text-center text-sm text-muted-foreground shadow-raised"
      >
        {t('asset.noResultsFound')}
      </div>
    {/if}
  {/if}
</div>
