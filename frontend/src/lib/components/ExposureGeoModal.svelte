<script lang="ts">
  import { tick, untrack } from 'svelte'
  import { Calculator, Globe2, Info, Loader2, X, Plus, Trash2 } from 'lucide-svelte'
  import type { ExposureRow } from '#lib/services/api.js'
  import { CANONICAL_COUNTRIES, countryDisplayName } from '#lib/countryNames.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import ExposurePie from './ExposurePie.svelte'
  import ProvenanceBadge from './ProvenanceBadge.svelte'
  import { colorForRow, resolvePalette } from '#lib/chartPalette.js'
  import { resolved } from '#lib/stores/theme.svelte.js'
  import { totalColorClass } from '#lib/ui-colors.js'

  let {
    open = $bindable(false),
    onClose,
    regionsEdit = $bindable([] as ExposureRow[]),
    countriesEdit = $bindable([] as ExposureRow[]),
    sumRegions = 0,
    sumCountries = 0,
    savingRegions = false,
    savingCountries = false,
    saveRegions,
    saveCountries,
    fetchingETF = false,
    fetchingMorningstar = false,
    derivingRegions = false,
    prefillCountriesFromETF,
    deriveRegionsFromCountries,
    prefillRegionsFromMorningstar,
    prefillCountriesFromMorningstar,
    countriesSource = null as string | null,
    regionsSource = null as string | null,
    countriesUpdatedAt = null as string | null,
    regionsUpdatedAt = null as string | null,
    onCountriesDirty,
    onRegionsDirty,
    assetType = 'stock',
  }: {
    open: boolean
    onClose: () => void
    regionsEdit: ExposureRow[]
    countriesEdit: ExposureRow[]
    sumRegions: number
    sumCountries: number
    savingRegions: boolean
    savingCountries: boolean
    saveRegions: () => void
    saveCountries: () => void
    fetchingETF: boolean
    fetchingMorningstar: boolean
    derivingRegions: boolean
    prefillCountriesFromETF: () => void
    deriveRegionsFromCountries: () => void
    prefillRegionsFromMorningstar: () => void
    prefillCountriesFromMorningstar: () => void
    countriesSource: string | null
    regionsSource: string | null
    countriesUpdatedAt: string | null
    regionsUpdatedAt: string | null
    onCountriesDirty: () => void
    onRegionsDirty: () => void
    assetType: string
  } = $props()

  /** Country codes currently present in the countries edit list. */
  const presentCodes = $derived(new Set(countriesEdit.map((r) => r.name)))

  /** Canonical codes not yet in the edit list — available for the add
   *  dropdown, ordered by friendly country name (not ISO code). */
  const availableCodes = $derived(
    CANONICAL_COUNTRIES.filter((c) => !presentCodes.has(c)).sort((a, b) =>
      countryDisplayName(a).localeCompare(countryDisplayName(b), 'it'),
    ),
  )

  let addCountryCode = $state('')

  // Keep the dropdown selection valid: if the current choice is already present
  // (e.g. after a Morningstar prefill), reset to the first available code.
  $effect(() => {
    const codes = availableCodes
    if (codes.length > 0 && !codes.includes(addCountryCode)) {
      addCountryCode = codes[0]
    }
  })

  // ---------------------------------------------------------------------------
  // Countries sorting
  //
  // The rows are the page-owned edit list (already filtered to weight > 0 at
  // assignment time; the only zero-weight rows are ones the user just added).
  // Display order is a snapshot of codes, re-sorted ONLY on list replacement
  // (load/prefill/save), add, remove and on input commit (change/blur) — never
  // while typing, so rows don't jump under the cursor; the animated bar gives
  // the live feedback instead.
  // ---------------------------------------------------------------------------
  let orderedCodes = $state<string[]>([])

  function resort(): void {
    orderedCodes = [...countriesEdit]
      .sort((a, b) => (Number(b.weight) || 0) - (Number(a.weight) || 0))
      .map((r) => r.name)
  }

  $effect(() => {
    // Reading each row's name (never its weight) tracks list replacements
    // without reacting to in-place weight edits.
    const names = countriesEdit.map((r) => r.name)
    untrack(() => {
      if (names.length === 0) orderedCodes = []
      else resort()
    })
  })

  const visibleCountries = $derived.by(() =>
    orderedCodes
      .map((code) => countriesEdit.find((r) => r.name === code))
      .filter((r): r is ExposureRow => r !== undefined),
  )

  // Largest visible weight: bars are scaled proportionally against it, same
  // formula as the page card bar list.
  const maxCountryWeight = $derived(
    visibleCountries.reduce((max, r) => Math.max(max, Number(r.weight) || 0), 0),
  )

  // Save/validation rule for both dimensions: sums up to 100 are valid (the
  // backend absorbs the residual), sums above 100 (± float epsilon) block the
  // save with an inline alert.
  const countriesOver = $derived(sumCountries > 100 + 1e-9)
  const regionsOver = $derived(sumRegions > 100 + 1e-9)

  function removeCountry(code: string): void {
    countriesEdit = countriesEdit.filter((r) => r.name !== code)
    onCountriesDirty()
    resort()
  }

  async function addCountry(): Promise<void> {
    if (!addCountryCode || presentCodes.has(addCountryCode)) return
    const code = addCountryCode
    countriesEdit = [...countriesEdit, { name: code, weight: '0' }]
    onCountriesDirty()
    resort()
    // Reset selection to the first country by name (availableCodes is
    // name-ordered, so the dropdown always proposes the alphabetically
    // first not-yet-added country).
    const next = availableCodes[0] ?? ''
    addCountryCode = next
    await tick()
    // Land the user straight on the new row's weight input.
    const input = document.querySelector<HTMLInputElement>(
      `[data-code="${code}"] input[type='number']`,
    )
    input?.focus()
    input?.select()
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape' && open) {
      onClose()
    }
  }

  function handleBackdropClick(e: MouseEvent): void {
    if (e.target === e.currentTarget) {
      onClose()
    }
  }

  // Resolved chart palette: keeps the HTML bars/swatches in sync with the
  // donut colors and re-evaluates on theme flips.
  const palette = $derived(resolvePalette(resolved()))
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-overlay/50"
    onclick={handleBackdropClick}
    onkeydown={(e) => e.key === 'Escape' && onClose()}
    role="dialog"
    aria-modal="true"
    aria-label={t('exposure.editGeo')}
    tabindex="-1"
  >
    <!-- max-w-6xl (1152px): with the lg:grid-cols-2 split each box gets ~536px
         of width, so the regions table (~264px after the 224px donut) has room
         for long region names like "Africa / Middle East" on one line. -->
    <div
      class="relative mx-2 max-h-[90vh] w-full max-w-6xl overflow-y-auto rounded-card border-border bg-surface p-3 shadow-raised sm:mx-4 sm:p-6"
    >
      <div class="mb-6 flex items-center justify-between">
        <h2 class="text-lg font-semibold">{t('exposure.editGeo')}</h2>
        <button
          onclick={onClose}
          class="rounded-control p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label={t('common.close')}
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <div class="grid grid-cols-1 gap-8 lg:grid-cols-2">
        <!-- Countries (regions update only manually, via the derive button,
             `exposure.deriveTitle`) -->
        <div class="flex flex-col rounded-card border border-border bg-muted p-3 sm:p-4">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <h3 class="font-medium">{t('exposure.countries')}</h3>
              <ProvenanceBadge source={countriesSource} updatedAt={countriesUpdatedAt} />
            </div>
            <div class="flex shrink-0 items-center gap-1.5">
              <button
                onclick={prefillCountriesFromETF}
                disabled={assetType !== 'etf' || fetchingETF || fetchingMorningstar}
                title={t('exposure.prefillJustEtf')}
                aria-label={t('exposure.prefillCountriesJustEtf')}
                class="rounded-control border border-input bg-surface p-1.5 shadow-card hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
              >
                {#if fetchingETF}
                  <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
                {:else}
                  <img
                    class="h-5 w-5 rounded"
                    src="https://www.google.com/s2/favicons?domain=justetf.com&sz=32"
                    alt="JustETF"
                  />
                {/if}
              </button>
              <button
                onclick={prefillCountriesFromMorningstar}
                disabled={assetType !== 'etf' || fetchingETF || fetchingMorningstar}
                title={t('exposure.prefillMorningstar')}
                aria-label={t('exposure.prefillCountriesMorningstar')}
                class="rounded-control border border-input bg-surface p-1.5 shadow-card hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
              >
                {#if fetchingMorningstar}
                  <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
                {:else}
                  <img
                    class="h-5 w-5 rounded"
                    src="https://www.google.com/s2/favicons?domain=www.morningstar.com&sz=32"
                    alt="Morningstar"
                  />
                {/if}
              </button>
            </div>
          </div>

          <!-- Middle area at natural height: the modal body is the only
               scroller, so the list never steals width from the rows. -->
          <div class="flex flex-col">
            <!-- Table-style header row, columns aligned with the rows below. -->
            <div
              class="flex items-center gap-1.5 border-b border-border pb-2 pl-2 pr-2 text-sm text-muted-foreground"
            >
              <span class="min-w-0 flex-1 truncate">{t('exposure.colCountry')}</span>
              <span class="w-16 shrink-0 text-right">{t('exposure.colWeightPct')}</span>
              <!-- Spacer matching the remove-button column of the rows. -->
              <span class="w-[22px] shrink-0" aria-hidden="true"></span>
            </div>

            {#if visibleCountries.length === 0}
              <div class="flex flex-col items-center justify-center py-10 text-center">
                <Globe2 class="h-8 w-8 text-input" />
                <p class="text-sm text-muted-foreground">{t('exposure.noCountries')}</p>
                <p class="text-xs text-muted-foreground">
                  {t('exposure.noCountriesHint')}
                </p>
              </div>
            {:else}
              <ul class="divide-y divide-border py-1">
                {#each visibleCountries as row, i (row.name)}
                  {@const weight = Number(row.weight) || 0}
                  {@const displayName = countryDisplayName(row.name)}
                  {@const barPct =
                    maxCountryWeight > 0 ? ((weight / maxCountryWeight) * 100).toFixed(1) : '0'}
                  <li
                    data-code={row.name}
                    class="group flex items-center gap-1.5 rounded-control px-2 py-2 transition-colors hover:bg-foreground/5 motion-reduce:transition-none"
                  >
                    <span class="w-6 shrink-0 text-xs font-medium text-muted-foreground">{row.name}</span>
                    <span
                      class="min-w-0 flex-1 truncate text-sm text-foreground"
                      title="{row.name} — {displayName}"
                    >{displayName}</span>
                    <div
                      class="hidden h-2.5 w-24 shrink-0 overflow-hidden rounded-full bg-input sm:block"
                      aria-hidden="true"
                    >
                      <div
                        class="h-full rounded-full transition-[width] duration-200 motion-reduce:transition-none"
                        style="width: {barPct}%; background-color: {palette[i % palette.length]};"
                      ></div>
                    </div>
                    <input
                      type="number"
                      min="0"
                      max="100"
                      step="0.01"
                      inputmode="decimal"
                      value={row.weight}
                      aria-label={t('exposure.weightAria', { name: displayName })}
                      oninput={(e) => {
                        row.weight = e.currentTarget.value
                        onCountriesDirty()
                      }}
                      onchange={resort}
                      class="no-spinner focus-ring w-16 shrink-0 rounded-control border border-input bg-surface px-2 py-1.5 text-right text-sm tabular-nums"
                    />
                    <button
                      onclick={() => removeCountry(row.name)}
                      class="shrink-0 rounded p-1 text-muted-foreground hover:bg-negative/10 hover:text-negative"
                      title={t('exposure.removeAria', { name: displayName })}
                      aria-label={t('exposure.removeAria', { name: displayName })}
                    >
                      <Trash2 class="h-3.5 w-3.5" />
                    </button>
                  </li>
                {/each}
              </ul>
            {/if}

            {#if availableCodes.length > 0}
              <div class="flex items-center gap-2 pt-3">
                <select
                  bind:value={addCountryCode}
                  aria-label={t('exposure.addCountryAria')}
                  class="focus-ring min-w-0 flex-1 rounded-control border border-input bg-surface px-3 py-1.5 text-sm"
                >
                  {#each availableCodes as code (code)}
                    <option value={code}>{code} — {countryDisplayName(code)}</option>
                  {/each}
                </select>
                <button
                  onclick={addCountry}
                  disabled={!addCountryCode}
                  class="flex shrink-0 items-center gap-1 rounded-control border border-input bg-surface px-3 py-1.5 text-sm font-medium text-foreground shadow-card hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <Plus class="h-4 w-4" />
                  {t('exposure.add')}
                </button>
              </div>
            {/if}
          </div>

          <!-- Footer rows (identical in both boxes so they line up):
               separator, fixed-height total, reserved 3-line message area,
               fixed-height Save area. -->
          <div class="mt-3 border-t border-border" aria-hidden="true"></div>

          <div class="mt-3 h-8">
            <div class="flex items-center justify-between">
              <span class="text-sm font-medium text-foreground">{t('common.colTotal')}</span>
              <span
                class="text-sm font-semibold tabular-nums {totalColorClass(sumCountries, countriesOver)}"
              >{sumCountries.toFixed(2)}%</span>
            </div>
            <div class="mt-1.5 h-1.5 overflow-hidden rounded-full bg-input" aria-hidden="true">
              <div
                class="h-full rounded-full transition-[width] duration-200 motion-reduce:transition-none {countriesOver
                  ? 'bg-negative'
                  : 'bg-accent'}"
                style="width: {Math.min(sumCountries, 100).toFixed(2)}%;"
              ></div>
            </div>
          </div>

          <div class="h-[3.75rem] overflow-hidden pt-1 text-xs leading-5">
            {#if countriesOver}
              <p role="alert" class="text-negative">
                {t('exposure.overSum', { pct: sumCountries.toFixed(2) })}
              </p>
            {:else if sumCountries < 99.5}
              <p class="text-muted-foreground">
                {t('exposure.residualCountries', { pct: (100 - sumCountries).toFixed(2) })}
              </p>
            {/if}
          </div>

          <div class="flex h-12 items-center justify-end">
            <button
              onclick={saveCountries}
              disabled={savingCountries || countriesOver}
              title={countriesOver ? t('exposure.over100Title') : undefined}
              class="flex items-center gap-2 rounded-control bg-accent px-4 py-2 text-sm text-accent-foreground hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              {#if savingCountries}
                <Loader2 class="h-4 w-4 animate-spin" />
              {/if}
              {savingCountries ? t('common.saving') : t('common.save')}
            </button>
          </div>
        </div>

        <!-- Regions -->
        <div class="flex flex-col rounded-card border border-border bg-muted p-3 sm:p-4">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <h3 class="font-medium">{t('exposure.regions')}</h3>
              <ProvenanceBadge source={regionsSource} updatedAt={regionsUpdatedAt} />
            </div>
            <div class="flex shrink-0 items-center gap-1.5">
              <button
                onclick={deriveRegionsFromCountries}
                disabled={derivingRegions || fetchingMorningstar}
                title={t('exposure.deriveTitle')}
                aria-label={t('exposure.deriveAria')}
                class="rounded-control border border-input bg-surface p-1.5 shadow-card hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
              >
                {#if derivingRegions}
                  <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
                {:else}
                  <Calculator class="h-5 w-5 text-muted-foreground" />
                {/if}
              </button>
              <button
                onclick={prefillRegionsFromMorningstar}
                disabled={assetType !== 'etf' || derivingRegions || fetchingMorningstar}
                title={t('exposure.prefillMorningstar')}
                aria-label={t('exposure.prefillRegionsMorningstar')}
                class="rounded-control border border-input bg-surface p-1.5 shadow-card hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
              >
                {#if fetchingMorningstar}
                  <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
                {:else}
                  <img
                    class="h-5 w-5 rounded"
                    src="https://www.google.com/s2/favicons?domain=www.morningstar.com&sz=32"
                    alt="Morningstar"
                  />
                {/if}
              </button>
            </div>
          </div>

          <!-- Middle area at natural height (header row + 10 region rows and
               the side donut fully visible, no fixed quota, no scrollbar). -->
          <div class="flex flex-col gap-4 md:flex-row">
            <div class="min-w-0 flex-1">
              <table class="w-full text-left text-sm">
                <thead>
                  <tr class="border-b border-border text-muted-foreground">
                    <th class="pb-2">{t('exposure.geoArea')}</th>
                    <th class="pb-2 text-right">{t('exposure.colWeightPct')}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each regionsEdit as r (r.name)}
                    <tr class="border-b border-border last:border-0 hover:bg-foreground/5">
                      <td class="py-2">
                        <!-- nowrap only once the panel truly reaches max-w-6xl
                             (viewport ≥ 1184px = 1152 + 2×mx-4); below that the
                             two-column table is too narrow for nowrap without
                             overflowing into the donut, so wrapping is allowed. -->
                        <span class="flex items-center gap-2 min-[1184px]:whitespace-nowrap">
                          <span
                            class="inline-block h-3 w-3 shrink-0 rounded"
                            style="background-color: {colorForRow(r, regionsEdit, palette)};"
                          ></span>
                          {r.name}
                        </span>
                      </td>
                      <td class="py-2 text-right">
                        <input
                          type="number"
                          min="0"
                          max="100"
                          step="0.01"
                          inputmode="decimal"
                          value={r.weight}
                          aria-label={t('exposure.weightAria', { name: r.name })}
                          oninput={(e) => {
                            r.weight = e.currentTarget.value
                            onRegionsDirty()
                          }}
                          class="no-spinner focus-ring w-16 shrink-0 rounded-control border border-input bg-surface px-2 py-1.5 text-right text-sm tabular-nums"
                        />
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
            <div class="w-48 shrink-0 md:w-56">
              <!-- Muted live preview next to the weight grid: the grid below
                   already lists (and edits) the same rows, so the K.5b table
                   toggle would only duplicate it. -->
              <ExposurePie
                data={regionsEdit}
                title={t('exposure.geoTitle')}
                mute
                complete={false}
                showTableToggle={false}
              />
            </div>
          </div>

          <!-- Footer rows (identical in both boxes so they line up):
               separator, fixed-height total, reserved 3-line message area,
               fixed-height Save area. -->
          <div class="mt-3 border-t border-border" aria-hidden="true"></div>

          <div class="mt-3 h-8">
            <div class="flex items-center justify-between">
              <span class="text-sm font-medium text-foreground">{t('common.colTotal')}</span>
              <span
                class="text-sm font-semibold tabular-nums {totalColorClass(sumRegions, regionsOver)}"
              >{sumRegions.toFixed(2)}%</span>
            </div>
            <div class="mt-1.5 h-1.5 overflow-hidden rounded-full bg-input" aria-hidden="true">
              <div
                class="h-full rounded-full transition-[width] duration-200 motion-reduce:transition-none {regionsOver
                  ? 'bg-negative'
                  : 'bg-accent'}"
                style="width: {Math.min(sumRegions, 100).toFixed(2)}%;"
              ></div>
            </div>
          </div>

          <div class="h-[3.75rem] overflow-hidden pt-1 text-xs leading-5">
            {#if regionsOver}
              <p role="alert" class="text-negative">
                {t('exposure.overSum', { pct: sumRegions.toFixed(2) })}
              </p>
            {:else if sumRegions < 99.5}
              <p class="flex items-center gap-1.5 text-muted-foreground">
                <Info class="h-3.5 w-3.5 shrink-0" />
                {t('exposure.residualRegions', { pct: (100 - sumRegions).toFixed(2) })}
              </p>
            {/if}
          </div>

          <div class="flex h-12 items-center justify-end">
            <button
              onclick={saveRegions}
              disabled={savingRegions || regionsOver}
              title={regionsOver ? t('exposure.over100Title') : undefined}
              class="flex items-center gap-2 rounded-control bg-accent px-4 py-2 text-sm text-accent-foreground hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              {#if savingRegions}
                <Loader2 class="h-4 w-4 animate-spin" />
              {/if}
              {savingRegions ? t('common.saving') : t('common.save')}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}
