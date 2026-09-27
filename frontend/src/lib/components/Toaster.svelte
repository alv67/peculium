<script lang="ts">
  import { toast, toasts } from '$lib/stores/toast.svelte'
  import { CheckCircle2, XCircle, AlertTriangle, X } from 'lucide-svelte'
  import { t } from '$lib/i18n/index.svelte'

  /**
   * Toast viewport (EPIC D.2): surface-raised cards with a semantic icon
   * color instead of saturated fills. The wrapper is always mounted with
   * `aria-live="polite"` (live regions must exist before content arrives);
   * error toasts additionally carry `role="alert"` so they interrupt.
   *
   * Optional inline action (EPIC K.4c, decision D11): when a toast carries
   * an `action`, a real `<button>` renders between the message and the ✕
   * (keyboard-reachable in DOM order); clicking it runs the handler and
   * dismisses the toast, so the undo window closes the moment the user acts.
   *
   * Z-index: toasts sit at the top of the scale in $lib/components/ui/utils.ts
   * (dropdown 20 / drawer 30 / modal 40 / toast 50).
   */
  const iconFor = {
    success: CheckCircle2,
    warning: AlertTriangle,
    error: XCircle,
  } as const

  const iconColor = {
    success: 'text-positive',
    warning: 'text-warning',
    error: 'text-negative',
  } as const
</script>

<div class="pointer-events-none fixed right-4 top-4 z-50 flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2" aria-live="polite">
  <!-- The item is named `note` (not `t`) so it never shadows the i18n `t()`. -->
  {#each toasts as note (note.id)}
    {@const Icon = iconFor[note.type]}
    <div
      class="pointer-events-auto flex items-start gap-2 rounded-control border border-border bg-surface-raised px-4 py-3 text-sm font-medium text-foreground shadow-raised"
      role={note.type === 'error' ? 'alert' : undefined}
    >
      <Icon class="mt-0.5 h-4 w-4 shrink-0 {iconColor[note.type]}" />
      <span class="min-w-0 flex-1">{note.message}</span>
      {#if note.action}
        <button
          type="button"
          onclick={() => {
            note.action?.onclick()
            toast.dismiss(note.id)
          }}
          class="focus-ring -my-1 shrink-0 rounded-control px-2 py-1 text-sm font-semibold text-accent-text hover:bg-muted"
        >
          {note.action.label}
        </button>
      {/if}
      <button
        type="button"
        onclick={() => toast.dismiss(note.id)}
        aria-label={t('a11y.dismissNotification')}
        class="focus-ring -mr-1 shrink-0 rounded-control p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
      >
        <X class="h-3.5 w-3.5" />
      </button>
    </div>
  {/each}
</div>
