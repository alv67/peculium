<script lang="ts">
  import { toast } from '#lib/stores/toast.svelte.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import { ASSET_TYPES, assetTypeLabel } from '#lib/format.js'
  import { assetApi, type AssetLookupResult, type Currency } from '#lib/services/api.js'
  import Modal from '#lib/components/ui/Modal.svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import Field from '#lib/components/ui/Field.svelte'
  import Input from '#lib/components/ui/Input.svelte'
  import Select from '#lib/components/ui/Select.svelte'
  import AssetSearchAutocomplete from './AssetSearchAutocomplete.svelte'
  import CurrencySelect from './CurrencySelect.svelte'

  /** Asset creation dialog (EPIC E.3): owns the form, the API call and the toast. */
  let {
    open = $bindable(false),
    currencies = [],
    onsuccess = undefined,
  }: {
    open?: boolean
    currencies?: Currency[]
    onsuccess?: () => void
  } = $props()

  const uid = $props.id()

  const defaultForm = () => ({
    ticker: '',
    isin: '',
    name: '',
    type: 'stock' as string,
    currency: 'USD',
    country: '',
    exchange: '',
    asset_class: '',
    sector: '',
    industry: '',
    price_source: 'yahoo',
  })

  let form = $state(defaultForm())
  let saving = $state(false)

  const defaultCurrency = $derived(
    currencies.find((c) => c.code === 'USD')?.code ?? currencies[0]?.code ?? 'USD',
  )

  function resetForm(): void {
    form = { ...defaultForm(), currency: defaultCurrency }
  }

  // Closing (or first mount while still closed) clears the draft form.
  $effect(() => {
    if (!open) resetForm()
  })

  async function handleSelect(result: AssetLookupResult): Promise<void> {
    form = {
      ticker: result.ticker,
      isin: form.isin,
      name: result.name || '',
      type: result.type || 'stock',
      currency: result.currency || 'USD',
      country: '',
      exchange: result.exchange || '',
      asset_class: '',
      sector: '',
      industry: '',
      price_source: 'yahoo',
    }

    try {
      const meta = await assetApi.meta(result.ticker)
      form = {
        ...form,
        name: meta.name || form.name,
        type: meta.type || form.type,
        currency: meta.currency || form.currency,
        country: meta.country || form.country,
        exchange: meta.exchange || form.exchange,
        asset_class: meta.asset_class || '',
        sector: meta.sector || '',
        industry: meta.industry || '',
      }
    } catch {
      // keep lookup defaults; currency/type remain editable
    }
  }

  async function submit(): Promise<void> {
    saving = true
    try {
      await assetApi.create(form)
      toast.success(t('asset.created'))
      open = false
      onsuccess?.()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.createFailed')
      toast.error(message)
    } finally {
      saving = false
    }
  }
</script>

{#snippet footer()}
  <Button variant="secondary" onclick={() => (open = false)} disabled={saving}>{t('common.cancel')}</Button>
  <Button onclick={submit} disabled={!form.ticker || !form.name} loading={saving}>{t('common.save')}</Button>
{/snippet}

<Modal
  bind:open
  title={t('asset.newTitle')}
  description={t('asset.lookupHint')}
  size="lg"
  {footer}
  dismissible={!saving}
>
  <div class="grid gap-3 sm:grid-cols-2">
    <Field label={t('positions.colTicker')} for={`${uid}-ticker`}>
      <AssetSearchAutocomplete
        inputId={`${uid}-ticker`}
        bind:ticker={form.ticker}
        onselect={handleSelect}
      />
    </Field>
    <Field label={t('chartView.colName')}>
      <Input bind:value={form.name} placeholder={t('chartView.colName')} />
    </Field>
    <Field label={t('asset.factIsin')} hint={t('common.optional')}>
      <Input bind:value={form.isin} />
    </Field>
    <Field label={t('asset.factType')}>
      <!-- Same value list the Data tab's Type select reads, localized labels
           from format.ts. -->
      <Select bind:value={form.type}>
        {#each ASSET_TYPES as value (value)}
          <option value={value}>{assetTypeLabel(value)}</option>
        {/each}
      </Select>
    </Field>
    <Field label={t('asset.factCurrency')}>
      <CurrencySelect bind:value={form.currency} {currencies} />
    </Field>
    <Field label={t('asset.factExchange')}>
      <Input bind:value={form.exchange} placeholder={t('asset.factExchange')} />
    </Field>
    <Field label={t('asset.factPriceSource')}>
      <Select bind:value={form.price_source}>
        <option value="yahoo">{t('asset.priceSourceYahoo')}</option>
        <option value="manual">{t('asset.priceSourceManual')}</option>
        <option value="none">{t('asset.priceSourceNone')}</option>
      </Select>
    </Field>
  </div>
</Modal>
