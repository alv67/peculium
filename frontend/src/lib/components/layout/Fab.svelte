<script lang="ts">
  import { HandCoins, Plus, RefreshCw, SquarePen } from 'lucide-svelte'
  import { goto } from '$app/navigation'
  import { resolve } from '$app/paths'
  import { t } from '#lib/i18n/index.svelte.js'
  import { portfolioApi } from '#lib/services/api.js'
  import { refreshPrices } from '#lib/stores/priceRefresh.svelte.js'
  import QuickActionSheet, { type QuickAction } from './QuickActionSheet.svelte'

  /**
   * Phone-only quick-actions Fab (EPIC K.2, decision D2): the single mobile
   * entry point for "record a transaction / add an asset / refresh prices",
   * pinned bottom-right *above* the bottom nav (72px offset vs its 56px +
   * safe area — they never overlap). It opens the `QuickActionSheet`, so it
   * carries `aria-haspopup="dialog"` and stays 56px wide (≥ 44px target).
   *
   * The action list is the single source of truth here (the sheet only
   * renders it); the Fab itself hides when nothing is enabled — a guard that
   * is defensive today (the nav/API actions are always available) but keeps
   * the D2 rule honest as actions become permission- or data-scoped. While
   * the sheet is open the Fab remains mounted under its backdrop (z-20 vs
   * z-30): hiding it would yank the focus-trap's restore anchor away.
   *
   * K.2 actions go through navigation or the API — the global forms land
   * with K.4. *Enter price* is the EPIC J.1 reserved slot: rendered disabled
   * with a muted "coming soon" (spec §6.3, same philosophy as the Activity
   * slot in "More").
   */
  let sheetOpen = $state(false)

  const actions: QuickAction[] = [
    {
      id: 'add-transaction',
      labelKey: 'quickActions.addTransaction',
      hintKey: 'quickActions.addTransactionHint',
      icon: HandCoins,
      // No global form yet: jump straight to the single portfolio when the
      // vault has exactly one, otherwise to the list so the user can pick
      // (spec §6.4 "portfolio picker step"). On failure the list is the
      // safest destination — it shows its own error state.
      onSelect: async () => {
        try {
          const portfolios = await portfolioApi.list();

          await goto(portfolios.length === 1
            ? resolve(`portfolios/${portfolios[0].id}`)
            : resolve('portfolios'));
        } catch {
          await goto(resolve('portfolios'));
        }
      },
    },
    {
      id: 'add-asset',
      labelKey: 'quickActions.addAsset',
      hintKey: 'quickActions.addAssetHint',
      icon: Plus,
      // The assets page owns the Yahoo-lookup creation flow.
      onSelect: () => goto(resolve('assets'))
    },
    {
      id: 'refresh-prices',
      labelKey: 'quickActions.refreshPrices',
      hintKey: 'quickActions.refreshPricesHint',
      icon: RefreshCw,
      // Manual session refresh through the shared store path: toasts + the
      // global stamp/strip update for everyone, the sheet awaits the shared
      // promise so its spinner tracks the real in-flight POST (the POST
      // clears the GET cache, so the next page fetch sees the new prices).
      onSelect: async () => {
        await refreshPrices({ announceSuccess: true })
      },
    },
    {
      id: 'enter-price',
      labelKey: 'quickActions.enterPrice',
      icon: SquarePen,
      disabled: true // EPIC J.1 has not landed yet — reserved slot.
    }
  ];

  const hasEnabledAction = $derived(actions.some((action) => !action.disabled))
</script>

{#if hasEnabledAction}
  <button
    type="button"
    class="focus-ring fixed bottom-[calc(4.5rem_+_env(safe-area-inset-bottom))] right-4 z-20 flex h-14 w-14 items-center justify-center rounded-full bg-accent text-accent-foreground shadow-raised transition-[background-color,transform] duration-base ease-standard hover:bg-accent-hover active:scale-95 sm:hidden"
    aria-label={t('fab.open')}
    aria-haspopup="dialog"
    aria-expanded={sheetOpen}
    onclick={() => sheetOpen = true}
  ><Plus class="h-6 w-6" /></button>
{/if}

<QuickActionSheet
  open={sheetOpen}
  actions={actions}
  onClose={() => sheetOpen = false}
/>
