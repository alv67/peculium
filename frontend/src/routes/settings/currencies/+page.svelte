<script lang="ts">
  import { onMount } from 'svelte'
  import { toast } from '#lib/stores/toast.svelte.js'
  import { settingsApi, type Currency } from '#lib/services/api.js'
  import { t } from '#lib/i18n/index.svelte.js'
  import { CURRENCIES } from '#lib/currencies.js'
  import { currencySymbol } from '#lib/format.js'
  import { Trash2 } from 'lucide-svelte'
  import SettingsTabs from '#lib/components/domain/SettingsTabs.svelte'
  import Button from '#lib/components/ui/Button.svelte'
  import Card from '#lib/components/ui/Card.svelte'
  import ConfirmDialog from '#lib/components/ui/ConfirmDialog.svelte'
  import Field from '#lib/components/ui/Field.svelte'
  import Input from '#lib/components/ui/Input.svelte'
  import Select from '#lib/components/ui/Select.svelte'
  import Table from '#lib/components/ui/Table.svelte'
  import THead from '#lib/components/ui/THead.svelte'
  import TBody from '#lib/components/ui/TBody.svelte'
  import Tr from '#lib/components/ui/Tr.svelte'
  import Th from '#lib/components/ui/Th.svelte'
  import Td from '#lib/components/ui/Td.svelte'

  let currencies = $state<Currency[]>([])
  let currenciesLoading = $state(true)
  let newCode = $state('')
  let newName = $state('')

  let availableCurrencies = $derived(CURRENCIES.filter((c) => !currencies.some((m) => m.code === c.code)))

  let addingCurrency = $state(false)
  let removingCode = $state('')

  let showDeleteDialog = $state(false)
  let currencyToDelete = $state('')

  onMount(async () => {
    try {
      currencies = await listCurrencies()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('currencies.loadFailed')
      toast.error(message)
    } finally {
      currenciesLoading = false
    }
  })

  async function listCurrencies(): Promise<Currency[]> {
    const res = await settingsApi.listCurrencies()
    return res.currencies
  }

  function errorStatus(err: unknown): number | undefined {
    return err instanceof Error && 'status' in err ? (err as Error & { status: number }).status : undefined
  }

  function selectCurrency(code: string): void {
    newCode = code
    const match = CURRENCIES.find((c) => c.code === code)
    if (match) newName = match.name
  }

  async function addCurrency(): Promise<void> {
    const code = newCode
    if (!code) return
    addingCurrency = true
    try {
      await settingsApi.addCurrency(code, newName.trim() || undefined)
      currencies = await listCurrencies()
      newCode = ''
      newName = ''
      toast.success(t('currencies.added', { code }))
    } catch (err: unknown) {
      const status = errorStatus(err)
      if (status === 422) {
        toast.error(t('currencies.conversionUnavailable', { code }))
      } else if (status === 409) {
        toast.error(t('currencies.alreadyPresent'))
      } else {
        toast.error(err instanceof Error ? err.message : t('currencies.addFailed'))
      }
    } finally {
      addingCurrency = false
    }
  }

  function requestDeleteCurrency(code: string): void {
    currencyToDelete = code
    showDeleteDialog = true
  }

  async function removeCurrency(): Promise<void> {
    const code = currencyToDelete
    if (!code) return
    removingCode = code
    try {
      await settingsApi.deleteCurrency(code)
      currencies = await listCurrencies()
      toast.success(t('currencies.removed', { code }))
    } catch (err: unknown) {
      const status = errorStatus(err)
      if (status === 409) {
        toast.error(t('currencies.inUse'))
      } else {
        toast.error(err instanceof Error ? err.message : t('currencies.removeFailed'))
      }
    } finally {
      removingCode = ''
    }
  }
</script>

<div class="p-6">
  <h1 class="mb-6 text-2xl font-bold">{t('nav.settings')}</h1>

  <SettingsTabs class="mb-6" />

  <Card class="max-w-2xl p-6">
    <h2 class="mb-4 font-semibold">{t('currencies.title')}</h2>

    {#if availableCurrencies.length > 0}
      <!-- Stacked full-width on phones (EPIC K bug-fix): the fixed `w-64`
           code picker + flex-1 name field + Add button overflowed 393px;
           from `sm` the row keeps its original inline shape. -->
      <div class="mb-4 flex flex-col items-stretch gap-3 sm:flex-row sm:items-end">
        <Field label={t('common.colCode')} class="w-full shrink-0 sm:w-64">
          <Select value={newCode} onchange={(e) => selectCurrency(e.currentTarget.value)}>
            <option value="" disabled>{t('currencies.select')}</option>
            {#each availableCurrencies as c (c.code)}
              <option value={c.code}>{c.code} — {c.name}</option>
            {/each}
          </Select>
        </Field>
        <Field label={t('chartView.colName')} class="w-full flex-1 sm:w-auto">
          <Input placeholder={t('currencies.namePlaceholder')} bind:value={newName} />
        </Field>
        <Button onclick={addCurrency} disabled={!newCode || addingCurrency} class="w-full sm:w-auto">
          {addingCurrency ? t('currencies.adding') : t('currencies.add')}
        </Button>
      </div>
    {:else}
      <p class="mb-4 text-sm text-muted-foreground">{t('currencies.allManaged')}</p>
    {/if}

    {#if currenciesLoading}
      <p class="text-muted-foreground">{t('common.loading')}</p>
    {:else if currencies.length === 0}
      <p class="text-sm text-muted-foreground">{t('currencies.empty')}</p>
    {:else}
      <Table>
        <THead>
          <Tr>
            <Th>{t('common.colCode')}</Th>
            <Th>{t('chartView.colName')}</Th>
            <Th align="right">{t('common.colActions')}</Th>
          </Tr>
        </THead>
        <TBody>
          {#each currencies as c (c.code)}
            <Tr>
              <Td class="font-medium">{c.code}</Td>
              <Td class="text-muted-foreground">
                {c.name || '—'} <span class="text-xs text-muted-foreground">{currencySymbol(c.code)}</span>
              </Td>
              <Td align="right">
                <Button
                  variant="ghost"
                  size="icon"
                  class="text-muted-foreground hover:text-negative"
                  aria-label={t('currencies.removeNamed', { code: c.code })}
                  title={t('currencies.remove')}
                  disabled={removingCode === c.code}
                  onclick={() => requestDeleteCurrency(c.code)}
                >
                  <Trash2 class="h-4 w-4" />
                </Button>
              </Td>
            </Tr>
          {/each}
        </TBody>
      </Table>
    {/if}
  </Card>
</div>

<ConfirmDialog
  bind:open={showDeleteDialog}
  variant="danger"
  title={t('currencies.deleteTitle')}
  message={t('currencies.deleteConfirm', { code: currencyToDelete })}
  confirmLabel={t('common.delete')}
  cancelLabel={t('common.cancel')}
  loading={showDeleteDialog && removingCode === currencyToDelete}
  onconfirm={removeCurrency}
/>
