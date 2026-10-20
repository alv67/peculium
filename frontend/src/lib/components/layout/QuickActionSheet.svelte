<script module lang="ts">
  import type { RefreshCw } from 'lucide-svelte'
  import type { MessageKey } from '#lib/i18n/index.svelte.js'

  /**
   * One row of the quick-actions sheet. The Fab owns the list (labels are
   * `MessageKey`s so the sheet re-renders on locale change, and `icon` is a
   * lucide component — `typeof RefreshCw` is the repo's convention for the
   * shared icon component type, cf. `SidebarNav`); `disabled` marks a
   * reserved slot (e.g. Enter price until EPIC J.1) and `onSelect` may be
   * async — the row shows a spinner and the sheet stays open until it
   * settles.
   */
  export interface QuickAction {
    id: string
    labelKey: MessageKey
    /** Muted explanation under the label (optional). */
    hintKey?: MessageKey
    icon: typeof RefreshCw
    disabled?: boolean
    onSelect?: () => Promise<void> | void
  }
</script>

<script lang="ts">
  import { t } from '#lib/i18n/index.svelte.js'
  import Button from '../ui/Button.svelte'
  import Sheet from '../ui/Sheet.svelte'

  /**
   * Quick-actions bottom sheet (EPIC K.2, decision D2): the list of global
   * tasks ("record a transaction", "add an asset", "refresh prices")
   * rendered inside the K.1c `ui/Sheet` primitive, opened by the Fab. In
   * this phase every action goes through navigation or the API — the global
   * create/transaction forms land with K.4 — so the sheet always collapses
   * once the action settles (record ≤ 2 taps from anywhere).
   *
   * Controlled like `Sheet` itself: the Fab owns `open`; closing paths (Esc,
   * backdrop, ✕) are delegated to `onClose`.
   */
  let {
    open,
    actions,
    onClose,
  }: {
    open: boolean
    actions: QuickAction[]
    onClose: () => void
  } = $props()

  // Id of the action currently running (drives its row's spinner and locks
  // the rest of the list while it settles).
  let pending = $state<string | null>(null)

  async function run(action: QuickAction): Promise<void> {
    if (action.disabled || pending !== null) return
    pending = action.id
    try {
      await action.onSelect?.()
    } finally {
      pending = null
      // Every K.2 action ends in a navigation or a toast, so the sheet
      // always closes afterwards (also on unexpected failures).
      onClose()
    }
  }
</script>

<Sheet {open} {onClose} title={t('quickActions.title')}>
  <div class="flex flex-col gap-1">
    {#each actions as action (action.id)}
      {@const Icon = action.icon}
      {@const hint = action.disabled ? 'quickActions.comingSoon' : action.hintKey}
      <Button
        variant="ghost"
        class="h-14 w-full justify-start"
        disabled={action.disabled || pending !== null}
        loading={pending === action.id}
        onclick={() => void run(action)}
      >
        <Icon class="h-5 w-5 shrink-0 text-accent-text" />
        <span class="flex min-w-0 flex-1 flex-col items-start">
          <span class="text-sm font-medium">{t(action.labelKey)}</span>
          {#if hint}
            <span class="text-xs font-normal text-muted-foreground">{t(hint)}</span>
          {/if}
        </span>
      </Button>
    {/each}
  </div>
</Sheet>
