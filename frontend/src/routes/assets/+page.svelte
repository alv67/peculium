<script lang="ts">
  import { onMount } from 'svelte'
  import { resolve } from '$app/paths'
  import { toast } from '$lib/stores/toast.svelte'
  import { t } from '$lib/i18n/index.svelte'
  import { assetApi, settingsApi, type Asset, type Currency } from '$lib/services/api'
  import { formatCurrency } from '$lib/format'
  import { Plus, Trash2 } from 'lucide-svelte'
  import Badge from '$lib/components/ui/Badge.svelte'
  import Button from '$lib/components/ui/Button.svelte'
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte'
  import Table from '$lib/components/ui/Table.svelte'
  import THead from '$lib/components/ui/THead.svelte'
  import TBody from '$lib/components/ui/TBody.svelte'
  import Tr from '$lib/components/ui/Tr.svelte'
  import Th from '$lib/components/ui/Th.svelte'
  import Td from '$lib/components/ui/Td.svelte'
  import CreateAssetModal from '$lib/components/domain/CreateAssetModal.svelte'

  let showCreate = $state(false)
  let assets = $state<Asset[] | null>(null)
  let loading = $state(true)
  let currencies = $state<Currency[]>([])

  // Delete-confirmation dialog (D.2): replaces the native confirm().
  let showDeleteDialog = $state(false)
  let deleting = $state(false)
  let assetToDelete = $state<{ id: string; ticker: string } | null>(null)

  onMount(async () => {
    try {
      const [assetList, curList] = await Promise.all([assetApi.list(), settingsApi.listCurrencies()])
      assets = assetList
      currencies = curList.currencies
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('asset.loadFailed')
      toast.error(message)
    } finally {
      loading = false
    }
  })

  async function reloadAssets(): Promise<void> {
    assets = await assetApi.list()
  }

  function requestDeleteAsset(id: string, ticker: string): void {
    assetToDelete = { id, ticker }
    showDeleteDialog = true
  }

  async function deleteAsset(): Promise<void> {
    if (!assetToDelete) return
    const { id } = assetToDelete
    deleting = true
    try {
      await assetApi.remove(id)
      assets = await assetApi.list()
      toast.success(t('asset.deleted'))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.deleteFailed')
      toast.error(message)
    } finally {
      deleting = false
    }
  }
</script>

<div class="p-6">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-2xl font-bold">{t('nav.assets')}</h1>
    <Button onclick={() => (showCreate = true)}>
      <Plus class="h-4 w-4" />
      {t('asset.add')}
    </Button>
  </div>

  {#if loading}
    <p class="text-muted-foreground">{t('common.loading')}</p>
  {:else}
    <div class="overflow-x-auto">
      <Table aria-label={t('nav.assets')}>
        <THead>
          <Tr>
            <Th>{t('positions.colTicker')}</Th>
            <Th>{t('chartView.colName')}</Th>
            <Th>{t('asset.factType')}</Th>
            <Th>{t('asset.factCurrency')}</Th>
            <Th>{t('asset.colCountry')}</Th>
            <Th align="right">{t('asset.colLastClose')}</Th>
            <Th align="right">{t('common.colActions')}</Th>
          </Tr>
        </THead>
        <TBody>
          {#each assets ?? [] as a (a.id)}
            <Tr>
              <Td class="font-medium">
                <a href={resolve(`/assets/${a.id}`)} class="text-accent-text hover:underline">{a.ticker}</a>
              </Td>
              <Td class="text-muted-foreground">{a.name}</Td>
              <Td><Badge>{a.type}</Badge></Td>
              <Td>{a.currency}</Td>
              <Td>{a.country || '-'}</Td>
              <Td align="right">
                {#if a.last_close}
                  <span class="font-semibold">{formatCurrency(a.last_close, a.currency)}</span>
                  {#if a.last_close_date}
                    <span class="block text-xs font-normal text-muted-foreground">
                      {new Date(a.last_close_date).toLocaleDateString()}
                    </span>
                  {/if}
                {:else}
                  <span class="text-muted-foreground">—</span>
                {/if}
              </Td>
              <Td align="right">
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t('asset.deleteNamed', { ticker: a.ticker })}
                  onclick={() => requestDeleteAsset(a.id, a.ticker)}
                >
                  <Trash2 class="h-4 w-4" />
                </Button>
              </Td>
            </Tr>
          {/each}
        </TBody>
      </Table>
    </div>
  {/if}
</div>

<CreateAssetModal bind:open={showCreate} {currencies} onsuccess={reloadAssets} />

<ConfirmDialog
  bind:open={showDeleteDialog}
  variant="danger"
  title={t('asset.delete')}
  message={t('asset.deleteQuestion', { ticker: assetToDelete?.ticker ?? '' })}
  confirmLabel={t('common.delete')}
  cancelLabel={t('common.cancel')}
  loading={deleting}
  onconfirm={deleteAsset}
/>
