<script lang="ts">
  import { formatPercent } from '#lib/format.js'
  import { resolvePalette } from '#lib/chartPalette.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import { resolved } from '#lib/stores/theme.svelte.js'
  import { countryDisplayName } from '#lib/countryNames.js'
  import { assetApi, type ExposureRow } from '#lib/services/api.js'
  import ExposurePie from '#lib/components/ExposurePie.svelte'
  import { Pencil } from 'lucide-svelte'
  import { getAssetPage } from '../context'
  import { withoutOther } from '../exposure-utils'

  /**
   * Exposure tab (EPIC K.4b, spec §6.3): the geographic distribution card
   * (countries bar list + regions open-donut) and the sector donut card
   * exactly as the old single page rendered them — always the STORED
   * exposure, never the modals' working copy. "Modifica" buttons hand over
   * to the layout-owned `openGeoModal`/`openSectorModal`, which re-hydrate
   * the edit lists and provenance badges from the saved exposure and open
   * the (layout-mounted) `ExposureGeoModal`/`ExposureSectorModal`: the whole
   * JustETF/Morningstar/Yahoo prefill, derive-regions and per-dimension save
   * machinery — provenance badges and sum validation included — lives
   * untouched in the shell/modals. Assets outside the equity universe keep
   * the hint banner instead of the editable cards: the equity-class rule for
   * most, and — for bonds, certificates and cash carrying an issuer country
   * (EPIC J.4) — a read-only geographic card derived from the Data-tab
   * issuer country (100% single country + server-derived regions), with no
   * edit affordance: the geography changes by editing that field in the
   * Data tab.
   */
  const ctx = getAssetPage()

  // Resolved chart palette: keeps the country bars and the geo/sector legend
  // swatches in sync with the donut colors across theme flips.
  const palette = $derived(resolvePalette(resolved()))

  // The distribution applies to the equity universe: stocks, and equity or
  // real-estate ETFs/mutual funds (same rule the old page gated on).
  const exposureApplicable = $derived.by(() => {
    const a = ctx.asset
    if (!a) return false
    return (
      a.type === 'stock' ||
      ((a.type === 'etf' || a.type === 'mutual_fund') &&
        (a.asset_class === 'equity' || a.asset_class === 'real_estate'))
    )
  })

  // Bonds, certificates and cash: their geography feeds the allocation from
  // the issuer country set in the Data tab (EPIC J.4), not from these cards.
  const fixedIncome = $derived(
    ctx.asset != null && ['bond', 'certificate', 'cash'].includes(ctx.asset.type),
  )

  // Read-only fixed-income geography: a single issuer country at 100% plus
  // the macro-regions the SERVER maps it to. The regions are a non-persisted
  // preview through POST /assets/{id}/exposure/derive, fetched once per
  // id+country; a failure stays silent and the card degrades to the country
  // row only (no region rows, no toast).
  const issuerCountry = $derived(ctx.asset?.issuer_country?.trim() ?? '')
  const fiReadOnlyGeo = $derived(fixedIncome && issuerCountry !== '')
  let fiRegions = $state<ExposureRow[]>([])
  // Key of the request currently owned by the effect (`''` when none): keeps
  // unrelated re-runs from firing duplicate calls and discards in-flight
  // responses that went stale when the asset or issuer country changed.
  // Plain (non-reactive) variable: reading or writing it never re-triggers
  // the effect itself.
  let fiRequestKey = ''

  $effect(() => {
    const id = ctx.id
    if (!fiReadOnlyGeo || !id) {
      fiRequestKey = ''
      fiRegions = []
      return
    }
    const key = `${id}:${issuerCountry}`
    if (fiRequestKey === key) return
    fiRequestKey = key
    assetApi
      .deriveRegions(id, [{ name: issuerCountry, weight: '100' }])
      .then((res) => {
        // Drop the response if a newer key took over meanwhile.
        if (fiRequestKey === key) fiRegions = res.regions.filter((r) => Number(r.weight) > 0)
      })
      .catch(() => {
        if (fiRequestKey === key) fiRegions = []
      })
  })

  // ---------------------------------------------------------------------------
  // Display vs edit split (kept from the old page): the cards always render
  // the STORED exposure (the layout's `exposure` state, refreshed by every
  // successful save). The edit lists are the modals' working copy and may
  // hold unsaved manual edits or provider prefill previews while a modal is
  // open; they never leak into these cards.
  // ---------------------------------------------------------------------------
  const displayCountries = $derived(
    (ctx.exposure?.countries ?? []).filter((c) => Number(c.weight) > 0),
  )
  // Stored regions without the «Other / Not Classified» residual (same filter
  // the edit list applies at assignment time, here re-used for display).
  const displayRegions = $derived(ctx.exposure ? withoutOther(ctx.exposure.regions) : [])
  // Sectors behave exactly like regions: the stored «Other / Not Classified»
  // residual is filtered out of the display too.
  const displaySectors = $derived(ctx.exposure ? withoutOther(ctx.exposure.sectors) : [])

  // Top 15 stored countries by weight (desc, > 0) for the geographic card bar
  // list. The copy-then-sort keeps the derived displayCountries array pristine.
  const topCountries = $derived.by(() =>
    [...displayCountries]
      .sort((a, b) => Number(b.weight) - Number(a.weight))
      .slice(0, 15),
  )
  // Largest visible weight: bars are scaled proportionally against it.
  const maxCountryWeight = $derived(
    topCountries.reduce((max, c) => Math.max(max, Number(c.weight) || 0), 0),
  )
</script>

{#if ctx.asset && exposureApplicable && ctx.exposure}
  <!-- Geographic distribution card (stored exposure): countries bar list +
       regions pie -->
  <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h2 class="font-semibold">{t('exposure.geoTitle')}</h2>
      <button
        onclick={ctx.openGeoModal}
        aria-label={t('exposure.editGeo')}
        class="flex items-center gap-2 rounded-control border border-border px-3 py-1.5 text-sm text-foreground hover:bg-muted"
      >
        <Pencil class="h-4 w-4" />
        {t('exposure.modify')}
      </button>
    </div>
    <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
      <div class="rounded-card border border-border bg-muted p-4">
        <h3 class="mb-2 font-medium">{t('exposure.countries')}</h3>
        {#if topCountries.length === 0}
          <div class="flex h-[240px] w-full items-center justify-center text-sm text-muted-foreground">
            {t('chartView.noDistribution')}
          </div>
        {:else}
          <div class="space-y-2 py-1">
            {#each topCountries as c, i (c.name)}
              {@const weight = Number(c.weight) || 0}
              {@const barPct = maxCountryWeight > 0 ? (weight / maxCountryWeight) * 100 : 0}
              <div class="flex items-center gap-2 text-xs">
                <span
                  class="w-28 shrink-0 truncate sm:w-36"
                  title={c.name + ' — ' + countryDisplayName(c.name)}
                >{countryDisplayName(c.name)}</span>
                <div class="h-2.5 min-w-0 flex-1 overflow-hidden rounded-full bg-input">
                  <div
                    class="h-full rounded-full"
                    style="width: {barPct.toFixed(1)}%; background-color: {palette[i % palette.length]};"
                  ></div>
                </div>
                <span class="w-14 shrink-0 text-right text-muted-foreground tabular-nums">{formatPercent(weight)}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
      <div class="rounded-card border border-border bg-muted p-4">
        <h3 class="mb-2 font-medium">{t('exposure.regions')}</h3>
        <!-- displayRegions never carries the «Other / Not Classified» row
             (withoutOther filters it out of the stored exposure), so the
             donut renders open: complete={false} adds the transparent
             residual gap. The K.5b table toggle is off: the legend under the
             chart already lists every visible row with its weight. -->
        <ExposurePie
          data={displayRegions}
          title={t('exposure.geoTitle')}
          complete={false}
          showTableToggle={false}
        />
        <div class="mt-3 grid grid-cols-2 gap-x-3 gap-y-1">
          {#each displayRegions.filter((r) => Number(r.weight) > 0) as r, i (r.name)}
            <div class="flex items-center gap-1.5 text-xs">
              <span
                class="inline-block h-2.5 w-2.5 shrink-0 rounded-sm"
                style="background-color: {palette[i % palette.length]};"
              ></span>
              <span class="truncate">{r.name}</span>
              <span class="ml-auto text-muted-foreground tabular-nums">{formatPercent(Number(r.weight))}</span>
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>

  <!-- Sector distribution card -->
  <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h2 class="font-semibold">{t('exposure.sectorTitle')}</h2>
      <button
        onclick={ctx.openSectorModal}
        aria-label={t('exposure.editSectors')}
        class="flex items-center gap-2 rounded-control border border-border px-3 py-1.5 text-sm text-foreground hover:bg-muted"
      >
        <Pencil class="h-4 w-4" />
        {t('exposure.modify')}
      </button>
    </div>
    <div class="rounded-card border border-border bg-muted p-4">
      <h3 class="mb-2 font-medium">{t('exposure.sectors')}</h3>
      <!-- Same as regions: the residual row is filtered out of displaySectors,
           so the donut renders open (complete={false} adds the transparent
           gap) and the legend below already lists every sector with its
           weight — no K.5b table toggle here. -->
      <ExposurePie
        data={displaySectors}
        title={t('exposure.sectorTitle')}
        complete={false}
        showTableToggle={false}
      />
      <div class="mt-3 grid grid-cols-2 gap-x-3 gap-y-1">
        {#each displaySectors.filter((r) => Number(r.weight) > 0) as s, i (s.name)}
          <div class="flex items-center gap-1.5 text-xs">
            <span
              class="inline-block h-2.5 w-2.5 shrink-0 rounded-sm"
              style="background-color: {palette[i % palette.length]};"
            ></span>
            <span class="truncate">{s.name}</span>
            <span class="ml-auto text-muted-foreground tabular-nums">{formatPercent(Number(s.weight))}</span>
          </div>
        {/each}
      </div>
    </div>
  </div>
{:else if ctx.asset && !exposureApplicable}
  {#if fiReadOnlyGeo}
    <!-- Read-only geographic card for fixed income (EPIC J.4): the Data-tab
         issuer country shown as the single 100% country row, plus the macro-
         regions the backend derives from it (non-persisted preview). No edit
         affordances by design — the values change with the issuer country. -->
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="font-semibold">{t('exposure.geoTitle')}</h2>
      <p class="mb-4 mt-1 text-sm text-muted-foreground">{t('exposure.issuerDerivedNote')}</p>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="rounded-card border border-border bg-muted p-4">
          <h3 class="mb-2 font-medium">{t('exposure.countries')}</h3>
          <div class="space-y-2 py-1">
            <div class="flex items-center gap-2 text-xs">
              <span
                class="w-28 shrink-0 truncate sm:w-36"
                title={issuerCountry + ' — ' + countryDisplayName(issuerCountry)}
              >{countryDisplayName(issuerCountry)} ({issuerCountry})</span>
              <div class="h-2.5 min-w-0 flex-1 overflow-hidden rounded-full bg-input">
                <div class="h-full w-full rounded-full" style="background-color: {palette[0]};"></div>
              </div>
              <span class="w-14 shrink-0 text-right text-muted-foreground tabular-nums">{formatPercent(100)}</span>
            </div>
          </div>
        </div>
        <div class="rounded-card border border-border bg-muted p-4">
          <h3 class="mb-2 font-medium">{t('exposure.regions')}</h3>
          {#if fiRegions.length === 0}
            <!-- Empty while the derive preview is in flight or failed: a muted
                 placeholder instead of an empty panel, no spinner or error. -->
            <span class="py-1 text-xs text-muted-foreground">—</span>
          {:else}
            <div class="space-y-2 py-1">
              {#each fiRegions as r, i (r.name)}
                <div class="flex items-center gap-2 text-xs">
                  <span class="w-28 shrink-0 truncate sm:w-36" title={r.name}>{r.name}</span>
                  <div class="h-2.5 min-w-0 flex-1 overflow-hidden rounded-full bg-input">
                    <div
                      class="h-full rounded-full"
                      style="width: {Number(r.weight)}%; background-color: {palette[i % palette.length]};"
                    ></div>
                  </div>
                  <span class="w-14 shrink-0 text-right text-muted-foreground tabular-nums">{formatPercent(Number(r.weight))}</span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
      <p class="mt-4 text-sm text-muted-foreground">{t('exposure.universeIssuerHint')}</p>
    </div>
  {:else}
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="mb-2 font-semibold">{t('exposure.universeTitle')}</h2>
      {#if fixedIncome}
        <p class="text-sm text-muted-foreground">
          {t('exposure.universeIssuerHint')}
        </p>
      {:else}
        <p class="text-sm text-muted-foreground">
          {t('exposure.universeHint')}
        </p>
        {#if ctx.asset.type !== 'stock'}
          <p class="mt-2 text-sm text-muted-foreground">
            {t('exposure.universeClassHint')}
          </p>
        {/if}
      {/if}
    </div>
  {/if}
{/if}
