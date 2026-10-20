<script lang="ts">
  import { t } from '$lib/i18n/index.svelte'
  import { ASSET_TYPES, ASSET_CLASSES, assetTypeLabel, assetClassLabel } from '$lib/format'
  import { History, RefreshCw, Trash2 } from 'lucide-svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
  import ManualPricesCard from '$lib/components/domain/ManualPricesCard.svelte'
  import { getAssetPage } from '../context'

  /**
   * Data tab (EPIC K.4b, spec §6.3): the "curate the data" job — the old
   * "Caratteristiche" card as an editable metadata form (same fields, same
   * dirty-save: `hasChanges` enables "Salva modifiche", the PATCH lives in
   * the layout and prefills still sync `form.isin` from here), the
   * `price_source` selector, the danger zone (Yahoo meta refresh / full
   * history backfill / delete asset — the same layout-owned actions as the
   * header `⋯` menu, sharing their busy spinners), the EPIC J.1 Manual
   * prices section (live only for `manual`/`none` price sources) and the
   * J.2 fixed-income slot still a muted "coming soon" placeholder with no
   * behaviour yet.
   */
  const ctx = getAssetPage()
</script>

{#if ctx.asset}
  <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h2 class="font-semibold">{t('asset.characteristics')}</h2>
      <Button onclick={ctx.saveAsset} disabled={!ctx.hasChanges || ctx.saving}>
        {ctx.saving ? t('common.saving') : t('common.saveChanges')}
      </Button>
    </div>
    <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
      <div>
        <label for="asset-ticker" class="mb-1 block text-xs font-medium text-muted-foreground">{t('positions.colTicker')}</label>
        <input
          id="asset-ticker"
          type="text"
          bind:value={ctx.form.ticker}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label for="asset-isin" class="mb-1 block text-xs font-medium text-muted-foreground">{t('asset.factIsin')}</label>
        <input
          id="asset-isin"
          type="text"
          bind:value={ctx.form.isin}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label for="asset-name" class="mb-1 block text-xs font-medium text-muted-foreground">{t('chartView.colName')}</label>
        <input
          id="asset-name"
          type="text"
          bind:value={ctx.form.name}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label for="asset-type" class="mb-1 block text-xs font-medium text-muted-foreground">{t('asset.factType')}</label>
        <select
          id="asset-type"
          bind:value={ctx.form.type}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        >
          {#each ASSET_TYPES as value (value)}
            <option value={value}>{assetTypeLabel(value)}</option>
          {/each}
        </select>
      </div>
      <div>
        <label for="asset-currency" class="mb-1 block text-xs font-medium text-muted-foreground">{t('asset.factCurrency')}</label>
        <input
          id="asset-currency"
          type="text"
          bind:value={ctx.form.currency}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label for="asset-exchange" class="mb-1 block text-xs font-medium text-muted-foreground">{t('asset.factExchange')}</label>
        <input
          id="asset-exchange"
          type="text"
          bind:value={ctx.form.exchange}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label for="asset-class" class="mb-1 block text-xs font-medium text-muted-foreground">{t('asset.factClass')}</label>
        <select
          id="asset-class"
          bind:value={ctx.form.asset_class}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        >
          {#each ASSET_CLASSES as value (value)}
            <option value={value}>{assetClassLabel(value)}</option>
          {/each}
        </select>
      </div>
      <div>
        <label for="asset-price-source" class="mb-1 block text-xs font-medium text-muted-foreground">{t('asset.factPriceSource')}</label>
        <select
          id="asset-price-source"
          bind:value={ctx.form.price_source}
          class="w-full rounded-control border border-input px-3 py-2 text-sm"
        >
          <option value="yahoo">{t('asset.priceSourceYahoo')}</option>
          <option value="manual">{t('asset.priceSourceManual')}</option>
          <option value="none">{t('asset.priceSourceNone')}</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Danger zone (spec §6.3): the three layout-owned actions the old
       "Caratteristiche" `⋮` menu carried — the only place they live now that
       the duplicated header `⋯` menu was removed (#117); the busy spinners
       and the delete confirmation are shared layout state. -->
  <div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
    <h2 class="mb-2 font-semibold">{t('asset.dangerZone')}</h2>
    <div class="flex flex-wrap items-center gap-2">
      <Button variant="secondary" disabled={ctx.refreshingMeta} onclick={ctx.refreshFromYahoo}>
        {#if ctx.refreshingMeta}
          <Spinner size="sm" />
        {:else}
          <RefreshCw class="h-4 w-4" aria-hidden="true" />
        {/if}
        {t('asset.refreshMeta')}
      </Button>
      <Button variant="secondary" disabled={ctx.backfillingHistory} onclick={ctx.backfillHistory}>
        {#if ctx.backfillingHistory}
          <Spinner size="sm" />
        {:else}
          <History class="h-4 w-4" aria-hidden="true" />
        {/if}
        {t('asset.backfillHistory')}
      </Button>
      <Button variant="danger" onclick={ctx.requestDelete}>
        <Trash2 class="h-4 w-4" aria-hidden="true" />
        {t('asset.delete')}
      </Button>
    </div>
  </div>

  <!-- EPIC J.1 (manual price entry, #105): a live section for assets without
       an automatic feed — the manual endpoints exist only for `manual`/`none`
       price sources, so the section renders only for them. -->
  {#if ctx.asset.price_source === 'manual' || ctx.asset.price_source === 'none'}
    <ManualPricesCard
      assetId={ctx.id ?? ''}
      currency={ctx.currency}
      refresh={ctx.refetchAssetData}
    />
  {/if}

  <!-- EPIC J reserved slot (J.2): placeholder only — no behaviour, no
       endpoint, nothing to ship yet; it makes the future surface
       discoverable. -->
  <div class="rounded-card border border-dashed border-border p-4">
    <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
      <h2 class="font-semibold text-muted-foreground">{t('asset.fixedIncome')}</h2>
      <Badge>{t('quickActions.comingSoon')}</Badge>
    </div>
    <p class="text-sm text-muted-foreground">{t('asset.fixedIncomeHint')}</p>
  </div>
{/if}
