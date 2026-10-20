<script lang="ts">
  import { Info, Loader2, X } from 'lucide-svelte'
  import type { ExposureRow } from '#lib/services/api.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import ExposurePie from './ExposurePie.svelte'
  import ProvenanceBadge from './ProvenanceBadge.svelte'
  import { colorForRow, resolvePalette } from '#lib/chartPalette.js'
  import { resolved } from '#lib/stores/theme.svelte.js'
  import { totalColorClass } from '#lib/ui-colors.js'

  let {
    open = $bindable(false),
    onClose,
    sectorsEdit = $bindable([] as ExposureRow[]),
    sumSectors = 0,
    sectorsValid = false,
    savingSectors = false,
    saveSectors,
    prefilling = false,
    fetchingETF = false,
    fetchingMorningstar = false,
    prefillSectorsFromETF,
    prefillSectorsFromYahoo,
    prefillSectorsFromMorningstar,
    sectorsSource = null as string | null,
    sectorsUpdatedAt = null as string | null,
    onSectorsDirty,
    assetType = 'stock',
  }: {
    open: boolean
    onClose: () => void
    sectorsEdit: ExposureRow[]
    sumSectors: number
    sectorsValid: boolean
    savingSectors: boolean
    saveSectors: () => void
    prefilling: boolean
    fetchingETF: boolean
    fetchingMorningstar: boolean
    prefillSectorsFromETF: () => void
    prefillSectorsFromYahoo: () => void
    prefillSectorsFromMorningstar: () => void
    sectorsSource: string | null
    sectorsUpdatedAt: string | null
    onSectorsDirty: () => void
    assetType: string
  } = $props()

  // Resolved chart palette: keeps the table swatches in sync with the donut
  // colors and re-evaluates on theme flips.
  const palette = $derived(resolvePalette(resolved()))

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
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-overlay/50"
    onclick={handleBackdropClick}
    onkeydown={(e) => e.key === 'Escape' && onClose()}
    role="dialog"
    aria-modal="true"
    aria-label={t('exposure.editSectors')}
    tabindex="-1"
  >
    <div
      class="relative mx-4 max-h-[90vh] w-full max-w-3xl overflow-y-auto rounded-card border-border bg-surface p-6 shadow-raised"
    >
      <div class="mb-6 flex items-center justify-between">
        <h2 class="text-lg font-semibold">{t('exposure.editSectors')}</h2>
        <button
          onclick={onClose}
          class="rounded-control p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label={t('common.close')}
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <!-- Sectors -->
      <div class="flex flex-col rounded-card border border-border bg-muted p-4">
        <div class="mb-3 flex items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <h3 class="font-medium">{t('exposure.sectorTitle')}</h3>
            <ProvenanceBadge source={sectorsSource} updatedAt={sectorsUpdatedAt} />
          </div>
          <div class="flex items-center gap-1.5">
            <button
              onclick={prefillSectorsFromETF}
              disabled={fetchingETF || assetType !== 'etf'}
              title={t('exposure.prefillJustEtf')}
              aria-label={t('exposure.prefillSectorsJustEtf')}
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
              onclick={prefillSectorsFromYahoo}
              disabled={prefilling}
              title={t('exposure.prefillYahoo')}
              aria-label={t('exposure.prefillSectorsYahoo')}
              class="rounded-control border border-input bg-surface p-1.5 shadow-card hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
            >
              {#if prefilling}
                <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
              {:else}
                <img
                  class="h-5 w-5 rounded"
                  src="https://www.google.com/s2/favicons?domain=finance.yahoo.com&sz=32"
                  alt="Yahoo"
                />
              {/if}
            </button>
            <button
              onclick={prefillSectorsFromMorningstar}
              disabled={fetchingMorningstar || assetType !== 'etf'}
              title={t('exposure.prefillMorningstar')}
              aria-label={t('exposure.prefillSectorsMorningstar')}
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
        <div class="flex flex-1 flex-col gap-4 md:flex-row">
          <div class="flex-1">
            <table class="w-full text-left text-sm">
              <thead>
                <tr class="border-b border-border text-muted-foreground">
                  <th class="pb-2">{t('exposure.gicsSector')}</th>
                  <th class="pb-2 text-right">{t('exposure.colWeightPct')}</th>
                </tr>
              </thead>
              <tbody>
                {#each sectorsEdit as s (s.name)}
                  <tr class="border-b border-border last:border-0">
                    <td class="py-2">
                      <span class="flex items-center gap-2">
                        <span
                          class="inline-block h-3 w-3 shrink-0 rounded"
                          style="background-color: {colorForRow(s, sectorsEdit, palette)};"
                        ></span>
                        {s.name}
                      </span>
                    </td>
                    <td class="py-2 text-right">
                      <input
                        type="number"
                        min="0"
                        max="100"
                        step="0.01"
                        value={s.weight}
                        oninput={(e) => {
                          s.weight = e.currentTarget.value
                          onSectorsDirty()
                        }}
                        class="no-spinner w-24 rounded-control border border-input px-3 py-1.5 text-right text-sm tabular-nums"
                      />
                    </td>
                  </tr>
                {/each}
                <tr class="border-t border-border font-semibold">
                  <td class="py-2">{t('common.colTotal')}</td>
                  <td class="py-2 text-right tabular-nums {totalColorClass(sumSectors, !sectorsValid)}">
                    {sumSectors.toFixed(2)}%
                  </td>
                </tr>
              </tbody>
            </table>
            <!-- Same rule as regions: only a sum above 100 blocks the save;
                 below 100 the backend folds the residual into «Other / Not
                 Classified» at persist time. -->
            {#if !sectorsValid}
              <p role="alert" class="mt-2 text-sm text-negative">
                {t('exposure.overSum', { pct: sumSectors.toFixed(2) })}
              </p>
            {:else if sumSectors < 99.5}
              <p class="mt-2 flex items-center gap-1.5 text-sm text-muted-foreground">
                <Info class="h-3.5 w-3.5 shrink-0" />
                {t('exposure.residualRegions', { pct: (100 - sumSectors).toFixed(2) })}
              </p>
            {/if}
          </div>
          <div class="w-48 shrink-0 md:w-56">
            <!-- Muted live preview next to the weight grid: the grid already
                 lists (and edits) the same rows, so the K.5b table toggle
                 would only duplicate it. Open donut (complete={false}): the
                 residual row is filtered out of the edit list, the gap shows
                 what the backend will fold into «Other / Not Classified». -->
            <ExposurePie
              data={sectorsEdit}
              title={t('exposure.sectorTitle')}
              mute
              complete={false}
              showTableToggle={false}
            />
          </div>
        </div>
        <div class="mt-auto flex justify-end pt-4">
          <button
            onclick={saveSectors}
            disabled={!sectorsValid || savingSectors}
            title={!sectorsValid ? t('exposure.over100Title') : undefined}
            class="rounded-control bg-accent px-4 py-2 text-sm text-accent-foreground hover:bg-accent-hover disabled:opacity-50"
          >
            {savingSectors ? t('common.saving') : t('common.save')}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
