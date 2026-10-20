<script lang="ts">
  import { toast } from '#lib/stores/toast.svelte.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import { transactionApi, type Asset, type Transaction } from '#lib/services/api.js'
  import { formatCurrency } from '#lib/format.js'
  import { viewport } from '#lib/stores/viewport.svelte.js'
  import Modal from '#lib/components/ui/Modal.svelte'
  import Sheet from '#lib/components/ui/Sheet.svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import Field from '#lib/components/ui/Field.svelte'
  import Input from '#lib/components/ui/Input.svelte'
  import Select from '#lib/components/ui/Select.svelte'
  import Textarea from '#lib/components/ui/Textarea.svelte'
  import AssetCombobox from './AssetCombobox.svelte'

  /**
   * Add/edit/delete transaction form (EPIC E.2): owns the form state, inline
   * validation, API calls and toasts so the portfolio page only toggles
   * `open` and refreshes via `onsuccess`.
   *
   * EPIC K.4c changed two things about this dialog, nothing else:
   * - **Responsive container (D4, spec §6.2/§6.4):** the same `formBody` +
   *   `footer` snippets render into the classic `ui/Modal` from `sm` up and
   *   into the K.1c `ui/Sheet` (bottom sheet) on phones, chosen by the
   *   reactive `viewport` store. Fields, validation and the live total are
   *   identical in both containers; the sheet's `onClose` mirrors the
   *   modal's busy-state `dismissible` gate.
   * - **Undo over confirm (D11, spec §6.4/§8.6):** Delete inside edit mode
   *   removes the transaction immediately and offers a 5 s "Undo" action in
   *   the resulting toast instead of the old `ConfirmDialog`. Undo
   *   re-POSTs the captured payload via `transactionApi.create` — which
   *   yields a NEW id (see `deleteTransaction`) — and re-enters the regular
   *   post-mutation refetch. `ConfirmDialog` remains for the irreducible
   *   destructive actions (portfolio/asset delete, import overwrite).
   */
  let {
    open = $bindable(false),
    portfolioId,
    assets = [],
    currency = 'USD',
    editing = null,
    onsuccess = undefined,
  }: {
    open?: boolean
    portfolioId: string
    assets?: Asset[]
    currency?: string
    editing?: Transaction | null
    onsuccess?: () => void
  } = $props()

  const uid = $props.id()

  /**
   * `ui/Input type="number"` coerces the bound value to `number` (or
   * `null`/`undefined` when empty) even though `TxForm` fields are typed
   * `string`. Normalize through this helper before any string operation.
   */
  const toStr = (v: unknown): string => (v === null || v === undefined ? '' : String(v))

  interface TxForm {
    asset_id: string
    /** Kept as `string` for the `ui/Select` binding; cast back on submit. */
    type: string
    quantity: string
    price: string
    date: string
    fees: string
    notes: string
  }

  interface ErrorBag {
    asset: string
    quantity: string
    price: string
    amount: string
    date: string
  }

  const defaultForm = (): TxForm => ({
    asset_id: '',
    type: 'buy',
    quantity: '',
    price: '',
    date: new Date().toISOString().split('T')[0],
    fees: '0',
    notes: '',
  })

  const emptyErrors = (): ErrorBag => ({
    asset: '',
    quantity: '',
    price: '',
    amount: '',
    date: '',
  })

  let form = $state(defaultForm())
  let dividendAmount = $state('')
  let errors = $state(emptyErrors())
  let saving = $state(false)
  let deleting = $state(false)

  const isDividend = $derived(form.type === 'dividend')
  const total = $derived(
    isDividend ? Number(dividendAmount) : Number(form.quantity) * Number(form.price),
  )
  const totalEntered = $derived(
    isDividend
      ? toStr(dividendAmount) !== '' && Number.isFinite(total)
      : toStr(form.quantity) !== '' && toStr(form.price) !== '' && Number.isFinite(total),
  )
  const totalText = $derived(totalEntered ? formatCurrency(total, currency) : '—')

  // Same title in either container; now routed through `t()` because the K.4c
  // bottom sheet promotes it to the phone screen's most visible label (D1).
  const title = $derived(editing ? t('tx.titleEdit') : t('tx.titleNew'))

  // Prefill on open, clear the draft on close (and while never opened).
  $effect(() => {
    if (open && editing) {
      form = {
        asset_id: editing.asset_id,
        type: editing.type,
        quantity: editing.quantity,
        price: editing.price,
        date: new Date(editing.date).toISOString().split('T')[0],
        fees: editing.fees || '0',
        notes: editing.notes || '',
      }
      dividendAmount = editing.type === 'dividend' ? String(Number(editing.price) * Number(editing.quantity)) : ''
    } else {
      form = defaultForm()
      dividendAmount = ''
    }
    errors = emptyErrors()
  })

  // The combobox selection is the "input" for the asset field.
  $effect(() => {
    if (form.asset_id && errors.asset) errors.asset = ''
  })

  function validate(): ErrorBag {
    const next = emptyErrors()
    if (!form.asset_id) next.asset = t('tx.selectAsset')
    if (!form.date) next.date = t('tx.dateRequired')
    if (isDividend) {
      if (!(Number(dividendAmount) > 0)) next.amount = t('tx.amountRequired')
    } else {
      if (!(Number(form.quantity) > 0)) next.quantity = t('tx.quantityRequired')
      if (toStr(form.price) === '' || !(Number(form.price) >= 0)) {
        next.price = t('tx.priceRequired')
      }
    }
    return next
  }

  async function submit(): Promise<void> {
    errors = validate()
    if (Object.values(errors).some(Boolean)) return
    saving = true
    const payload: Partial<Transaction> = {
      asset_id: form.asset_id,
      type: form.type as Transaction['type'],
      quantity: isDividend ? '1' : toStr(form.quantity),
      price: isDividend ? toStr(dividendAmount) : toStr(form.price),
      date: new Date(form.date).toISOString(),
      fees: toStr(form.fees) || '0',
      notes: form.notes,
    }
    try {
      if (editing) {
        await transactionApi.update(editing.id, payload)
      } else {
        await transactionApi.create(portfolioId, payload)
      }
      toast.success(editing ? t('tx.updated') : t('tx.added'))
      open = false
      onsuccess?.()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.saveFailed')
      toast.error(message)
    } finally {
      saving = false
    }
  }

  /** How long the undo window stays open after a delete (spec §6.4, D11). */
  const UNDO_WINDOW_MS = 5000

  /**
   * Delete without confirmation (D11), then offer Undo. The full row is
   * snapshotted FIRST because undo cannot PATCH a gone record: it re-POSTs
   * the same payload to the owning portfolio, which yields a NEW `id` (and
   * a fresh `created_at`) for the restored transaction. Accepted at family
   * scale: no view links to a transaction id across deletes, and the
   * original date/type/amounts — the parts analytics order and aggregate
   * by — are carried verbatim.
   */
  async function deleteTransaction(): Promise<void> {
    if (!editing) return
    deleting = true
    const snapshot: Partial<Transaction> = {
      asset_id: editing.asset_id,
      type: editing.type,
      quantity: editing.quantity,
      price: editing.price,
      fees: editing.fees || '0',
      date: editing.date,
      notes: editing.notes,
    }
    const targetPortfolioId = editing.portfolio_id || portfolioId
    try {
      await transactionApi.remove(editing.id)
      open = false
      onsuccess?.()
      toast.success(t('tx.deleted'), {
        duration: UNDO_WINDOW_MS,
        action: {
          label: t('tx.undo'),
          onclick: () => void undoDelete(targetPortfolioId, snapshot),
        },
      })
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.deleteFailed')
      toast.error(message)
    } finally {
      deleting = false
    }
  }

  /** Undo action body: recreate the deleted row and rerun the standard
   * post-mutation refetch, so the (filtered) list, KPIs, allocations and
   * performance all come back in sync through the layout's one path. */
  async function undoDelete(targetPortfolioId: string, snapshot: Partial<Transaction>): Promise<void> {
    try {
      await transactionApi.create(targetPortfolioId, snapshot)
      onsuccess?.()
    } catch {
      toast.error(t('tx.undoFailed'))
    }
  }

  /** Sheet close gate: Esc/backdrop/✕ stay inert while a request is in
   * flight, mirroring the Modal's `dismissible` prop in the other branch. */
  function requestClose(): void {
    if (saving || deleting) return
    open = false
  }
</script>

{#snippet footer()}
  <div class="flex flex-wrap items-center justify-end gap-2">
    <Button variant="secondary" onclick={requestClose} disabled={saving || deleting}>
      {t('common.cancel')}
    </Button>
    {#if editing}
      <!-- D11: one click deletes, the toast's Undo (5 s) puts it back. -->
      <Button variant="danger" onclick={() => void deleteTransaction()} disabled={saving || deleting} loading={deleting}>
        {t('common.delete')}
      </Button>
    {/if}
    <Button onclick={submit} loading={saving} disabled={deleting}>
      {editing ? t('common.saveChanges') : t('common.save')}
    </Button>
  </div>
{/snippet}

{#snippet formBody()}
  <div class="grid gap-3 sm:grid-cols-2">
    <Field label={t('common.colAsset')} for={`${uid}-asset`} error={errors.asset || undefined}>
      <AssetCombobox
        inputId={`${uid}-asset`}
        bind:value={form.asset_id}
        {assets}
        error={errors.asset || undefined}
        disabled={saving || deleting}
      />
    </Field>
    <Field label={t('common.colType')}>
      <Select
        bind:value={form.type}
        oninput={() => {
          errors.quantity = ''
          errors.price = ''
          errors.amount = ''
        }}
      >
        <option value="buy">{t('activity.typeBuy')}</option>
        <option value="sell">{t('activity.typeSell')}</option>
        <option value="dividend">{t('activity.typeDividend')}</option>
      </Select>
    </Field>
    {#if isDividend}
      <Field label={t('tx.amount')} error={errors.amount || undefined}>
        <Input
          type="number"
          step="0.01"
          min="0"
          placeholder={t('tx.amount')}
          bind:value={dividendAmount}
          error={errors.amount || undefined}
          oninput={() => (errors.amount = '')}
        />
      </Field>
    {:else}
      <Field label={t('tx.quantity')} error={errors.quantity || undefined}>
        <Input
          type="number"
          step="any"
          placeholder={t('tx.quantity')}
          bind:value={form.quantity}
          error={errors.quantity || undefined}
          oninput={() => (errors.quantity = '')}
        />
      </Field>
      <Field label={t('common.colPrice')} error={errors.price || undefined}>
        <Input
          type="number"
          step="0.01"
          placeholder={t('common.colPrice')}
          bind:value={form.price}
          error={errors.price || undefined}
          oninput={() => (errors.price = '')}
        />
      </Field>
    {/if}
    <Field label={t('common.colDate')} error={errors.date || undefined}>
      <Input
        type="date"
        bind:value={form.date}
        error={errors.date || undefined}
        oninput={() => (errors.date = '')}
      />
    </Field>
    <Field label={t('tx.fees')}>
      <Input type="number" step="0.01" placeholder={t('tx.fees')} bind:value={form.fees} />
    </Field>
    <Field label={t('tx.notes')} class="sm:col-span-2">
      <Textarea placeholder={t('tx.notes')} rows={2} bind:value={form.notes} />
    </Field>
  </div>

  <div class="mt-4 flex items-baseline justify-end gap-2 text-sm">
    <span class="text-muted-foreground">{t('common.colTotal')}</span>
    <span class="font-semibold tabular-nums text-foreground">{totalText}</span>
  </div>
{/snippet}

{#if viewport.isPhone}
  <!-- Phone < sm: bottom sheet (D4). Same snippets, same state — flipping
       containers (rotate/resize) only swaps the chrome under the form. -->
  <Sheet
    {open}
    onClose={requestClose}
    {title}
    closeLabel={t('common.close')}
    footer={footer}
  >
    {@render formBody()}
  </Sheet>
{:else}
  <Modal
    bind:open
    {title}
    size="lg"
    {footer}
    dismissible={!saving && !deleting}
  >
    {@render formBody()}
  </Modal>
{/if}
