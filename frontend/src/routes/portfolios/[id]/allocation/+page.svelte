<script lang="ts">
  import ExposureBarChart, { type ExposureBarRow } from '#lib/components/domain/ExposureBarChart.svelte'
  import ClassDonut from '#lib/components/domain/ClassDonut.svelte'
  import AllocationDrillPanel from '#lib/components/domain/AllocationDrillPanel.svelte'
  import { countryDisplayName } from '#lib/countryNames.js'
  import { chartSemanticColors } from '#lib/chartPalette.js'
  import { assetClassLabel } from '#lib/format.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import { resolved } from '#lib/stores/theme.svelte.js'
  import {
    portfolioApi,
    type AllocationDrill,
    type AllocationDrillDim,
  } from '#lib/services/api.js'
  import { getPortfolioPage } from '../context'

  /**
   * Allocation tab (EPIC K.4a): the EPIC I.7 (#86) section promoted to its
   * own deep-linkable route — the asset-class donut plus the region, sector
   * and country bars, mirroring the dashboard's
   * "Allocazione complessiva" card, all in the portfolio currency. The
   * three payloads (and their per-endpoint isolated error flags) come from
   * the shell context; a failed allocation call still only shows its own
   * "non disponibile" panel without blocking the rest of the tab.
   */
  const ctx = getPortfolioPage()
  const currency = $derived(ctx.currency)

  // EPIC I.7 (#86): the region/sector/country payloads are mapped onto the
  // generic ExposureBarRow shape consumed by ExposureBarChart (countries
  // keep their raw ISO codes as row identity; the chart maps them to full
  // names via `labelFor`, see the country panel below).
  const regionBarRows = $derived<ExposureBarRow[]>(
    (ctx.geoAlloc?.regions ?? []).map((r) => ({ name: r.region, value: r.value, weight: r.weight })),
  )
  const sectorBarRows = $derived<ExposureBarRow[]>(
    (ctx.sectorAlloc?.sectors ?? []).map((s) => ({ name: s.sector, value: s.value, weight: s.weight })),
  )
  const countryBarRows = $derived<ExposureBarRow[]>(
    (ctx.geoAlloc?.countries ?? []).map((c) => ({ name: c.country, value: c.value, weight: c.weight })),
  )

  // Universe coverage notes for the bar panels, like the dashboard: shown
  // only when holdings were actually excluded. Geography covers the equity
  // universe plus fixed income carrying an issuer country (EPIC J.4);
  // sectors stay equity-only. Geography and sectors come from two isolated
  // endpoints, so each payload carries its own covered/excluded note.
  function universeNote(
    covered: string | undefined,
    excluded: string | undefined,
    key: 'allocation.geoUniverse' | 'allocation.equityUniverse',
  ): string | undefined {
    const coveredNum = Number(covered || 0)
    const excludedNum = Number(excluded || 0)
    const total = coveredNum + excludedNum
    if (total <= 0 || excludedNum <= 0) return undefined
    return t(key, { pct: ((coveredNum / total) * 100).toFixed(1) })
  }
  const geoUniverseNote = $derived(
    universeNote(ctx.geoAlloc?.covered_value, ctx.geoAlloc?.excluded_value, 'allocation.geoUniverse'),
  )
  const sectorUniverseNote = $derived(
    universeNote(ctx.sectorAlloc?.covered_value, ctx.sectorAlloc?.excluded_value, 'allocation.equityUniverse'),
  )

  // Aggregated "Other" buckets are muted grey, like the slice treatment in
  // the donut charts (same helper as the dashboard card). Read via
  // chartSemanticColors so it re-evaluates on theme flips (the {#key} blocks
  // inside the charts re-init them anyway).
  function otherGrey(name: string): string | undefined {
    return name === 'Other' || name === 'Other / Not Classified'
      ? chartSemanticColors(resolved()).other
      : undefined
  }

  // ── Allocation drill-down (EPIC K.5, spec §6.5) ─────────────────────────
  // One shared panel for the whole tab, like the dashboard card: clicking a
  // class slice or a sector/region/country bar sets the bucket and opens it
  // (drawer ≥ lg / sheet < lg, D4). The `key` is the RAW value the chart
  // carries (class key / ISO code / region / sector name — may contain
  // spaces); the `title` is the label the same chart displays. Portfolio
  // scope: `allocationDrill(id, dim, key)` in the portfolio currency.
  let drillOpen = $state(false)
  let drillDim = $state<AllocationDrillDim>('class')
  let drillKey = $state('')
  let drillTitle = $state('')

  function openDrill(dim: AllocationDrillDim, key: string, label = key): void {
    drillDim = dim
    drillKey = key
    drillTitle = label
    drillOpen = true
  }
  function closeDrill(): void {
    drillOpen = false
  }
  // Stable per page instance (Svelte 5 script bodies run once): the panel's
  // fetch effect depends on the identity, not on the bucket values. The id is
  // read at call time (the shell always mounts this tab under a portfolio).
  function drillFetch(dim: AllocationDrillDim, key: string): Promise<AllocationDrill> {
    if (!ctx.id) return Promise.reject(new Error(t('drill.error')))
    return portfolioApi.allocationDrill(ctx.id, dim, key)
  }
</script>

<div class="grid gap-4 lg:grid-cols-2">
  <div class="rounded-card border-border bg-surface p-4 shadow-card">
    {#if ctx.classAllocError}
      <h3 class="mb-3 font-semibold">{t('allocation.assetClasses')}</h3>
      <p class="text-sm text-muted-foreground">{t('allocation.classUnavailable')}</p>
    {:else}
      <ClassDonut
        data={ctx.classAlloc?.classes ?? []}
        currency={ctx.classAlloc?.currency || currency}
        label={t('allocation.assetClasses')}
        onDrill={(cls) => openDrill('class', cls, assetClassLabel(cls))}
      />
    {/if}
  </div>
  <div class="rounded-card border-border bg-surface p-4 shadow-card">
    {#if ctx.sectorAllocError}
      <h3 class="mb-1 font-semibold">{t('allocation.sectorsEquity')}</h3>
      <p class="text-sm text-muted-foreground">{t('allocation.sectorUnavailable')}</p>
    {:else}
      <ExposureBarChart
        rows={sectorBarRows}
        currency={ctx.sectorAlloc?.currency || currency}
        label={t('allocation.sectorsEquity')}
        note={sectorUniverseNote}
        colorFor={otherGrey}
        onDrill={(name) => openDrill('sector', name)}
      />
    {/if}
  </div>
  <div class="rounded-card border-border bg-surface p-4 shadow-card">
    {#if ctx.geoAllocError}
      <h3 class="mb-1 font-semibold">{t('allocation.regions')}</h3>
      <p class="text-sm text-muted-foreground">{t('allocation.geoUnavailable')}</p>
    {:else}
      <ExposureBarChart
        rows={regionBarRows}
        currency={ctx.geoAlloc?.currency || currency}
        label={t('allocation.regions')}
        note={geoUniverseNote}
        colorFor={otherGrey}
        onDrill={(name) => openDrill('region', name)}
      />
    {/if}
  </div>
  <div class="rounded-card border-border bg-surface p-4 shadow-card">
    {#if ctx.geoAllocError}
      <h3 class="mb-1 font-semibold">{t('allocation.countries')}</h3>
      <p class="text-sm text-muted-foreground">{t('allocation.geoUnavailable')}</p>
    {:else}
      <ExposureBarChart
        rows={countryBarRows}
        currency={ctx.geoAlloc?.currency || currency}
        label={t('allocation.countries')}
        note={geoUniverseNote}
        colorFor={otherGrey}
        labelFor={countryDisplayName}
        maxVisibleRows={10}
        onDrill={(code) => openDrill('country', code, countryDisplayName(code))}
      />
    {/if}
  </div>
</div>
<!-- One drill panel per page (D4): the clicked chart fills the bucket and
     the fetcher runs the portfolio-scope request in the portfolio currency. -->
<AllocationDrillPanel
  open={drillOpen}
  onClose={closeDrill}
  title={drillTitle}
  dim={drillDim}
  key={drillKey}
  fetcher={drillFetch}
  currencyHint={currency}
/>
