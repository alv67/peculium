<script lang="ts">
  import type { Snippet } from 'svelte'
  import { onMount, onDestroy } from 'svelte'
  import { goto } from '$app/navigation'
  import { resolve } from '$app/paths'
  import { page } from '$app/state'
  import { toast } from '$lib/stores/toast.svelte'
  import { priceRefresh } from '$lib/stores/priceRefresh.svelte'
  import { t } from '$lib/i18n/index.svelte'
  import type { MessageKey } from '$lib/i18n/index.svelte'
  import {
    assetApi,
    clearGetCache,
    jobsApi,
    portfolioApi,
    pricesApi,
    type Asset,
    type AssetExposure,
    type AssetPatch,
    type AssetQuote,
    type ExposureRow,
    type Price,
    type SplitInfo,
  } from '$lib/services/api'
  import { formatCurrency, assetTypeLabel, assetClassLabel, priceSourceLabel } from '$lib/format'
  import { ArrowLeft } from 'lucide-svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte'
  import PnlValue from '$lib/components/ui/PnlValue.svelte'
  import Tabs from '$lib/components/ui/Tabs.svelte'
  import ExposureGeoModal from '$lib/components/ExposureGeoModal.svelte'
  import ExposureSectorModal from '$lib/components/ExposureSectorModal.svelte'
  import { setAssetPage, type AssetForm, type AssetPageContext, type HeldRow } from './context'
  import { positiveCountries, sectorsList, withoutOther } from './exposure-utils'

  /**
   * Asset shell (EPIC K.4b, spec §4.2/§6.3): the single long asset page
   * split into a sticky header + three deep-linkable nested-route tabs
   * (Overview `/`, Exposure `/exposure`, Data `/data`). The header carries
   * the identity row (back link, ticker + name + identity chips
   * [type · class · currency · exchange] and the non-Yahoo "no auto sync"
   * warning — the update-from-Yahoo / backfill / delete actions the old
   * "Caratteristiche" `⋮` menu exposed live in the Data tab's danger zone
   * only: the duplicated header `⋯` menu was removed in #117), the quote
   * strip in the asset currency (headline last close + compact 1D/1W/1M/
   * 1Y/YTD delta chips, the old "Metriche quote" card promoted into the
   * always-visible header) and the route-linked `ui/Tabs` bar; the tabs
   * swap below via `{@render children()}` while this layout — and all of
   * its data — stays mounted.
   *
   * Data ownership: every fetch the old page performed is performed here
   * unchanged (asset + quote + prices + exposure + splits together, the
   * fresh quote/prices refetch when the session price refresh completes —
   * the refresh itself is triggered globally by the shell now, watched here
   * via `priceRefresh.revision` so deep links update like everywhere else —
   * the metadata PATCH and the whole exposure save/prefill/derive
   * machinery) and handed to the tab pages through the typed context in
   * `./context.ts` — no tab re-fetches anything on its own. The geo/sector
   * edit modals and the delete confirmation are mounted here (same
   * contract as K.4a's transaction modal): their working-copy `$state`
   * lists are bound natively, and the tabs only trigger them through
   * `openGeoModal`/`openSectorModal`/`requestDelete`.
   *
   * New here: `loadHoldings()` — the isolated, non-blocking
   * `portfolioApi.dashboard()` fetch that feeds the Overview tab's "Where
   * held" rows (spec §4.2 decision 5): the endpoint already answers with
   * `assets: PortfolioAssets[]` (per-portfolio `AssetPerformance[]`), so
   * the "which of my portfolios hold this asset" view is derived
   * client-side without any backend addition; a failure just flips
   * `heldFailed` (the Overview tab then degrades the block to an
   * "unavailable" note) and never toasts or blocks the page.
   */
  let { children }: { children: Snippet } = $props()

  const id = $derived(page.params.id as string | undefined)

  let loading = $state(true)
  let asset = $state<Asset | null>(null)
  let quote = $state<AssetQuote | null>(null)
  let prices = $state<Price[]>([])
  let splits = $state<SplitInfo[]>([])
  let exposure = $state<AssetExposure | null>(null)
  // "Where held" rows (see the docstring): `null` until the isolated
  // dashboard fetch resolves, `heldFailed` when it did not.
  let held = $state<HeldRow[] | null>(null)
  let heldFailed = $state(false)
  // Working copy of the modals. Prefill handlers write provider data here as a
  // non-persisted preview: the page cards never read these lists, they render
  // the stored `exposure` (see the Exposure tab's display* derivations), so an
  // unsaved preview only lives inside the modal until the user presses Save.
  // The pending copy is scoped to a single modal session: `openGeoModal`/
  // `openSectorModal` re-hydrate these lists (and their provenance state)
  // from the saved `exposure` before every open, so edits left unsaved when
  // the modal was last closed are discarded on reopen.
  let regionsEdit = $state<ExposureRow[]>([])
  let sectorsEdit = $state<ExposureRow[]>([])
  let countriesEdit = $state<ExposureRow[]>([])
  // Data provenance for the geo/sector modal badges: which source currently
  // owns each dimension ('manual' once the user edits it) and when it was
  // last persisted. Hydrated from `ex.provenance` on load and re-hydrated on
  // every modal reopen (see `openGeoModal`/`openSectorModal`), set by the
  // prefill/dirty handlers and confirmed by every save. A null source means
  // unknown (no badge); a null updatedAt means the shown source is not
  // persisted yet (unsaved preview or fresh manual edit), so the badge shows
  // the label without a date until the next successful save.
  let countriesSource = $state<string | null>(null)
  let regionsSource = $state<string | null>(null)
  let sectorsSource = $state<string | null>(null)
  let countriesUpdatedAt = $state<string | null>(null)
  let regionsUpdatedAt = $state<string | null>(null)
  let sectorsUpdatedAt = $state<string | null>(null)
  let savingRegions = $state(false)
  let savingSectors = $state(false)
  let savingCountries = $state(false)
  let saving = $state(false)
  let prefilling = $state(false)
  let fetchingETF = $state(false)
  let fetchingMorningstar = $state(false)
  let derivingRegions = $state(false)
  let refreshingMeta = $state(false)
  let backfillingHistory = $state(false)
  // #193: true while an open sync/backfill job updates this asset's data
  // (`data_status.state === 'syncing'`); the Overview tab swaps the empty
  // chart for a spinner while it holds.
  let dataSyncing = $state(false)
  let syncTimer: ReturnType<typeof setInterval> | undefined
  let geoModalOpen = $state(false)
  let sectorModalOpen = $state(false)

  let form = $state<AssetForm>({
    ticker: '',
    isin: '',
    name: '',
    type: 'stock',
    currency: 'USD',
    exchange: '',
    asset_class: 'other',
    price_source: 'yahoo',
  })

  const currency = $derived(asset?.currency || 'USD')

  const hasChanges = $derived.by(() => {
    if (!asset) return false
    return (
      form.ticker !== asset.ticker ||
      form.isin !== (asset.isin || '') ||
      form.name !== asset.name ||
      form.type !== asset.type ||
      form.currency !== asset.currency ||
      form.exchange !== (asset.exchange || '') ||
      form.asset_class !== (asset.asset_class || 'other') ||
      form.price_source !== (asset.price_source || 'yahoo')
    )
  })

  // Totals are computed on the 2-decimal values that actually enter the edit
  // lists (also mirrored in the modal footer): this keeps the displayed sum,
  // the save validation and the persisted weights identical, so a provider
  // total at float precision (e.g. 100.004) never trips the > 100 guard.
  const sumRegions = $derived(
    Math.round(regionsEdit.reduce((acc, r) => acc + (Number(r.weight) || 0), 0) * 100) / 100,
  )
  const sumSectors = $derived(
    sectorsEdit.reduce((acc, r) => acc + (Number(r.weight) || 0), 0),
  )
  const sumCountries = $derived(
    Math.round(
      countriesEdit.reduce((acc, r) => acc + (Number(r.weight) || 0), 0) * 100,
    ) / 100,
  )
  // Countries and regions may legitimately sum below 100: for regions the
  // backend folds the residual into «Other / Not Classified» at persist time;
  // for countries the residual just stays unattributed (saving countries no
  // longer re-derives the regions — that happens only via «Calcola da paesi»).
  // Only a sum above 100 (± float epsilon) blocks saving. Sectors still
  // require 100 ±0.5.
  const regionsValid = $derived(sumRegions <= 100 + 1e-9)
  const sectorsValid = $derived(Math.abs(sumSectors - 100) <= 0.5)
  const countriesValid = $derived(sumCountries <= 100 + 1e-9)

  // 1D/1W/1M/1Y/YTD quote deltas of `AssetQuote`, as compact chips in the
  // sticky header (spec §6.3; the old body-level "Metriche quote" card's
  // five metrics — same values, same zero-muted coloring via `PnlValue`).
  const METRICS: {
    key: 'change_1d' | 'change_1w' | 'change_1m' | 'change_1y' | 'change_ytd'
    label: MessageKey
  }[] = [
    { key: 'change_1d', label: 'asset.chip1d' },
    { key: 'change_1w', label: 'asset.chip1w' },
    { key: 'change_1m', label: 'asset.chip1m' },
    { key: 'change_1y', label: 'asset.chip1y' },
    { key: 'change_ytd', label: 'asset.chipYtd' },
  ]

  // Identity chips of the header (spec §6.3 `[ETF | equity | EUR | XETRA]`):
  // type and class go through the shared localized label helpers, currency
  // and exchange are shown raw (exchange only when known).
  const identityChips = $derived.by(() => {
    if (!asset) return [] as string[]
    const chips: string[] = [assetTypeLabel(asset.type)]
    if (asset.asset_class) {
      chips.push(assetClassLabel(asset.asset_class))
    }
    chips.push(asset.currency)
    if (asset.exchange) chips.push(asset.exchange)
    return chips
  })

  // The old header's non-Yahoo warning (kept verbatim): manual/no-price
  // sources never participate in the automatic sync.
  const priceSourceWarning = $derived(
    asset?.price_source && asset.price_source !== 'yahoo'
      ? t('asset.noAutoSync', { source: priceSourceLabel(asset.price_source) })
      : '',
  )

  function fillForm(a: Asset): void {
    form = {
      ticker: a.ticker,
      isin: a.isin || '',
      name: a.name,
      type: a.type,
      currency: a.currency,
      exchange: a.exchange || '',
      asset_class: a.asset_class || 'other',
      price_source: a.price_source || 'yahoo',
    }
  }

  onMount(load)
  onDestroy(stopSyncPolling)

  // #193: while the asset reports `syncing` (per-asset backfill or the
  // global app-load asset sync), poll it every ~4s; once the job is gone,
  // refetch everything the job wrote (quote, prices AND splits) and stop.
  // The poll bypasses the 60s client GET cache, otherwise it would keep
  // reading the stale "syncing" payload for up to a minute. Plain interval
  // id, re-armed by `load()`.
  function startSyncPolling(): void {
    stopSyncPolling()
    if (!id) return
    syncTimer = setInterval(async () => {
      if (!id) return
      try {
        const a = await assetApi.get(id, { noCache: true })
        asset = a
        dataSyncing = a.data_status?.state === 'syncing'
        if (!dataSyncing) {
          stopSyncPolling()
          void refetchAssetData()
        }
      } catch {
        // Keep the previous data; the next tick retries.
      }
    }, 4000)
  }

  function stopSyncPolling(): void {
    if (syncTimer !== undefined) {
      clearInterval(syncTimer)
      syncTimer = undefined
    }
  }

  /** Refetch the price-derived data a backfill/sync writes — the asset (for
   * its fresh `data_status`), quote, prices and splits — clearing the client
   * GET cache first so none of them comes back stale (the poll bypasses the
   * cache, so the old payloads would otherwise be served for up to a
   * minute). Used when a job ends. */
  async function refetchAssetData(): Promise<void> {
    if (!id) return
    clearGetCache()
    try {
      const [freshAsset, freshQuote, freshPrices, freshSplits] = await Promise.all([
        assetApi.get(id, { noCache: true }),
        assetApi.quote(id),
        pricesApi.byAsset(id),
        assetApi.splits(id),
      ])
      asset = freshAsset
      dataSyncing = freshAsset.data_status?.state === 'syncing'
      quote = freshQuote
      prices = freshPrices
      splits = freshSplits
    } catch {
      // Keep current data.
    }
  }

  async function load(): Promise<void> {
    if (!id) return
    stopSyncPolling()
    try {
      const [a, q, ps, ex, sp] = await Promise.all([
        assetApi.get(id),
        assetApi.quote(id),
        pricesApi.byAsset(id),
        assetApi.exposure(id),
        assetApi.splits(id),
      ])
      asset = a
      dataSyncing = a.data_status?.state === 'syncing'
      if (dataSyncing) startSyncPolling()
      quote = q
      prices = ps
      exposure = ex
      splits = sp
      regionsEdit = withoutOther(ex.regions)
      sectorsEdit = sectorsList(ex.sectors)
      countriesEdit = positiveCountries(ex.countries)
      // Hydrate the persisted provenance per dimension (source + last-update
      // date). Dimensions never persisted carry no provenance entry: badge
      // hidden (source null).
      countriesSource = ex.provenance?.countries?.source ?? null
      countriesUpdatedAt = ex.provenance?.countries?.updated_at ?? null
      regionsSource = ex.provenance?.regions?.source ?? null
      regionsUpdatedAt = ex.provenance?.regions?.updated_at ?? null
      sectorsSource = ex.provenance?.sectors?.source ?? null
      sectorsUpdatedAt = ex.provenance?.sectors?.updated_at ?? null
      fillForm(a)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.detailLoadFailed')
      toast.error(message)
      const status = (err as { status?: number } | null)?.status
      if (status === 404) {
        goto(resolve('/assets'), { replaceState: true })
        return
      }
    } finally {
      loading = false
    }

    // "Where held" (K.4b): never awaited — the tabs render as soon as the
    // main payload is in.
    void loadHoldings()
  }

  // Refetch the price-derived header data when the (globally triggered)
  // session refresh completes: the POST cleared the GET cache, so the quote
  // strip and the price history come back fresh — same on a deep link, where
  // this layout is the first page mounted.
  async function refetchAfterRefresh(): Promise<void> {
    if (!id) return
    try {
      const [freshQuote, freshPrices] = await Promise.all([
        assetApi.quote(id),
        pricesApi.byAsset(id),
      ])
      quote = freshQuote
      prices = freshPrices
    } catch {
      // Keep current data.
    }
  }

  // Plain (non-`$state`) baseline seeded at component init: only refresh
  // completions that happen while this layout is mounted refetch.
  let seenRefresh = priceRefresh.revision
  $effect(() => {
    const rev = priceRefresh.revision
    if (rev === seenRefresh) return
    seenRefresh = rev
    void refetchAfterRefresh()
  })

  /** Fill `held` with one row per portfolio currently holding this asset
   * (qty > 0: closed holdings are skipped), derived from the per-portfolio
   * `assets` of `GET /dashboard`. Deliberately silent on failure — the
   * block is supplementary and the Overview tab shows an "unavailable"
   * note via `heldFailed` instead of ever blocking or toasting. */
  async function loadHoldings(): Promise<void> {
    if (!id) return
    try {
      const dash = await portfolioApi.dashboard()
      const rows: HeldRow[] = []
      for (const p of dash.assets ?? []) {
        for (const a of p.assets ?? []) {
          if (a.asset_id === id && Number(a.qty) > 0) {
            rows.push({
              portfolioId: p.portfolio_id,
              portfolioName: p.portfolio_name,
              currency: a.currency,
              qty: a.qty,
              invested: a.invested,
              value: a.value,
              gain_loss: a.gain_loss,
              roi: a.roi,
            })
          }
        }
      }
      // Richest holding first, like the dashboard's invested-assets table.
      rows.sort((x, y) => Number(y.value) - Number(x.value))
      held = rows
    } catch {
      heldFailed = true
    }
  }

  async function saveAsset(): Promise<void> {
    if (!id || !asset) return
    if (!form.ticker.trim() || !form.name.trim() || !form.currency.trim()) {
      toast.error(t('asset.formRequiredFields'))
      return
    }
    saving = true
    const patch: AssetPatch = {
      ticker: form.ticker.trim(),
      isin: form.isin.trim(),
      name: form.name.trim(),
      type: form.type,
      currency: form.currency.trim(),
      exchange: form.exchange.trim(),
      asset_class: form.asset_class,
      price_source: form.price_source,
    }
    try {
      const updated = await assetApi.update(id, patch)
      asset = updated
      fillForm(updated)
      toast.success(t('asset.updated'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.saveFailed')
      toast.error(message)
    } finally {
      saving = false
    }
  }

  async function refreshFromYahoo(): Promise<void> {
    if (!asset) return
    refreshingMeta = true
    try {
      // La classe arriva da meta; l'override manuale deve vincere: se l'asset
      // ha già una classe diversa da "other"/vuota, il refresh non la sovrascrive.
      const meta = await assetApi.meta(asset.ticker)
      form = {
        ...form,
        name: meta.name || form.name,
        type: meta.type || form.type,
        currency: meta.currency || form.currency,
        exchange: meta.exchange || form.exchange,
        asset_class:
          !asset.asset_class || asset.asset_class === 'other'
            ? meta.asset_class || form.asset_class
            : form.asset_class,
      }
      toast.success(t('asset.metaRefreshed'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.metaRefreshFailed')
      toast.error(message)
    } finally {
      refreshingMeta = false
    }
  }

  // Backfill asincrono: il POST accoda un job (202) e `backfillingHistory`
  // copre l'attesa del polling. Al termine `refetchAssetData` ricarica quote,
  // prezzi e split (non solo i prezzi): il job scrive anche gli split, e
  // senza il refetch del cache client resterebbero quelli vecchi. `partial`
  // significa storico ok ma split non recuperati.
  async function backfillHistory(): Promise<void> {
    if (!id) return
    backfillingHistory = true
    try {
      const { job_id } = await assetApi.backfillHistory(id)
      const job = await jobsApi.wait(job_id)
      if (job.status === 'done' || job.status === 'partial') {
        await refetchAssetData()
      }
      if (job.status === 'done') {
        toast.success(t('asset.backfillDone'))
      } else if (job.status === 'partial') {
        toast.warning(t('asset.backfillPartial'))
      } else {
        throw new Error(job.error || t('asset.backfillFailed'))
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.backfillFailed')
      toast.error(message)
    } finally {
      backfillingHistory = false
    }
  }

  // Prefill da JustETF: popola SOLO la lista paesi della modale (raw JustETF).
  // È un'anteprima NON persistita: `exposure` (e quindi le card) resta ai dati
  // salvati; solo «Salva paesi» invierà la lista al backend. L'ISIN arriva già
  // persistito dal server, quindi lo sincronizziamo nel form.
  async function prefillCountriesFromETF(): Promise<void> {
    if (!id || !asset) return
    fetchingETF = true
    try {
      const preview = await assetApi.fetchETFExposure(id)
      countriesEdit = positiveCountries(preview.countries)
      countriesSource = 'justetf'
      // Unsaved preview: no persisted date yet (badge shows the label only).
      countriesUpdatedAt = null
      if (preview.isin) form.isin = preview.isin
      toast.success(t('asset.countriesPrefilledJustEtf'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.downloadFailed')
      toast.error(message)
    } finally {
      fetchingETF = false
    }
  }

  // Deriva le regioni canoniche dai paesi correnti (preview, non persistito).
  async function deriveRegionsFromCountries(): Promise<void> {
    if (!id || !asset) return
    if (!countriesEdit.some((c) => Number(c.weight) > 0)) {
      toast.error(t('asset.noWeightedCountries'))
      return
    }
    derivingRegions = true
    try {
      const result = await assetApi.deriveRegions(id, countriesEdit)
      regionsEdit = withoutOther(result.regions)
      // The residual «Other / Not Classified» row is filtered out: the modal's
      // totals line already explains what is left unattributed.
      regionsSource = countriesSource === 'justetf' ? 'derived-etf' : 'derived'
      // Preview only: drop any previously persisted date until it is saved.
      regionsUpdatedAt = null
      toast.success(t('asset.regionsRecomputed'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.computeFailed')
      toast.error(message)
    } finally {
      derivingRegions = false
    }
  }

  // Prefill da Morningstar: popola SOLO la lista regioni della modale con le
  // regioni ufficiali Morningstar. Anteprima NON persistita: la card regioni
  // continua a mostrare le regioni salvate finché non si preme «Salva regioni».
  async function prefillRegionsFromMorningstar(): Promise<void> {
    if (!id || !asset) return
    fetchingMorningstar = true
    try {
      const preview = await assetApi.fetchMorningstarExposure(id)
      regionsEdit = withoutOther(preview.regions)
      regionsSource = 'morningstar-regions'
      regionsUpdatedAt = null
      if (preview.isin) form.isin = preview.isin
      toast.success(t('asset.regionsPrefilledMorningstar'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.downloadFailed')
      toast.error(message)
    } finally {
      fetchingMorningstar = false
    }
  }

  // Prefill da JustETF: popola SOLO la lista settori della modale. Anteprima
  // NON persistita: la card settori resta ai dati salvati finché non si salva.
  async function prefillSectorsFromETF(): Promise<void> {
    if (!id || !asset) return
    fetchingETF = true
    try {
      const preview = await assetApi.fetchETFExposure(id)
      sectorsEdit = sectorsList(preview.sectors)
      sectorsSource = 'justetf'
      sectorsUpdatedAt = null
      if (preview.isin) form.isin = preview.isin
      toast.success(t('asset.sectorsPrefilledJustEtf'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.downloadFailed')
      toast.error(message)
    } finally {
      fetchingETF = false
    }
  }

  // Prefill da Yahoo: popola SOLO la lista settori della modale (topHoldings).
  // Per le azioni singole ricade sul settore unico al 100% (assetProfile).
  // Anteprima NON persistita: la card settori mostra i dati salvati finché non
  // si preme «Salva».
  async function prefillSectorsFromYahoo(): Promise<void> {
    if (!id) return
    prefilling = true
    try {
      const preview = await assetApi.fetchExposure(id)
      sectorsEdit = sectorsList(preview.sectors)
      sectorsSource = 'yahoo'
      sectorsUpdatedAt = null
      toast.success(t('asset.sectorsPrefilledYahoo'))
    } catch (err: unknown) {
      const status = (err as { status?: number } | null)?.status
      const message =
        status === 502
          ? t('asset.yahooNoResponse')
          : err instanceof Error
            ? err.message
            : t('asset.prefillFailed')
      toast.error(message)
    } finally {
      prefilling = false
    }
  }

  // Prefill da Morningstar: popola SOLO la lista settori della modale. Anteprima
  // NON persistita: la card settori resta ai dati salvati finché non si salva
  // (stesso pattern dei prefill settori JustETF/Yahoo). L'endpoint è cachato per
  // ISIN lato backend: se paesi/regioni sono già stati letti la chiamata è
  // immediata.
  async function prefillSectorsFromMorningstar(): Promise<void> {
    if (!id || !asset) return
    fetchingMorningstar = true
    try {
      const preview = await assetApi.fetchMorningstarExposure(id)
      sectorsEdit = sectorsList(preview.sectors)
      sectorsSource = 'morningstar'
      sectorsUpdatedAt = null
      if (preview.isin) form.isin = preview.isin
      toast.success(t('asset.sectorsPrefilledMorningstar'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.downloadFailed')
      toast.error(message)
    } finally {
      fetchingMorningstar = false
    }
  }

  // Il backend salva ogni dimensione indipendentemente: ogni sezione invia
  // SOLO la propria dimensione (l'altra viene omessa dal JSON → nil → non
  // toccata) insieme alla sua fonte di provenienza (`*_source`; senza fonte
  // il backend usa 'manual'). Dopo il successo la risposta canonica rinfresca
  // `exposure` (quindi le card), risincronizza SOLO la lista di edit della
  // dimensione salvata e ne riporta la provenance persistita (source +
  // data): le altre liste di edit sono la working copy della modale e
  // possono contenere modifiche pendenti non salvate, quindi non vanno
  // mai toccate qui.
  async function saveRegions(): Promise<void> {
    if (!id || !exposure || !regionsValid) return
    savingRegions = true
    const sentSource = regionsSource ?? 'manual'
    try {
      const saved = await assetApi.saveExposure(id, {
        regions: regionsEdit,
        regions_source: sentSource,
      })
      exposure = saved
      regionsEdit = withoutOther(saved.regions)
      // Provenance confirmed by the canonical response; fall back to the
      // sent source if the backend did not echo it, and drop the date when
      // absent so the badge never shows a stale timestamp.
      regionsSource = saved.provenance?.regions?.source ?? sentSource
      regionsUpdatedAt = saved.provenance?.regions?.updated_at ?? null
      toast.success(t('asset.geoSaved'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.saveFailed')
      toast.error(message)
    } finally {
      savingRegions = false
    }
  }

  async function saveSectors(): Promise<void> {
    if (!id || !exposure || !sectorsValid) return
    savingSectors = true
    const sentSource = sectorsSource ?? 'manual'
    try {
      const saved = await assetApi.saveExposure(id, {
        sectors: sectorsEdit,
        sectors_source: sentSource,
      })
      exposure = saved
      sectorsEdit = sectorsList(saved.sectors)
      sectorsSource = saved.provenance?.sectors?.source ?? sentSource
      sectorsUpdatedAt = saved.provenance?.sectors?.updated_at ?? null
      toast.success(t('asset.sectorsSaved'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.saveFailed')
      toast.error(message)
    } finally {
      savingSectors = false
    }
  }

  // Prefill da Morningstar: popola paesi e settori della modale (ETF only).
  // Anteprima NON persistita: le card restano ai dati salvati; ogni dimensione
  // va salvata con il proprio pulsante Salva.
  async function prefillCountriesFromMorningstar(): Promise<void> {
    if (!id || !asset) return
    fetchingMorningstar = true
    try {
      const preview = await assetApi.fetchMorningstarExposure(id)
      countriesEdit = positiveCountries(preview.countries)
      countriesSource = 'morningstar'
      countriesUpdatedAt = null
      sectorsEdit = sectorsList(preview.sectors)
      sectorsSource = 'morningstar'
      sectorsUpdatedAt = null
      if (preview.isin) form.isin = preview.isin
      toast.success(t('asset.countriesSectorsPrefilledMorningstar'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.downloadFailed')
      toast.error(message)
    } finally {
      fetchingMorningstar = false
    }
  }

  async function saveCountries(): Promise<void> {
    if (!id || !exposure || !countriesValid) return
    savingCountries = true
    const sentSource = countriesSource ?? 'manual'
    try {
      const saved = await assetApi.saveExposure(id, {
        countries: countriesEdit,
        countries_source: sentSource,
      })
      exposure = saved
      countriesEdit = positiveCountries(saved.countries)
      countriesSource = saved.provenance?.countries?.source ?? sentSource
      countriesUpdatedAt = saved.provenance?.countries?.updated_at ?? null
      // Regions and sectors are deliberately NOT touched here: saving one
      // dimension must never clobber the other dimensions' edit lists, which
      // are the modal's working copy and may hold unsaved pending edits
      // (manual or from a prefill). The stored data still refreshes via
      // `exposure` (cards), and regionsSource is left alone so the regions
      // provenance badge keeps reflecting its real source.
      toast.success(t('asset.countriesSaved'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.saveFailed')
      toast.error(message)
    } finally {
      savingCountries = false
    }
  }

  // Provenance flips to 'manual' on the first user mutation of a dimension
  // (invoked by the geo/sector modals at every add/remove/weight-edit point).
  // The persisted date is dropped as well: the badge shows a date only once
  // the new state has actually been saved.
  function markCountriesManual(): void {
    countriesSource = 'manual'
    countriesUpdatedAt = null
  }

  function markRegionsManual(): void {
    regionsSource = 'manual'
    regionsUpdatedAt = null
  }

  function markSectorsManual(): void {
    sectorsSource = 'manual'
    sectorsUpdatedAt = null
  }

  // Open the geo modal after resetting its working copy from the SAVED
  // exposure: the same hydration `load` performs (`withoutOther`/
  // `positiveCountries` normalisation + persisted provenance with the `?? null`
  // fallbacks). Any unsaved manual edits or prefill previews left over from
  // the previous modal session — including a 'manual' provenance flip that was
  // never persisted — are discarded here, so reopening always shows the
  // stored data.
  function openGeoModal(): void {
    if (!exposure) return
    regionsEdit = withoutOther(exposure.regions)
    countriesEdit = positiveCountries(exposure.countries)
    countriesSource = exposure.provenance?.countries?.source ?? null
    countriesUpdatedAt = exposure.provenance?.countries?.updated_at ?? null
    regionsSource = exposure.provenance?.regions?.source ?? null
    regionsUpdatedAt = exposure.provenance?.regions?.updated_at ?? null
    geoModalOpen = true
  }

  // Sector-modal counterpart of `openGeoModal`: restores `sectorsEdit` and the
  // sectors provenance from the saved exposure before opening, so unsaved
  // sector edits/previews never resurface on reopen.
  function openSectorModal(): void {
    if (!exposure) return
    sectorsEdit = sectorsList(exposure.sectors)
    sectorsSource = exposure.provenance?.sectors?.source ?? null
    sectorsUpdatedAt = exposure.provenance?.sectors?.updated_at ?? null
    sectorModalOpen = true
  }

  // Delete mirrors the portfolio shell and the assets list: confirm dialog →
  // API → toast → leave the (now gone) asset. `goto` is deliberately not
  // awaited so this handler never races the dialog's close-then-unmount.
  // `requestDelete` reaches this layout only through the context (the Data
  // tab's Danger zone): the header `⋯` menu that used to duplicate these
  // actions was removed (#117), together with its open/close machinery.
  let showDeleteDialog = $state(false)
  let deleting = $state(false)

  function requestDelete(): void {
    showDeleteDialog = true
  }

  async function deleteAsset(): Promise<void> {
    if (!id) return
    deleting = true
    try {
      await assetApi.remove(id)
      toast.success(t('asset.deleted'))
      void goto(resolve('/assets'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.deleteFailed')
      toast.error(message)
    } finally {
      deleting = false
    }
  }

  // --- Tabs (spec §4.2.2: nested routes, real URLs) ------------------------
  // hrefs resolved per the repo convention; `ui/Tabs` derives the active
  // tab from the URL, so deep links and the back button just work.
  const tabItems = $derived(
    id
      ? [
          { href: resolve(`/assets/${id}`), label: t('asset.tabOverview') },
          { href: resolve(`/assets/${id}/exposure`), label: t('asset.tabExposure') },
          { href: resolve(`/assets/${id}/data`), label: t('asset.tabData') },
        ]
      : [],
  )

  // --- Context for the tab pages (see ./context.ts) -------------------------
  setAssetPage({
    get id() {
      return id
    },
    get loading() {
      return loading
    },
    get asset() {
      return asset
    },
    get quote() {
      return quote
    },
    get prices() {
      return prices
    },
    get splits() {
      return splits
    },
    get exposure() {
      return exposure
    },
    get currency() {
      return currency
    },
    get held() {
      return held
    },
    get heldFailed() {
      return heldFailed
    },
    get refreshingMeta() {
      return refreshingMeta
    },
    refreshFromYahoo,
    get backfillingHistory() {
      return backfillingHistory
    },
    backfillHistory,
    get dataSyncing() {
      return dataSyncing
    },
    requestDelete,
    get form() {
      return form
    },
    get hasChanges() {
      return hasChanges
    },
    get saving() {
      return saving
    },
    saveAsset,
    openGeoModal,
    openSectorModal,
  } satisfies AssetPageContext)
</script>

<div class="p-4 lg:p-6">
  <!-- Sticky entity header (spec §5.1/§6.3): `top` follows the live
       --app-header-h published by AppShell, so it stays flush while the app
       header condenses 56px → 44px (no gap strip); the same transition makes
       the two move in lockstep. The negative margins + padding are the
       responsive mirror of the wrapper padding (= `<main>`'s), so the opaque
       `bg-background` band spans the full content width at every size. -->
  <header
    class="sticky top-[var(--app-header-h)] z-10 -mx-4 -mt-4 mb-6 flex flex-col gap-3 bg-background px-4 pb-3 pt-4 transition-[top] duration-base ease-standard lg:-mx-6 lg:-mt-6 lg:px-6 lg:pt-6"
  >
    <div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
      <div class="min-w-0">
        <a
          href={resolve('/assets')}
          class="focus-ring -ml-1 inline-flex items-center gap-1 rounded-control text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft class="h-4 w-4 shrink-0" aria-hidden="true" />
          {t('asset.back')}
        </a>
        {#if asset}
          <h1 class="mt-1 flex flex-wrap items-baseline gap-x-2 text-2xl font-bold">
            <span class="font-mono">{asset.ticker}</span>
            <span class="min-w-0 text-sm font-medium text-muted-foreground">{asset.name}</span>
          </h1>
          <div class="mt-2 flex flex-wrap items-center gap-1.5">
            {#each identityChips as chip (chip)}
              <Badge>{chip}</Badge>
            {/each}
            {#if priceSourceWarning}
              <Badge variant="warning">{priceSourceWarning}</Badge>
            {/if}
          </div>
        {:else if loading}
          <p class="mt-1 text-sm text-muted-foreground">{t('common.loading')}</p>
        {/if}
      </div>
    </div>

    {#if quote?.has_data}
      <!-- Quote strip (spec §6.3 "identity chips + quote metrics"): the
           headline last close in the ASSET currency followed by the compact
           1D/1W/1M/1Y/YTD delta chips (signed ▲▼ via `PnlValue`, never
           colour alone, D6) and the last price date. Same values as the old
           body-level "Metriche quote" card, always visible while tabs swap. -->
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <span class="text-xl font-bold tabular-nums">
          {formatCurrency(quote.last_close, quote.currency)}
        </span>
        {#each METRICS as m (m.key)}
          <span class="inline-flex items-center gap-1.5 rounded-full border border-border px-2 py-0.5 text-xs">
            <span class="text-muted-foreground">{t(m.label)}</span>
            <PnlValue value={quote[m.key]} kind="percent" size="sm" />
          </span>
        {/each}
        <span class="text-xs text-muted-foreground">
          {t('asset.dataAsOf', { date: new Date(quote.last_date).toLocaleDateString() })}
        </span>
      </div>
    {:else if quote}
      <p class="text-sm text-muted-foreground">{t('asset.noPriceData')}</p>
    {/if}

    <Tabs items={tabItems} ariaLabel={t('asset.tabsLabel')} />
  </header>

  {@render children()}
</div>

{#if asset}
  <ExposureGeoModal
    bind:open={geoModalOpen}
    onClose={() => (geoModalOpen = false)}
    bind:regionsEdit
    bind:countriesEdit
    {sumRegions}
    {sumCountries}
    {savingRegions}
    {savingCountries}
    {saveRegions}
    {saveCountries}
    {fetchingETF}
    {fetchingMorningstar}
    {derivingRegions}
    {prefillCountriesFromETF}
    {deriveRegionsFromCountries}
    {prefillRegionsFromMorningstar}
    {prefillCountriesFromMorningstar}
    {countriesSource}
    {regionsSource}
    {countriesUpdatedAt}
    {regionsUpdatedAt}
    onCountriesDirty={markCountriesManual}
    onRegionsDirty={markRegionsManual}
    assetType={asset.type}
  />

  <ExposureSectorModal
    bind:open={sectorModalOpen}
    onClose={() => (sectorModalOpen = false)}
    bind:sectorsEdit
    {sumSectors}
    {sectorsValid}
    {savingSectors}
    {saveSectors}
    {prefilling}
    {fetchingETF}
    {fetchingMorningstar}
    {prefillSectorsFromETF}
    {prefillSectorsFromYahoo}
    {prefillSectorsFromMorningstar}
    {sectorsSource}
    {sectorsUpdatedAt}
    onSectorsDirty={markSectorsManual}
    assetType={asset.type}
  />
{/if}

<ConfirmDialog
  bind:open={showDeleteDialog}
  variant="danger"
  title={t('asset.delete')}
  message={t('asset.deleteConfirm', { ticker: asset?.ticker ?? '' })}
  confirmLabel={t('common.delete')}
  cancelLabel={t('common.cancel')}
  loading={deleting}
  onconfirm={deleteAsset}
/>
