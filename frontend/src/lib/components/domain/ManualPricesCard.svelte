<script lang="ts">
  import { onMount } from 'svelte'
  import { t } from '#lib/i18n/index.svelte.js'
  import { toast } from '#lib/stores/toast.svelte.js'
  import { assetApi, type ManualPrice } from '#lib/services/api.js'
  import { formatCurrency } from '#lib/format.js'
  import { Pencil, Trash2 } from 'lucide-svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import ConfirmDialog from '#lib/components/ui/ConfirmDialog.svelte'
  import Field from '#lib/components/ui/Field.svelte'
  import Input from '#lib/components/ui/Input.svelte'
  import Table from '#lib/components/ui/Table.svelte'
  import THead from '#lib/components/ui/THead.svelte'
  import TBody from '#lib/components/ui/TBody.svelte'
  import Tr from '#lib/components/ui/Tr.svelte'
  import Th from '#lib/components/ui/Th.svelte'
  import Td from '#lib/components/ui/Td.svelte'

  /**
   * Manual prices section (EPIC J.1, #105): record dated closes by hand for
   * assets without an automatic feed (bonds, OTC certificates, pension funds,
   * time deposits). Mounted by the Data tab only when the asset's
   * `price_source` is `manual` or `none` — manual bars exist only for those.
   *
   * Self-contained like the other domain components that own their endpoint
   * pair: the compact insert form and the history table (newest date first)
   * run `GET/PUT/DELETE /assets/{id}/prices/manual`. The PUT upserts by date,
   * so "edit" just prefills the form with the row's values and saves the same
   * date. Client-side validation mirrors the backend guards (no future dates,
   * close > 0); any backend 400 message surfaces through the error toast.
   * After every successful write the table reloads and `refresh` (the shell's
   * `refetchAssetData`) updates the quote strip and the Overview price chart,
   * which read the same history.
   */
  let {
    assetId,
    currency,
    refresh,
  }: {
    assetId: string
    currency: string
    refresh: () => void
  } = $props()

  // Same UTC-today recipe the transaction form uses for its date default;
  // matches the backend's (UTC) future-date guard.
  const today = new Date().toISOString().split('T')[0]

  let prices = $state<ManualPrice[]>([])
  let loading = $state(true)
  let saving = $state(false)
  // Date of the row currently loaded into the form for an edit; `null` means
  // the form is a fresh insert. The upsert needs no separate PUT path.
  let editingDate = $state<string | null>(null)
  let form = $state({ date: today, close: '', volume: '' })
  let errors = $state({ date: '', close: '' })
  let showDelete = $state(false)
  let deleteRow = $state<ManualPrice | null>(null)
  let deleting = $state(false)

  onMount(loadPrices)

  async function loadPrices(): Promise<void> {
    loading = true
    try {
      prices = await assetApi.manualPrices(assetId)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.manualLoadFailed')
      toast.error(message)
    } finally {
      loading = false
    }
  }

  function validate(): boolean {
    errors = { date: '', close: '' }
    if (!form.date) errors.date = t('asset.manualDateRequired')
    else if (form.date > today) errors.date = t('asset.manualDateFuture')
    const close = Number(form.close)
    if (!form.close || !Number.isFinite(close) || close <= 0) {
      errors.close = t('asset.manualCloseInvalid')
    }
    return !errors.date && !errors.close
  }

  function resetForm(): void {
    editingDate = null
    form = { date: today, close: '', volume: '' }
    errors = { date: '', close: '' }
  }

  function startEdit(p: ManualPrice): void {
    editingDate = p.date
    form = { date: p.date, close: p.close, volume: p.volume === null ? '' : String(p.volume) }
    errors = { date: '', close: '' }
  }

  async function submit(): Promise<void> {
    if (!validate()) return
    saving = true
    try {
      // `volume` is omitted (not null) when the field is empty — the PUT
      // body carries it only when provided.
      const body: { date: string; close: string; volume?: number } = {
        date: form.date,
        close: form.close,
      }
      if (form.volume !== '') body.volume = Number(form.volume)
      await assetApi.saveManualPrice(assetId, body)
      toast.success(t('asset.manualSaved'))
      resetForm()
      await loadPrices()
      refresh()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.saveFailed')
      toast.error(message)
    } finally {
      saving = false
    }
  }

  function requestDelete(p: ManualPrice): void {
    deleteRow = p
    showDelete = true
  }

  async function confirmDelete(): Promise<void> {
    if (!deleteRow) return
    deleting = true
    try {
      await assetApi.deleteManualPrice(assetId, deleteRow.date)
      toast.success(t('asset.manualDeleted'))
      if (editingDate === deleteRow.date) resetForm()
      await loadPrices()
      refresh()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.deleteFailed')
      toast.error(message)
    } finally {
      deleting = false
    }
  }
</script>

<div class="mb-6 rounded-card border-border bg-surface p-4 shadow-card">
  <h2 class="mb-1 font-semibold">{t('asset.manualPrice')}</h2>
  <p class="mb-4 text-sm text-muted-foreground">{t('asset.manualPriceHint')}</p>

  <div class="mb-4 flex flex-wrap items-start gap-3">
    <Field label={t('common.colDate')} class="w-40" error={errors.date || undefined}>
      <Input
        type="date"
        max={today}
        bind:value={form.date}
        error={errors.date || undefined}
        oninput={() => (errors.date = '')}
      />
    </Field>
    <Field label={t('common.colPrice')} class="w-40" error={errors.close || undefined}>
      <Input
        type="number"
        step="any"
        min="0"
        bind:value={form.close}
        error={errors.close || undefined}
        oninput={() => (errors.close = '')}
      />
    </Field>
    <Field label={`${t('asset.manualColVolume')} (${t('common.optional')})`} class="w-32">
      <Input type="number" step="1" min="0" bind:value={form.volume} />
    </Field>
    <div class="flex items-center gap-2 self-end">
      <Button disabled={saving} onclick={submit}>
        {editingDate ? t('common.save') : t('asset.manualAdd')}
      </Button>
      {#if editingDate}
        <Button variant="secondary" onclick={resetForm}>{t('common.cancel')}</Button>
      {/if}
    </div>
  </div>

  {#if loading}
    <p class="text-sm text-muted-foreground">{t('common.loading')}</p>
  {:else if prices.length === 0}
    <p class="text-sm text-muted-foreground">{t('asset.manualEmpty')}</p>
  {:else}
    <div class="overflow-x-auto">
      <Table aria-label={t('asset.manualPrice')}>
        <THead>
          <Tr>
            <Th>{t('common.colDate')}</Th>
            <Th align="right">{t('common.colPrice')}</Th>
            <Th align="right">{t('asset.manualColVolume')}</Th>
            <Th align="right">{t('common.colActions')}</Th>
          </Tr>
        </THead>
        <TBody>
          {#each prices as p (p.date)}
            <Tr>
              <Td>{new Date(p.date).toLocaleDateString()}</Td>
              <Td align="right">{formatCurrency(p.close, currency)}</Td>
              <Td align="right">{p.volume ?? '—'}</Td>
              <Td align="right">
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t('asset.manualEditNamed', { date: p.date })}
                  onclick={() => startEdit(p)}
                >
                  <Pencil class="h-4 w-4" aria-hidden="true" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t('asset.manualDeleteNamed', { date: p.date })}
                  onclick={() => requestDelete(p)}
                >
                  <Trash2 class="h-4 w-4" aria-hidden="true" />
                </Button>
              </Td>
            </Tr>
          {/each}
        </TBody>
      </Table>
    </div>
  {/if}
</div>

<ConfirmDialog
  bind:open={showDelete}
  variant="danger"
  title={t('asset.manualDelete')}
  message={t('asset.manualDeleteConfirm', { date: deleteRow?.date ?? '' })}
  confirmLabel={t('common.delete')}
  cancelLabel={t('common.cancel')}
  loading={deleting}
  onconfirm={confirmDelete}
/>
