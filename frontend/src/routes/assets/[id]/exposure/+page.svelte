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
   * untouched in the shell/modals. Per asset type: equity/real-estate
   * ETFs/mutual funds keep the editable cards; STOCKS get the read-only
   * geographic card derived from the Data-tab `country` (editable sectors
   * only — the geography changes by editing that field, also filled by the
   * Yahoo refresh); bonds, certificates and cash keep the read-only card
   * derived from the issuer country (EPIC J.4, no sector card); anything
   * else outside the equity universe keeps the hint banner.
   */
  const ctx = getAssetPage()

  // Resolved chart palette: keeps the country bars and the geo/sector legend
  // swatches in sync with the donut colors across theme flips.
  const palette = $derived(resolvePalette(resolved()))

  // The EDITABLE distribution applies to ETFs/mutual funds of the equity or
  // real-estate class. Stocks are excluded: their geography is read-only,
  // derived from the Data-tab country (see below), while sectors stay editable.
  const exposureApplicable = $derived.by(() => {
    const a = ctx.asset
    if (!a) return false
    return (
      (a.type === 'etf' || a.type === 'mutual_fund') &&
      (a.asset_class === 'equity' || a.asset_class === 'real_estate')
    )
  })

  const stock = $derived(ctx.asset?.type === 'stock')

  // Bonds, certificates and cash: their geography feeds the allocation from
  // the issuer country set in the Data tab (EPIC J.4), not from these cards.
  const fixedIncome = $derived(
    ctx.asset != null && ['bond', 'certificate', 'cash'].includes(ctx.asset.type),
  )

  // Read-only derived geography (stocks and fixed income): a single country
  // at 100% plus the macro-regions the SERVER maps it to — the stock country
  // (Data tab / Yahoo) or the bond issuer country. The regions are a
  // non-persisted preview through POST /assets/{id}/exposure/derive, fetched
  // once per id+country; a failure stays silent and the card degrades to the
  // country row only (no region rows, no toast).
  const country = $derived(ctx.asset?.country?.trim() ?? '')
  const issuerCountry = $derived(ctx.asset?.issuer_country?.trim() ?? '')
  const derivedCountry = $derived(
    stock ? country : fixedIncome ? issuerCountry : '',
  )
  const readOnlyGeo = $derived(derivedCountry !== '')
  let roRegions = $state<ExposureRow[]>([])
  // Key of the request currently owned by the effect (`''` when none): keeps
  // unrelated re-runs from firing duplicate calls and discards in-flight
  // responses that went stale when the asset or derived country changed.
  // Plain (non-reactive) variable: reading or writing it never re-triggers
  // the effect itself.
  let roRequestKey = ''

  $effect(() => {
    const id = ctx.id
    if (!readOnlyGeo || !id) {
      roRequestKey = ''
      roRegions = []
      return
    }
    const key = `${id}:${derivedCountry}`
    if (roRequestKey === key) return
    roRequestKey = key
    assetApi
      .deriveRegions(id, [{ name: derivedCountry, weight: '100' }])
      .then((res) => {
        // Drop the response if a newer key took over meanwhile.
        if (roRequestKey === key) roRegions = res.regions.filter((r) => Number(r.weight) > 0)
      })
      .catch(() => {
        if (roRequestKey === key) roRegions = []
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
{:else if ctx.asset && !exposureApplicable}
  {#if readOnlyGeo}
    <!-- Read-only geographic card for stocks and fixed income: the Data-tab
         country (stocks, also filled by the Yahoo refresh) or issuer country
         shown as the single 100% country row, plus the macro-regions the
         backend derives from it (non-persisted preview). No edit affordances
         by design — the values change with that field. -->
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="font-semibold">{t('exposure.geoTitle')}</h2>
      <p class="mb-4 mt-1 text-sm text-muted-foreground">
        {stock ? t('exposure.stockDerivedNote') : t('exposure.issuerDerivedNote')}
      </p>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="rounded-card border border-border bg-muted p-4">
          <h3 class="mb-2 font-medium">{t('exposure.countries')}</h3>
          <div class="space-y-2 py-1">
            <div class="flex items-center gap-2 text-xs">
              <span
                class="w-28 shrink-0 truncate sm:w-36"
                title={derivedCountry + ' — ' + countryDisplayName(derivedCountry)}
              >{countryDisplayName(derivedCountry)} ({derivedCountry})</span>
              <div class="h-2.5 min-w-0 flex-1 overflow-hidden rounded-full bg-input">
                <div class="h-full w-full rounded-full" style="background-color: {palette[0]};"></div>
              </div>
              <span class="w-14 shrink-0 text-right text-muted-foreground tabular-nums">{formatPercent(100)}</span>
            </div>
          </div>
        </div>
        <div class="rounded-card border border-border bg-muted p-4">
          <h3 class="mb-2 font-medium">{t('exposure.regions')}</h3>
          {#if roRegions.length === 0}
            <!-- Empty while the derive preview is in flight or failed: a muted
                 placeholder instead of an empty panel, no spinner or error. -->
            <span class="py-1 text-xs text-muted-foreground">—</span>
          {:else}
            <div class="space-y-2 py-1">
              {#each roRegions as r, i (r.name)}
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
      {#if fixedIncome}
        <p class="mt-4 text-sm text-muted-foreground">{t('exposure.universeIssuerHint')}</p>
      {/if}
    </div>
  {:else if stock}
    <!-- Stock without a country: the derived geography is unavailable until
         the Data-tab country is set. Sectors below stay editable anyway. -->
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="mb-2 font-semibold">{t('exposure.geoTitle')}</h2>
      <p class="text-sm text-muted-foreground">{t('exposure.stockCountryHint')}</p>
    </div>
  {:else if fixedIncome}
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="mb-2 font-semibold">{t('exposure.universeTitle')}</h2>
      <p class="text-sm text-muted-foreground">
        {t('exposure.universeIssuerHint')}
      </p>
    </div>
  {:else}
    <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
      <h2 class="mb-2 font-semibold">{t('exposure.universeTitle')}</h2>
      <p class="text-sm text-muted-foreground">
        {t('exposure.universeHint')}
      </p>
      <p class="mt-2 text-sm text-muted-foreground">
        {t('exposure.universeClassHint')}
      </p>
    </div>
  {/if}
{/if}

{#if ctx.asset && ctx.exposure && (stock || exposureApplicable)}
  <!-- Sector distribution card: editable for the equity universe and for
       stocks (whose geography is read-only while sectors stay editable). -->
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
{/if}
