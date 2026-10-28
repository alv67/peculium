<script lang="ts">
  import { toast } from '#lib/stores/toast.svelte.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import { ASSET_TYPES, assetTypeLabel } from '#lib/format.js'
  import { Banknote, Bitcoin, ChartCandlestick, Coins, FileBadge, Layers, PiggyBank, ScrollText } from 'lucide-svelte'
  import { assetProfile, createHint, type AssetProfile } from '#lib/assetProfile.js'
  import { CANONICAL_COUNTRIES, countryDisplayName } from '#lib/countryNames.js'
  import { assetApi, type Asset, type AssetLookupResult, type Currency } from '#lib/services/api.js'
  import Modal from '#lib/components/ui/Modal.svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import Field from '#lib/components/ui/Field.svelte'
  import Input from '#lib/components/ui/Input.svelte'
  import Select from '#lib/components/ui/Select.svelte'
  import AssetSearchAutocomplete from './AssetSearchAutocomplete.svelte'
  import CurrencySelect from './CurrencySelect.svelte'

  /** Asset creation dialog (EPIC E.3): owns the form, the API call and the
   * toast; the rendered fields come from the per-type `assetProfile`. */
  let {
    open = $bindable(false),
    currencies = [],
    onsuccess = undefined,
  }: {
    open?: boolean
    currencies?: Currency[]
    onsuccess?: () => void
  } = $props()

  /** Icon of each asset type in the create-modal radio cards. */
  const TYPE_ICONS = {
    stock: ChartCandlestick,
    etf: Layers,
    bond: ScrollText,
    mutual_fund: PiggyBank,
    crypto: Bitcoin,
    commodity: Coins,
    cash: Banknote,
    certificate: FileBadge,
  } as const

  const uid = $props.id()

  const defaultForm = () => ({
    ticker: '',
    isin: '',
    name: '',
    // No preselected type: the graphic radio cards (below) must be chosen first.
    type: '' as string,
    currency: 'USD',
    country: '',
    exchange: '',
    asset_class: '',
    sector: '',
    industry: '',
    price_source: 'yahoo',
    quote_divisor: '1',
    maturity_date: '',
    issuer: '',
    issuer_country: '',
    attributes: {} as Record<string, string>,
  })

  /** Canonical country codes ordered by display name — same list and ordering
   *  as the Data tab's country selects. */
  const sortedCountryCodes = [...CANONICAL_COUNTRIES].sort((a, b) =>
    countryDisplayName(a).localeCompare(countryDisplayName(b), 'it'),
  )

  let form = $state(defaultForm())
  let saving = $state(false)
  // Bumped whenever the draft changes under an in-flight lookup, so a late
  // meta response never overrides the type the user picked meanwhile.
  let selection = 0

  const profile = $derived(assetProfile(form.type))
  /** Type is picked with the graphic radio cards instead of a select field. */
  const fields = $derived(profile.createFields.filter((f) => f.control !== 'asset-type'))

  const defaultCurrency = $derived(
    currencies.find((c) => c.code === 'USD')?.code ?? currencies[0]?.code ?? 'USD',
  )

  function resetForm(): void {
    selection++
    form = { ...defaultForm(), currency: defaultCurrency }
  }

  /** Attribute keys of a profile, all empty: the form always binds to a
   * defined string, never to a missing key. */
  function emptyAttributes(p: AssetProfile): Record<string, string> {
    return Object.fromEntries((p.createAttributes ?? []).map((a) => [a.key, '']))
  }

  /** Switching type re-applies the new profile's defaults and drops the
   * fields it does not render, so a hidden stale value is never saved. */
  function setType(type: string): void {
    selection++
    form.type = type
    const p = assetProfile(type)
    form.price_source = p.defaultPriceSource
    form.quote_divisor = p.defaultQuoteDivisor
    const keys = new Set(p.createFields.map((f) => f.key))
    if (!keys.has('exchange')) form.exchange = ''
    if (!keys.has('issuer')) form.issuer = ''
    if (!keys.has('issuer_country')) form.issuer_country = ''
    if (!keys.has('maturity_date')) form.maturity_date = ''
    const attrs = emptyAttributes(p)
    for (const key of Object.keys(attrs)) attrs[key] = String(form.attributes[key] ?? '')
    form.attributes = attrs
  }

  /** Fixed income has no ticker: once the ISIN is known, offer it as code. */
  function maybeCodeFromIsin(key: string): void {
    if (key === 'isin' && profile.codeFromIsin && form.ticker.trim() === '') {
      form.ticker = form.isin.trim()
    }
  }

  // Closing (or first mount while still closed) clears the draft form.
  $effect(() => {
    if (!open) resetForm()
  })

  async function handleSelect(result: AssetLookupResult): Promise<void> {
    const request = ++selection
    const next = result.type || 'stock'
    form = {
      ticker: result.ticker,
      isin: form.isin,
      name: result.name || '',
      type: next,
      currency: result.currency || 'USD',
      country: '',
      exchange: result.exchange || '',
      asset_class: '',
      sector: '',
      industry: '',
      // A Yahoo result is always quoted.
      price_source: 'yahoo',
      quote_divisor: assetProfile(next).defaultQuoteDivisor,
      maturity_date: '',
      issuer: '',
      issuer_country: '',
      attributes: emptyAttributes(assetProfile(next)),
    }

    try {
      const meta = await assetApi.meta(result.ticker)
      // The user changed the draft while the lookup was running: keep theirs.
      if (request !== selection) return
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
      // The backend decodes `maturity_date` as a nullable time: send the
      // picked date at UTC midnight or omit it, never an empty string.
      const payload: Partial<Asset> = {
        ...form,
        quote_divisor: form.quote_divisor || '1',
        attributes: Object.fromEntries(
          Object.entries(form.attributes)
            // Number inputs write numbers (and null when cleared): normalize
            // before trimming and never send empty extras.
            .map(([key, value]) => [key, String(value ?? '').trim()])
            .filter(([, value]) => value !== ''),
        ),
      }
      if (form.maturity_date) payload.maturity_date = `${form.maturity_date}T00:00:00Z`
      else delete payload.maturity_date
      await assetApi.create(payload)
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
  <Button onclick={submit} disabled={saving || !form.type || !profile.required.every((key) => form[key].trim() !== '')} loading={saving}>{t('common.save')}</Button>
{/snippet}

<Modal
  bind:open
  title={t('asset.newTitle')}
  description={form.type ? t(createHint(form.type)) : t('asset.chooseTypeHint')}
  size="lg"
  {footer}
  dismissible={!saving}
>
  <fieldset>
    <legend class="mb-2 block text-sm font-medium text-foreground">{t('asset.factType')}</legend>
    <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
      {#each ASSET_TYPES as type (type)}
        {@const Icon = TYPE_ICONS[type]}
        <!-- One wrapper per card: `peer-focus-visible` uses the sibling
             combinator, so without it a focused radio would ring every
             later card. -->
        <div>
          <input
            id={`${uid}-type-${type}`}
            class="peer sr-only"
            type="radio"
            name={`${uid}-asset-type`}
            value={type}
            checked={form.type === type}
            onchange={() => setType(type)}
          />
          <label
            for={`${uid}-type-${type}`}
            class="flex h-full cursor-pointer flex-col items-center justify-center gap-1.5 rounded-control border px-2 py-3 transition-colors duration-fast ease-standard peer-focus-visible:ring-2 peer-focus-visible:ring-ring peer-focus-visible:ring-offset-2 peer-focus-visible:ring-offset-background {form.type === type ? 'border-accent bg-accent/10 text-foreground' : 'border-border bg-surface text-muted-foreground hover:bg-muted hover:text-foreground'}"
          >
            <Icon class="h-5 w-5" aria-hidden="true" />
            <span class="text-center text-sm font-medium leading-tight">{assetTypeLabel(type)}</span>
          </label>
        </div>
      {/each}
    </div>
  </fieldset>
  {#if form.type}
  <div class="mt-4 grid gap-3 sm:grid-cols-2">
    {#each fields as field (field.key)}
      {#if field.control === 'ticker'}
        <Field label={t(field.label)} for={`${uid}-ticker`} hint={field.hint ? t(field.hint) : undefined}>
          <AssetSearchAutocomplete
            inputId={`${uid}-ticker`}
            bind:ticker={form.ticker}
            onselect={handleSelect}
          />
        </Field>
      {:else if field.control === 'currency'}
        <Field label={t(field.label)} hint={field.hint ? t(field.hint) : undefined}>
          <CurrencySelect bind:value={form.currency} {currencies} />
        </Field>
      {:else if field.control === 'price-source'}
        <Field label={t(field.label)} hint={field.hint ? t(field.hint) : undefined}>
          <Select bind:value={form.price_source}>
            <option value="yahoo">{t('asset.priceSourceYahoo')}</option>
            <option value="manual">{t('asset.priceSourceManual')}</option>
            <option value="none">{t('asset.priceSourceNone')}</option>
          </Select>
        </Field>
      {:else if field.control === 'country'}
        <Field label={t(field.label)} hint={field.hint ? t(field.hint) : undefined}>
          <Select bind:value={form[field.key]}>
            <option value="">{t('asset.countryNone')}</option>
            {#if form[field.key] && !CANONICAL_COUNTRIES.includes(form[field.key])}
              <!-- Legacy free-text value that is not a canonical code: kept
                   selectable so it is never silently erased. -->
              <option value={form[field.key]}>{form[field.key]}</option>
            {/if}
            {#each sortedCountryCodes as code (code)}
              <option value={code}>{code} — {countryDisplayName(code)}</option>
            {/each}
          </Select>
        </Field>
      {:else if field.control === 'date'}
        <Field label={t(field.label)} hint={field.hint ? t(field.hint) : undefined}>
          <Input type="date" bind:value={form[field.key]} />
        </Field>
      {:else}
        <Field label={t(field.label)} hint={field.hint ? t(field.hint) : undefined}>
          <Input
            bind:value={form[field.key]}
            placeholder={field.placeholder ? t(field.placeholder) : undefined}
            onblur={() => maybeCodeFromIsin(field.key)}
          />
        </Field>
      {/if}
    {/each}
    {#each profile.createAttributes ?? [] as attr (attr.key)}
      <Field label={t(attr.label)}>
        <!-- One-way value + input handler: an attribute key may be missing for
             the first render after a type change, and binding that undefined
             would throw `props_invalid_value` in Svelte 5. -->
        <Input
          type={attr.kind === 'number' ? 'number' : 'text'}
          value={form.attributes[attr.key] ?? ''}
          oninput={(e) => (form.attributes[attr.key] = e.currentTarget.value)}
        />
      </Field>
    {/each}
  </div>
  {/if}
</Modal>
